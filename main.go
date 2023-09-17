package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
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

	eurekaurlsprovider "github.com/buoyantio/slow_cooker/eurekaUrlsProvider"
	"github.com/buoyantio/slow_cooker/hdrreport"
	"github.com/buoyantio/slow_cooker/ring"
	"github.com/buoyantio/slow_cooker/window"
	"github.com/codahale/hdrhistogram"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MeasuredResponse holds metadata about the response
// we receive from the server under test.
type MeasuredResponse struct {
	sz              uint64
	code            int
	latency         time.Duration
	timeout         bool
	failedHashCheck bool
	err             error
}

func newClient(
	compress bool,
	noreuse bool,
	maxConn int,
	timeout time.Duration,
) *http.Client {
	tr := http.Transport{
		DisableCompression:  !compress,
		DisableKeepAlives:   noreuse,
		MaxIdleConnsPerHost: maxConn,
		Proxy:               http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: &tr,
	}
}

func sendRequest(
	client *http.Client,
	method string,
	url *url.URL,
	host string,
	headers headerSet,
	requestData []byte,
	reqID uint64,
	noreuse bool,
	hashValue uint64,
	checkHash bool,
	hasher hash.Hash64,
	received chan *MeasuredResponse,
	bodyBuffer []byte,
) {
	req, err := http.NewRequest(method, url.String(), bytes.NewBuffer(requestData))
	req.Close = noreuse
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintf(os.Stderr, "\n")
	}
	if host != "" {
		req.Host = host
	}
	req.Header.Add("Sc-Req-Id", strconv.FormatUint(reqID, 10))
	for k, v := range headers {
		req.Header.Add(k, v)
	}

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
		received <- &MeasuredResponse{err: err}
	} else {
		defer response.Body.Close()
		if !checkHash {
			if sz, err := io.CopyBuffer(ioutil.Discard, response.Body, bodyBuffer); err == nil {

				received <- &MeasuredResponse{
					sz:      uint64(sz),
					code:    response.StatusCode,
					latency: elapsed}
			} else {
				received <- &MeasuredResponse{err: err}
			}
		} else {
			if bytes, err := ioutil.ReadAll(response.Body); err != nil {
				received <- &MeasuredResponse{err: err}
			} else {
				hasher.Write(bytes)
				sum := hasher.Sum64()
				failedHashCheck := false
				if hashValue != sum {
					failedHashCheck = true
				}
				received <- &MeasuredResponse{
					sz:              uint64(len(bytes)),
					code:            response.StatusCode,
					latency:         elapsed,
					failedHashCheck: failedHashCheck}
			}
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

type headerSet map[string]string

func (h *headerSet) String() string {
	return ""
}

func (h *headerSet) Set(s string) error {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) < 2 || len(parts[0]) == 0 {
		return fmt.Errorf("Header invalid")
	}
	name := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	(*h)[name] = value
	return nil
}

func loadData(data string) []byte {
	var file *os.File
	var requestData []byte
	var err error
	if strings.HasPrefix(data, "@") {
		path := data[1:]
		if path == "-" {
			file = os.Stdin
		} else {
			file, err = os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}
			defer file.Close()
		}

		requestData, err = ioutil.ReadAll(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
	} else {
		requestData = []byte(data)
	}

	return requestData
}

func loadURLs(urldest string) []*url.URL {
	var urls []*url.URL
	var err error
	var scanner *bufio.Scanner

	if strings.HasPrefix(urldest, "@") {
		var file *os.File
		path := urldest[1:]
		if path == "-" {
			file = os.Stdin
		} else {
			file, err = os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}
			defer file.Close()
		}
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(strings.NewReader(urldest))
	}

	for i := 1; scanner.Scan(); i++ {
		line := scanner.Text()
		URL, err := url.Parse(line)
		if err != nil {
			exUsage("invalid URL on line %d: '%s': %s\n", i, line, err.Error())
		} else if URL.Scheme == "" {
			exUsage("invalid URL on line %d: '%s': Missing scheme\n", i, line)
		} else if URL.Host == "" {
			exUsage("invalid URL on line %d: '%s': Missing host\n", i, line)
		}
		urls = append(urls, URL)
	}

	return urls
}

var (
	promRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "requests",
		Help: "Number of requests",
	})

	promSuccesses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "successes",
		Help: "Number of successful requests",
	})

	promLatencyMSHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "latency_ms",
		Help: "RPC latency distributions in milliseconds.",
		// 50 exponential buckets ranging from 0.5 ms to 3 minutes
		// TODO: make this tunable
		Buckets: prometheus.ExponentialBuckets(0.5, 1.3, 50),
	})
	promLatencyUSHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "latency_us",
		Help: "RPC latency distributions in microseconds.",
		// 50 exponential buckets ranging from 1 us to 2.4 seconds
		// TODO: make this tunable
		Buckets: prometheus.ExponentialBuckets(1, 1.35, 50),
	})
	promLatencyNSHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "latency_ns",
		Help: "RPC latency distributions in nanoseconds.",
		// 50 exponential buckets ranging from 1 ns to 0.4 seconds
		// TODO: make this tunable
		Buckets: prometheus.ExponentialBuckets(1, 1.5, 50),
	})
)

func registerMetrics() {
	prometheus.MustRegister(promRequests)
	prometheus.MustRegister(promSuccesses)
	prometheus.MustRegister(promLatencyMSHistogram)
	prometheus.MustRegister(promLatencyUSHistogram)
	prometheus.MustRegister(promLatencyNSHistogram)
}

// Sample Rate is between [0.0, 1.0] and determines what percentage of request bodies
// should be checked that their hash matches a known hash.
func shouldCheckHash(sampleRate float64) bool {
	return rand.Float64() < sampleRate
}

func main() {
	qps := flag.Int("qps", 1, "QPS to send to backends per request thread")
	concurrency := flag.Int("concurrency", 1, "Number of request threads")
	numIterations := flag.Uint64("iterations", 0, "Number of iterations (0 for infinite)")
	host := flag.String("host", "", "value of Host header to set")
	method := flag.String("method", "GET", "HTTP method to use")
	interval := flag.Duration("interval", 10*time.Second, "reporting interval")
	noreuse := flag.Bool("noreuse", false, "don't reuse connections")
	compress := flag.Bool("compress", false, "use compression")
	clientTimeout := flag.Duration("timeout", 10*time.Second, "individual request timeout")
	noLatencySummary := flag.Bool("noLatencySummary", false, "suppress the final latency summary")
	reportLatenciesCSV := flag.String("reportLatenciesCSV", "",
		"filename to output hdrhistogram latencies in CSV")
	latencyUnit := flag.String("latencyUnit", "ms", "latency units [ms|us|ns]")
	help := flag.Bool("help", false, "show help message")
	totalRequests := flag.Uint64("totalRequests", 0, "total number of requests to send before exiting")
	headers := make(headerSet)
	flag.Var(&headers, "header", "HTTP request header. (can be repeated.)")
	data := flag.String("data", "", "HTTP request data")
	metricAddr := flag.String("metric-addr", "", "address to serve metrics on")
	hashValue := flag.Uint64("hashValue", 0, "fnv-1a hash value to check the request body against")
	hashSampleRate := flag.Float64("hashSampleRate", 0.0, "Sampe Rate for checking request body's hash. Interval in the range of [0.0, 1.0]")
	useEureka := flag.Bool("useEureka", false, "Eureka will be used for getting urls list by a specific service")
	eurekaService := flag.String("eurekaService", "", "Specify service from Eureka's list for testing. % may be used as wildcard")

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

	var dstURLs []*url.URL
	if *useEureka {
		dstURLs = eurekaurlsprovider.LoadEurekaURLs(urldest, *eurekaService)
	} else {
		dstURLs = loadURLs(urldest)
	}

	if *qps < 1 {
		exUsage("qps must be at least 1")
	}

	if *concurrency < 1 {
		exUsage("concurrency must be at least 1")
	}

	latencyDur := time.Millisecond
	if *latencyUnit == "ms" {
		latencyDur = time.Millisecond
	} else if *latencyUnit == "us" {
		latencyDur = time.Microsecond
	} else if *latencyUnit == "ns" {
		latencyDur = time.Nanosecond
	} else {
		exUsage("latency unit should be [ms | us | ns].")
	}
	latencyDurNS := latencyDur.Nanoseconds()
	msInNS := time.Millisecond.Nanoseconds()
	usInNS := time.Microsecond.Nanoseconds()

	hosts := strings.Split(*host, ",")

	requestData := loadData(*data)

	iteration := uint64(0)

	// Response tracking metadata.
	count := uint64(0)
	size := uint64(0)
	good := uint64(0)
	bad := uint64(0)
	failed := uint64(0)
	min := int64(math.MaxInt64)
	max := int64(0)
	failedHashCheck := int64(0)

	// dayInTimeUnits represents the number of time units (ms, us, or ns) in a 24-hour day.
	dayInTimeUnits := int64(24 * time.Hour / latencyDur)

	hist := hdrhistogram.New(0, dayInTimeUnits, 3)
	globalHist := hdrhistogram.New(0, dayInTimeUnits, 3)
	latencyHistory := ring.New(5)
	received := make(chan *MeasuredResponse)
	timeout := time.After(*interval)
	timeToWait := CalcTimeToWait(qps)
	var totalTrafficTarget int
	totalTrafficTarget = *qps * *concurrency * int(interval.Seconds())

	client := newClient(*compress, *noreuse, *concurrency, *clientTimeout)
	var sendTraffic sync.WaitGroup
	// The time portion of the header can change due to timezone.
	timeLen := len(time.Now().Format(time.RFC3339))
	timePadding := strings.Repeat(" ", timeLen-len("# "))
	intLen := len(fmt.Sprintf("%s", *interval))
	intPadding := strings.Repeat(" ", intLen-2)

	if len(dstURLs) == 1 {
		fmt.Printf("# sending %d %s req/s with concurrency=%d to %s ...\n", (*qps * *concurrency), *method, *concurrency, dstURLs[0])
	} else {
		fmt.Printf("# sending %d %s req/s with concurrency=%d using url list %s ...\n", (*qps * *concurrency), *method, *concurrency, urldest[1:])
	}

	fmt.Printf("# %s iter   good/b/f t   goal%% %s min [p50 p95 p99  p999]  max bhash change\n", timePadding, intPadding)

	callTimes := make([]int, len(dstURLs))
	for i := 0; i < *concurrency; i++ {
		ticker := time.NewTicker(timeToWait)
		go func(offset int) {
			y := offset
			// For each goroutine we want to reuse a buffer for performance reasons.
			bodyBuffer := make([]byte, 50000)
			sendTraffic.Add(1)
			for _ = range ticker.C {
				var checkHash bool
				hasher := fnv.New64a()
				if *hashSampleRate > 0.0 {
					checkHash = shouldCheckHash(*hashSampleRate)
				} else {
					checkHash = false
				}
				shouldFinishLock.RLock()
				if !shouldFinish {
					shouldFinishLock.RUnlock()
					callTimes[y]++
					sendRequest(client, *method, dstURLs[y], hosts[rand.Intn(len(hosts))], headers, requestData, atomic.AddUint64(&reqID, 1), *noreuse, *hashValue, checkHash, hasher, received, bodyBuffer)
				} else {
					shouldFinishLock.RUnlock()
					sendTraffic.Done()
					return
				}
				y++
				if y >= len(dstURLs) {
					y = 0
				}
			}
		}(i % len(dstURLs))
	}

	cleanup := make(chan bool, 3)
	interrupted := make(chan os.Signal, 2)
	signal.Notify(interrupted, syscall.SIGINT)

	if *metricAddr != "" {
		registerMetrics()
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			http.ListenAndServe(*metricAddr, nil)
		}()
	}

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
				fmt.Println(callTimes)
				os.Exit(0)
			}()
		case t := <-timeout:
			// When all requests are failures, ensure we don't accidentally
			// print out a monstrously huge number.
			if min == math.MaxInt64 {
				min = 0
			}
			// Periodically print stats about the request load.
			percentAchieved := int(math.Min((((float64(good) + float64(bad)) /
				float64(totalTrafficTarget)) * 100), 100))

			lastP99 := int(hist.ValueAtQuantile(99))
			// We want the change indicator to be based on
			// how far away the current value is from what
			// we've seen historically. This is why we call
			// CalculateChangeIndicator() first and then Push()
			changeIndicator := window.CalculateChangeIndicator(latencyHistory.Items, lastP99)
			latencyHistory.Push(lastP99)

			fmt.Printf("%s %4d %6d/%1d/%1d %d %3d%% %s %3d [%3d %3d %3d %4d ] %4d %6d %s\n",
				t.Format(time.RFC3339),
				iteration,
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
				failedHashCheck,
				changeIndicator)

			iteration++

			if *numIterations > 0 && iteration >= *numIterations {
				cleanup <- true
			}
			count = 0
			size = 0
			good = 0
			bad = 0
			min = math.MaxInt64
			max = 0
			failed = 0
			failedHashCheck = 0
			hist.Reset()
			timeout = time.After(*interval)

			if *totalRequests != 0 && reqID > *totalRequests {
				cleanup <- true
			}
		case managedResp := <-received:
			count++
			promRequests.Inc()
			if managedResp.err != nil {
				fmt.Fprintln(os.Stderr, managedResp.err)
				failed++
			} else {
				respLatencyNS := managedResp.latency.Nanoseconds()
				latency := respLatencyNS / latencyDurNS

				size += managedResp.sz
				if managedResp.failedHashCheck {
					failedHashCheck++
				}
				if managedResp.code >= 200 && managedResp.code < 500 {
					good++
					promSuccesses.Inc()
					promLatencyMSHistogram.Observe(float64(respLatencyNS / msInNS))
					promLatencyUSHistogram.Observe(float64(respLatencyNS / usInNS))
					promLatencyNSHistogram.Observe(float64(respLatencyNS))
				} else {
					bad++
				}

				if latency < min {
					min = latency
				}

				if latency > max {
					max = latency
				}

				hist.RecordValue(latency)
				globalHist.RecordValue(latency)
			}
		}
	}
}
