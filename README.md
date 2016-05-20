# slow_cooker
A load tester for tenderizing your servers.

Most load testers work by sending as much traffic as possible to a
backend. We wanted a different approach, we wanted to be able to test
a service with a predictable load and concurrency level for a long
period of time. Instead of getting a report at the end, we wanted
periodic reports of qps and latency.

# Flags

`-qps 1`

Queries per second to send to a backend.

`-concurrency 1`

How many goroutines to run, each at the specified qps level. Measure
total qps as `qps * concurrency`.

`-host web`

Set a `Host:` header. By default, we'll send `Host: web` for
Buoyant-specific reasons.

`-interval 10s`

How often to report to stdout.

`-url http://localhost:4140/`

The url to send backend traffic to.

`-reuse`

Reuse connections and reuse a single thread-safe http client.

`-compress`

Ask for compressed responses.

# Using multiple Host headers

If you want to send multiple Host headers to a backend, run multiple
`slow_cooker` processes and manually calculate the qps split.

```$ slow_cooker -host web_a -qps 100```

```$ slow_cooker -host web_b -qps 200```

This command will send 300 qps total to the default URL
(`http://localhost:4140/`) with 100 qps sent with `Host: web_a` and
200 qps sent with `Host: web_b`

# TLS use

Pass in an https url with the `-url` flag and it'll use TLS.

_Warning_ We do not verify the certificate, we use `InsecureSkipVerify: true`

# Example usage

```
$ ./slow_cooker -qps 100 -concurrency 10
2016-05-16T20:45:05Z   7102/0 requests   4244 kilobytes 10s [ 12  26  37  91 ]
2016-05-16T20:45:16Z   7120/0 requests   4255 kilobytes 10s [ 11  27  37  53 ]
2016-05-16T20:45:26Z   7158/0 requests   4278 kilobytes 10s [ 11  27  37  74 ]
2016-05-16T20:45:36Z   7169/0 requests   4284 kilobytes 10s [ 11  27  36  52 ]
2016-05-16T20:45:46Z   7273/0 requests   4346 kilobytes 10s [ 11  27  36  58 ]
2016-05-16T20:45:56Z   7087/0 requests   4235 kilobytes 10s [ 11  28  37  61 ]
2016-05-16T20:46:07Z   7231/0 requests   4321 kilobytes 10s [ 11  26  35  71 ]
2016-05-16T20:46:17Z   7257/0 requests   4337 kilobytes 10s [ 11  27  36  57 ]
2016-05-16T20:46:27Z   7205/0 requests   4306 kilobytes 10s [ 11  27  36  64 ]
2016-05-16T20:46:37Z   7256/0 requests   4336 kilobytes 10s [ 11  27  36  62 ]
2016-05-16T20:46:47Z   7164/0 requests   4281 kilobytes 10s [ 11  27  38  74 ]
2016-05-16T20:46:58Z   7232/0 requests   4322 kilobytes 10s [ 11  26  35  63 ]
```

## Docker usage

### Run

```bash
docker run -it buoyantio/slow_cooker -url http://$(docker-machine ip default):4140 -qps 100 -concurrency 10
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
$timestamp $good/$bad requests $size kilobytes $interval [$p50 $p95 $p99 $p999]
```

# TODO
 * Instrument the http client rather than just measuring around the function call.
 * Evaluate whether bytes returned is valuable enough for default output.
 * Test that the HDR buckets are set appropriately for an http backend.
