package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/codahale/hdrhistogram"
)

// MeasuredResponse holds metadata about the response
// we receive from the server under test.
type MeasuredResponse struct {
	sz      uint64
	code    int
	latency int64
	timeout bool
	err     error
}

func newClient(
	compress bool,
	https bool,
	noreuse bool,
	maxConn uint,
) *http.Client {
	tr := http.Transport{
		DisableCompression:  !compress,
		DisableKeepAlives:   noreuse,
		MaxIdleConnsPerHost: int(maxConn),
	}
	if https {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{Transport: &tr}
}

func sendRequest(
	client *http.Client,
	url *url.URL,
	host *string,
	reqID uint64,
	received chan *MeasuredResponse,
	bodyBuffer []byte,
) {
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintf(os.Stderr, "\n")
	}
	req.Host = *host
	req.Header.Add("Sc-Req-Id", strconv.FormatUint(reqID, 10))

	// FIX: find a way to measure latency with the http client.
	start := time.Now()
	response, err := client.Do(req)

	if err != nil {
		received <- &MeasuredResponse{0, 0, 0, false, err}
	} else {
		if sz, err := io.CopyBuffer(ioutil.Discard, response.Body, bodyBuffer); err == nil {
			response.Body.Close()
			elapsed := time.Since(start)
			received <- &MeasuredResponse{uint64(sz), response.StatusCode, elapsed.Nanoseconds(), false, nil}
		} else {
			received <- &MeasuredResponse{0, 0, 0, false, err}
		}
	}
}

func exUsage(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", path.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(64)
}

// CalcTimeToWait calculates how many Nanoseconds to wait between actions.
func CalcTimeToWait(qps *int) time.Duration {
	return time.Duration(int(time.Second) / *qps)
}

var reqID = uint64(0)

var shouldFinish = false
var shouldFinishLock sync.RWMutex

// Signals the system to stop sending traffic and clean up after itself.
func finishSendingTraffic() {
	shouldFinishLock.Lock()
	shouldFinish = true
	shouldFinishLock.Unlock()
}

func main() {
	qps := flag.Int("qps", 1, "QPS to send to backends per request thread")
	concurrency := flag.Uint("concurrency", 1, "Number of request threads")
	host := flag.String("host", "web", "value of Host header to set")
	urldest := flag.String("url", "http://localhost:4140/", "Destination url")
	interval := flag.Duration("interval", 10*time.Second, "reporting interval")
	// FIX: remove this flag before open source release.
	reuse := flag.Bool("reuse", true, "reuse connections. (deprecated: no need to set)")
	noreuse := flag.Bool("noreuse", false, "don't reuse connections")
	compress := flag.Bool("compress", false, "use compression")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.Parse()

	if *qps < 1 {
		exUsage("qps must be at least 1")
	}

	if *concurrency < 1 {
		exUsage("concurrency must be at least 1")
	}

	if *reuse {
		fmt.Printf("-reuse has been deprecated. Connection reuse is now the default\n")
	}

	hosts := strings.Split(*host, ",")

	dstURL, err := url.Parse(*urldest)
	if err != nil {
		exUsage(fmt.Sprintf("invalid URL: '%s': %s\n", *urldest, err.Error()))
	}

	// Repsonse tracking metadata.
	count := uint64(0)
	size := uint64(0)
	good := uint64(0)
	bad := uint64(0)
	failed := uint64(0)
	min := int64(math.MaxInt64)
	max := int64(0)
	// from 0 to 1 minute in nanoseconds
	// FIX: verify that these buckets work correctly for our use case.
	hist := hdrhistogram.New(0, 60000000000, 5)
	received := make(chan *MeasuredResponse)
	timeout := time.After(*interval)
	timeToWait := CalcTimeToWait(qps)

	doTLS := dstURL.Scheme == "https"
	client := newClient(*compress, doTLS, *noreuse, *concurrency)
	var sendTraffic sync.WaitGroup

	for i := uint(0); i < *concurrency; i++ {
		ticker := time.NewTicker(timeToWait)
		go func() {
			// For each goroutine we want to reuse a buffer for performance reasons.
			bodyBuffer := make([]byte, 50000)
			sendTraffic.Add(1)
			for _ = range ticker.C {
				shouldFinishLock.RLock()
				if !shouldFinish {
					shouldFinishLock.RUnlock()
					sendRequest(client, dstURL, &hosts[rand.Intn(len(hosts))], atomic.AddUint64(&reqID, 1), received, bodyBuffer)
				} else {
					shouldFinishLock.RUnlock()
					sendTraffic.Done()
					return
				}
			}
		}()
	}

	cleanup := make(chan os.Signal)
	signal.Notify(cleanup, syscall.SIGINT)

	for {
		select {
		case <-cleanup:
			finishSendingTraffic()
			go func() {
				// Don't Wait() in the event loop or else we'll block the workers
				// from draining.
				sendTraffic.Wait()
				os.Exit(1)
			}()
		case t := <-timeout:
			// When all requests are failures, ensure we don't accidentally
			// print out a monstrously huge number.
			if min == math.MaxInt64 {
				min = 0
			}
			// Periodically print stats about the request load.
			fmt.Printf("%s %6d/%1d/%1d requests %6d kilobytes %s %3d [%3d %3d %3d %4d ] %4d\n",
				t.Format(time.RFC3339),
				good,
				bad,
				failed,
				(size / 1024),
				interval,
				min/1000000,
				hist.ValueAtQuantile(50)/1000000,
				hist.ValueAtQuantile(95)/1000000,
				hist.ValueAtQuantile(99)/1000000,
				hist.ValueAtQuantile(999)/1000000,
				max/1000000)
			count = 0
			size = 0
			good = 0
			bad = 0
			min = math.MaxInt64
			max = 0
			failed = 0
			hist.Reset()
			timeout = time.After(*interval)
		case managedResp := <-received:
			count++
			if managedResp.err != nil {
				fmt.Fprintln(os.Stderr, managedResp.err)
				failed++
			} else {
				size += managedResp.sz
				if managedResp.code >= 200 && managedResp.code < 500 {
					good++
				} else {
					bad++
				}

				if managedResp.latency < min {
					min = managedResp.latency
				}

				if managedResp.latency > max {
					max = managedResp.latency
				}

				hist.RecordValue(managedResp.latency)
			}
		}
	}
}
