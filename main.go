package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/codahale/hdrhistogram"
)

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
	reuse bool,
	maxConn uint,
) *http.Client {
	tr := http.Transport{
		DisableCompression:  !compress,
		DisableKeepAlives:   !reuse,
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
	received chan *MeasuredResponse,
) {
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintln(os.Stderr, "\n")
	}
	req.Host = *host

	// FIX: find a way to measure latency with the http client.
	start := time.Now()
	response, err := client.Do(req)

	elapsed := time.Since(start)
	if err != nil {
		received <- &MeasuredResponse{0, 0, 0, false, err}
	} else {
		sz, _ := io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
		received <- &MeasuredResponse{uint64(sz), response.StatusCode, elapsed.Nanoseconds(), false, nil}
	}
}

func exUsage(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", path.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(64)
}

// To achieve a target qps, we need to wait this many Nanoseconds
// between actions.
func CalcTimeToWait(qps *int) time.Duration {
	return time.Duration(int(time.Second) / *qps)
}

func main() {
	qps := flag.Int("qps", 1, "QPS to send to backends per request thread")
	concurrency := flag.Uint("concurrency", 1, "Number of request threads")
	host := flag.String("host", "web", "value of Host header to set")
	urldest := flag.String("url", "http://localhost:4140/", "Destination url")
	interval := flag.Duration("interval", 10*time.Second, "reporting interval")
	reuse := flag.Bool("reuse", false, "reuse connections")
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

	dstURL, err := url.Parse(*urldest)
	if err != nil {
		exUsage(fmt.Sprintf("invalid URL: '%s': %s\n", urldest, err.Error()))
	}

	// Repsonse tracking metadata.
	count := uint64(0)
	size := uint64(0)
	good := uint64(0)
	bad := uint64(0)
	failed := uint64(0)
	// from 0 to 1 minute in nanoseconds
	// FIX: verify that these buckets work correctly for our use case.
	hist := hdrhistogram.New(0, 60000000000, 5)
	received := make(chan *MeasuredResponse)
	timeout := time.After(*interval)
	timeToWait := CalcTimeToWait(qps)

	doTLS := dstURL.Scheme == "https"
	client := newClient(*compress, doTLS, *reuse, *concurrency)

	for i := uint(0); i < *concurrency; i++ {
		ticker := time.NewTicker(timeToWait)
		go func() {
			for _ = range ticker.C {
				sendRequest(client, dstURL, host, received)
			}
		}()
	}

	for {
		select {
		case t := <-timeout:
			// Periodically print stats about the request load.
			fmt.Printf("%s %6d/%1d/%1d requests %6d kilobytes %s [%3d %3d %3d %4d ]\n",
				t.Format(time.RFC3339),
				good,
				bad,
				failed,
				(size / 1024),
				interval,
				hist.ValueAtQuantile(50)/1000000,
				hist.ValueAtQuantile(95)/1000000,
				hist.ValueAtQuantile(99)/1000000,
				hist.ValueAtQuantile(999)/1000000)
			count = 0
			size = 0
			good = 0
			bad = 0
			failed = 0
			hist = hdrhistogram.New(0, 60000000000, 5)
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
				hist.RecordValue(managedResp.latency)
			}
		}
	}
}
