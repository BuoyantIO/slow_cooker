package main

import (
	"flag"
	"fmt"
	"github.com/codahale/hdrhistogram"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

type MeasuredResponse struct {
	sz      uint64
	code    int
	latency int64
	timeout bool
}

func sendRequest(url *url.URL, host *string, received chan *MeasuredResponse) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	req.Host = *host

	// FIX: find a way to measure latency with the http client.
	start := time.Now()
	response, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		// FIX: handle errors more gracefully.
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		sz, _ := io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
		received <- &MeasuredResponse{uint64(sz), response.StatusCode, elapsed.Nanoseconds(), false}
	}
}

func exUsage(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n", path.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(64)
}

func main() {
	qps := flag.Int("qps", 1, "QPS to send to backends per request thread")
	concurrency := flag.Uint("concurrency", 1, "Number of request threads")
	host := flag.String("host", "web", "value of Host header to set")
	urldest := flag.String("url", "http://localhost:4140/", "Destination url")
	interval := flag.Duration("interval", 10*time.Second, "reporting interval")

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
		exUsage(fmt.Sprintf("invalid URL: '%s': %s", urldest, err.Error()))
	}

	// Repsonse tracking metadata.
	count := uint64(0)
	size := uint64(0)
	good := uint64(0)
	bad := uint64(0)
	// from 0 to 1 minute in nanoseconds
	// FIX: verify that these buckets work correctly for our use case.
	hist := hdrhistogram.New(0, 60000000000, 5)
	received := make(chan *MeasuredResponse)
	timeout := time.After(*interval)

	timeToWait := time.Millisecond * time.Duration(1000 / *qps)

	for i := uint(0); i < *concurrency; i++ {
		ticker := time.NewTicker(timeToWait)
		go func() {
			for _ = range ticker.C {
				sendRequest(dstURL, host, received)
			}
		}()
	}

	for {
		select {
		case t := <-timeout:
			// Periodically print stats about the request load.
			fmt.Printf("%s %6d/%1d requests %6d kilobytes %s [%3d %3d %3d %3d]\n",
				t.Format(time.RFC3339),
				good,
				bad,
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
			hist = hdrhistogram.New(0, 60000000000, 5)
			timeout = time.After(*interval)
		case managedResp := <-received:
			count++
			size += managedResp.sz
			if managedResp.code >= 200 && managedResp.code < 300 {
				good++
			} else {
				bad++
			}
			hist.RecordValue(managedResp.latency)
		}
	}
}
