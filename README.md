[![CircleCI](https://circleci.com/gh/BuoyantIO/slow_cooker.svg?style=shield)](https://circleci.com/gh/BuoyantIO/slow_cooker)

# slow_cooker
A load tester for tenderizing your servers.

Most load testers work by sending as much traffic as possible to a
backend. We wanted a different approach, we wanted to be able to test
a service with a predictable load and concurrency level for a long
period of time. Instead of getting a report at the end, we wanted
periodic reports of qps and latency.

# Running it

`go build; ./slow_cooker <url>`

or:

`go run main.go <url>`

# Testing

`go test ./...`

# Flags

| Flag                  | Default   | Description |
|-----------------------|-----------|-------------|
| `-qps`                | 1         | QPS to send to backends per request thread. |
| `-concurrency`        | 1         | Number of goroutines to run, each at the specified QPS level. Measure total QPS as `qps * concurrency`. |
| `-compress`           | `<unset>` | If set, ask for compressed responses. |
| `-data`               | `<none>`  | Include the specified body data in requests. If the data starts with a '@' the remaining value will be treated as a file path to read the body data from, or if the data value is '@-', the body data will be read from stdin. |
| `-hashSampleRate`     | `0.0`     | Sampe Rate for checking request body's hash. Interval in the range of [0.0, 1.0] |
| `-hashValue`          | `<none>`  | fnv-1a hash value to check the request body against |
| `-header`             | `<none>`  | Adds additional headers to each request. Can be specified multiple times. Format is `key: value`. |
| `-host`               | `<none>`  | Overrides the default host header value that's set on each request. |
| `-interval`           | 10s       | How often to report stats to stdout. |
| `-method`             | GET       | Determines which HTTP method to use when making the request. |
| `-metric-addr`        | `<none>`  | Address to use when serving the Prometheus `/metrics` endpoint. No metrics are served if unset. Format is `host:port` or `:port`. |
| `-noLatencySummary`   | `<unset>` | If set, don't print the latency histogram report at the end. |
| `-noreuse`            | `<unset>` | If set, do not reuse connections. Default is to reuse connections. |
| `-reportLatenciesCSV` | `<none>`  | Filename to write CSV latency values. Format of CSV is millisecond buckets with number of requests in each bucket. |
| `-timeout`            | 10s       | Individual request timeout. |
| `-totalRequests`      | `<none>`  | Exit after sending this many requests. |
| `-help`               | `<unset>` | If set, print all available flags and exit. |

# Using multiple Host headers

If you want to send multiple Host headers to a backend, pass a comma separated
list to the host flag. Each request will be selected randomly from the list.

For more complex distributions, you can run multiple slow_cooker processes:

```$ slow_cooker -host web_a,web_b -qps 200 http://localhost:4140```

```$ slow_cooker -host web_b -qps 100 http://localhost:4140```

This example will send 300 qps total to `http://localhost:4140/` with 100 qps
sent with `Host: web_a` and 200 qps sent with `Host: web_b`

# TLS use

Pass in an https url and it'll use TLS automatically.

_Warning_ We do not verify the certificate, we use `InsecureSkipVerify: true`

# Example usage

```
$ ./slow_cooker -qps 100 -concurrency 10 http://slow_server

2016-05-16T20:45:05Z   7102/0/0 10000 71% 10s 0 [ 12  26  37  91 ] 91
2016-05-16T20:45:16Z   7120/0/0 10000 71% 10s 1 [ 11  27  37  53 ] 53
2016-05-16T20:45:26Z   7158/0/0 10000 71% 10s 0 [ 11  27  37  74 ] 74
2016-05-16T20:45:36Z   7169/0/0 10000 71% 10s 1 [ 11  27  36  52 ] 52
2016-05-16T20:45:46Z   7273/0/0 10000 72% 10s 0 [ 11  27  36  58 ] 58
2016-05-16T20:45:56Z   7087/0/0 10000 70% 10s 1 [ 11  28  37  61 ] 61
2016-05-16T20:46:07Z   7231/0/0 10000 72% 10s 0 [ 11  26  35  71 ] 71
2016-05-16T20:46:17Z   7257/0/0 10000 72% 10s 0 [ 11  27  36  57 ] 57
2016-05-16T20:46:27Z   7205/0/0 10000 72% 10s 0 [ 11  27  36  64 ] 64
2016-05-16T20:46:37Z   7256/0/0 10000 72% 10s 0 [ 11  27  36  62 ] 62
2016-05-16T20:46:47Z   7164/0/0 10000 71% 10s 0 [ 11  27  38  74 ] 74
2016-05-16T20:46:58Z   7232/0/0 10000 72% 10s 0 [ 11  26  35  63 ] 63
```

In this example, we see that the server is too slow to keep up with
our requested load. that slowness is noted via the throughput
percentage.

## Docker usage

### Run

```bash
docker run -it buoyantio/slow_cooker -qps 100 -concurrency 10 http://$(docker-machine ip default):4140
```

### Build your own

```bash
docker build -t buoyantio/slow_cooker -f Dockerfile .
```

# Log format

We use vertical alignment in the output to help find anomalies and spot
slowdowns. If you're running multi-hour tests, bumping up the reporting
interval to 60 seconds (`60s` or `1m`) is recommended.

```
$timestamp $good/$bad/$failed $trafficGoal $percentGoal $interval $min [$p50 $p95 $p99 $p999] $max $bhash
```

`bad` means a status code in the 500 range. `failed` means a connection failure.
`percentGoal` is calculated as the total number of `good` and `bad` requests as
a percentage of `trafficGoal`.

`bhash` is the number of failed hashes of body content. A value greater than 0 indicates a real problem.

## Tips and tricks

### keep a logfile

Use `tee` to keep a logfile of slow_cooker results and `cut` to find bad or failed requests.

```bash
./slow_cooker_linux_amd64 -qps 5 -concurrency 20 -interval 10s http://localhost:4140 | tee slow_cooker.log
```

### use cut to look at specific fields from your tee'd logfile

`cat slow_cook.log |cut -d ' ' -f 3 | cut -d '/' -f 2 |sort -rn |uniq -c`

will show all bad (status code >= 500) requests.

`cat slow_cook.log |cut -d ' ' -f 3 | cut -d '/' -f 3 |sort -rn |uniq -c`

will show all failed (connection refused, dropped, etc) requests.

### dig into the full latency report

With the `-reportLatenciesCSV` flag, you can thoroughly inspect your
service's latency instead of relying on pre-computed statistical
summaries. We chose CSV to allow for easy integration with statistical
environments like R and standard spreadsheet tools like Excel.

### use the latency CSV output to see system performance changes

Use `-totalRequests` and `-reportLatenciesCSV` to see how your system
latency grows as a function of traffic levels.

### use -concurrency to improve throughput

If you're not hitting the throughput numbers you expect, try
increasing `-concurrency` so your requests are issued over more
goroutines. Each goroutine issues requests serially, waiting for a
response before issuing the next request.

If you have scripts that process slow_cooker logs, feel free to add
them to this project!
