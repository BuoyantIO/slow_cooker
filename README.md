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

# Flags

`-qps <int>`

Queries per second to send to a backend.

`-concurrency <int>`

How many goroutines to run, each at the specified qps level. Measure
total qps as `qps * concurrency`.

`-host <string>`

Set a `Host:` header.

`-header <header>`

Set header to include in the requests. (Can be repeated.)

Example:

```$ slow_cooker -header 'X-Foo: bar' http://localhost:4140```

`-interval 10s`

How often to report to stdout.

`-noreuse`

Do not reuse connections. (Connection reuse is the default.)

`-compress`

Ask for compressed responses.

`-noLatencySummary`

Don't print the latency histogram report at the end.

`-reportLatenciesCSV <filename>`

Writes a CSV file of latencies. Format is: milliseconds to number of
requests that fall into that bucket.

`-totalRequests <int>`

Exit after sending this many requests.

`-data <string>`

Include the specified body data in requests. If the data starts with a '@' the
remaining value will be treated as a file path to read the data contents from,
or '-' to read from stdin.

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
$timestamp $good/$bad/$failed $trafficGoal $percentGood $interval $min [$p50 $p95 $p99 $p999] $max
```

`bad` means a status code in the 500 range. `failed` means a
connection failure.

## Tips and tricks

### keep a logfile

Use `tee` to keep a logfile of slow_cooker results and `cut` to find bad or failed requests.

```bash
./slow_cooker_linux_amd64 -qps 5 -concurrency 20 -interval 10s -reuse http://localhost:4140 | tee slow_cooker.log
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
