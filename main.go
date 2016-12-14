package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptrace"
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

	"github.com/buoyantio/slow_cooker/hdrreport"
	"github.com/buoyantio/slow_cooker/ring"
	"github.com/buoyantio/slow_cooker/window"
	"github.com/codahale/hdrhistogram"
)

// DayInMs 1 day in milliseconds
const DayInMs int64 = 24 * 60 * 60 * 1000000

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
	maxConn int,
) *http.Client {
	tr := http.Transport{
		DisableCompression:  !compress,
		DisableKeepAlives:   noreuse,
		MaxIdleConnsPerHost: maxConn,
		Proxy:               http.ProxyFromEnvironment,
	}
	if https {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{Transport: &tr}
}

func sendRequest(
	client *http.Client,
	method string,
	url *url.URL,
	host string,
	reqID uint64,
	received chan *MeasuredResponse,
	bodyBuffer []byte,
) {
	req, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintf(os.Stderr, "\n")
	}
	if host != "" {
		req.Host = host
	}
	req.Header.Add("Sc-Req-Id", strconv.FormatUint(reqID, 10))

	var elapsed time.Duration
	start := time.Now()

	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			elapsed = time.Since(start)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	response, err := client.Do(req)

	if err != nil {
		received <- &MeasuredResponse{0, 0, 0, false, err}
	} else {
		if sz, err := io.CopyBuffer(ioutil.Discard, response.Body, bodyBuffer); err == nil {
			response.Body.Close()
			received <- &MeasuredResponse{
				uint64(sz),
				response.StatusCode,
				elapsed.Nanoseconds() / 1000000,
				false,
				nil}
		} else {
			received <- &MeasuredResponse{0, 0, 0, false, err}
		}
	}
}

func exUsage(msg string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(msg, args...))
	fmt.Fprintln(os.Stderr, "Try --help for help.")
	os.Exit(64)
}

// CalcTimeToWait calculates how many Nanoseconds to wait between actions.
func CalcTimeToWait(qps *int) time.Duration {
	return time.Duration(int(time.Second) / *qps)
}

var reqID = uint64(0)

var shouldFinish = false
var shouldFinishLock sync.RWMutex

// finishSendingTraffic signals the system to stop sending traffic and clean up after itself.
func finishSendingTraffic() {
	shouldFinishLock.Lock()
	shouldFinish = true
	shouldFinishLock.Unlock()
}

func main() {
	qps := flag.Int("qps", 1, "QPS to send to backends per request thread")
	concurrency := flag.Int("concurrency", 1, "Number of request threads")
	host := flag.String("host", "", "value of Host header to set")
	method := flag.String("method", "GET", "HTTP method to use")
	interval := flag.Duration("interval", 10*time.Second, "reporting interval")
	noreuse := flag.Bool("noreuse", false, "don't reuse connections")
	compress := flag.Bool("compress", false, "use compression")
	noLatencySummary := flag.Bool("noLatencySummary", false, "suppress the final latency summary")
	reportLatenciesCSV := flag.String("reportLatenciesCSV", "",
		"filename to output hdrhistogram latencies in CSV")
	help := flag.Bool("help", false, "show help message")
	totalRequests := flag.Uint64("totalRequests", 0, "total number of requests to send before exiting")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <url> [flags]\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(64)
	}

	if flag.NArg() != 1 {
		exUsage("Expecting one argument: the target url to test, e.g. http://localhost:4140/")
	}

	urldest := flag.Arg(0)
	dstURL, err := url.Parse(urldest)
	if err != nil {
		exUsage("invalid URL: '%s': %s\n", urldest, err.Error())
	}

	if *qps < 1 {
		exUsage("qps must be at least 1")
	}

	if *concurrency < 1 {
		exUsage("concurrency must be at least 1")
	}

	hosts := strings.Split(*host, ",")

	// Repsonse tracking metadata.
	count := uint64(0)
	size := uint64(0)
	good := uint64(0)
	bad := uint64(0)
	failed := uint64(0)
	min := int64(math.MaxInt64)
	max := int64(0)

	hist := hdrhistogram.New(0, DayInMs, 3)
	globalHist := hdrhistogram.New(0, DayInMs, 3)
	latencyHistory := ring.New(5)
	received := make(chan *MeasuredResponse)
	timeout := time.After(*interval)
	timeToWait := CalcTimeToWait(qps)
	var totalTrafficTarget int
	totalTrafficTarget = *qps * *concurrency * int(interval.Seconds())

	doTLS := dstURL.Scheme == "https"
	client := newClient(*compress, doTLS, *noreuse, *concurrency)
	var sendTraffic sync.WaitGroup
	// The time portion of the header can change due to timezone.
	timeLen := len(time.Now().Format(time.RFC3339))
	timePadding := strings.Repeat(" ", timeLen)
	intLen := len(fmt.Sprintf("%s", *interval))
	intPadding := strings.Repeat(" ", intLen-2)

	fmt.Printf("# sending %d %s req/s with concurrency=%d to %s ...\n", (*qps * *concurrency), *method, *concurrency, dstURL)
	fmt.Printf("# %s good/b/f t   good%% %s min [p50 p95 p99  p999]  max change\n", timePadding, intPadding)
	for i := 0; i < *concurrency; i++ {
		ticker := time.NewTicker(timeToWait)
		go func() {
			// For each goroutine we want to reuse a buffer for performance reasons.
			bodyBuffer := make([]byte, 50000)
			sendTraffic.Add(1)
			for _ = range ticker.C {
				shouldFinishLock.RLock()
				if !shouldFinish {
					shouldFinishLock.RUnlock()
					sendRequest(client, *method, dstURL, hosts[rand.Intn(len(hosts))], atomic.AddUint64(&reqID, 1), received, bodyBuffer)
				} else {
					shouldFinishLock.RUnlock()
					sendTraffic.Done()
					return
				}
			}
		}()
	}

	cleanup := make(chan bool, 2)
	interrupted := make(chan os.Signal, 2)
	signal.Notify(interrupted, syscall.SIGINT)

	for {
		select {
		// If we get a SIGINT, then start the shutdown process.
		case <-interrupted:
			cleanup <- true
		case <-cleanup:
			finishSendingTraffic()
			if !*noLatencySummary {
				hdrreport.PrintLatencySummary(globalHist)
			}
			if *reportLatenciesCSV != "" {
				err := hdrreport.WriteReportCSV(reportLatenciesCSV, globalHist)
				if err != nil {
					log.Panicf("Unable to write Latency CSV file: %v\n", err)
				}
			}
			go func() {
				// Don't Wait() in the event loop or else we'll block the workers
				// from draining.
				sendTraffic.Wait()
				os.Exit(0)
			}()
		case t := <-timeout:
			// When all requests are failures, ensure we don't accidentally
			// print out a monstrously huge number.
			if min == math.MaxInt64 {
				min = 0
			}
			// Periodically print stats about the request load.
			percentAchieved := int(math.Min(((float64(good) /
				float64(totalTrafficTarget)) * 100), 100))

			lastP99 := int(hist.ValueAtQuantile(99))
			// We want the change indicator to be based on
			// how far away the current value is from what
			// we've seen historically. This is why we call
			// CalculateChangeIndicator() first and then Push()
			changeIndicator := window.CalculateChangeIndicator(latencyHistory.Items, lastP99)
			latencyHistory.Push(lastP99)

			fmt.Printf("%s %6d/%1d/%1d %d %3d%% %s %3d [%3d %3d %3d %4d ] %4d %s\n",
				t.Format(time.RFC3339),
				good,
				bad,
				failed,
				totalTrafficTarget,
				percentAchieved,
				interval,
				min,
				hist.ValueAtQuantile(50),
				hist.ValueAtQuantile(95),
				hist.ValueAtQuantile(99),
				hist.ValueAtQuantile(999),
				max,
				changeIndicator)

			count = 0
			size = 0
			good = 0
			bad = 0
			min = math.MaxInt64
			max = 0
			failed = 0
			hist.Reset()
			timeout = time.After(*interval)

			if *totalRequests != 0 && reqID > *totalRequests {
				cleanup <- true
			}
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
				globalHist.RecordValue(managedResp.latency)
			}
		}
	}
}
