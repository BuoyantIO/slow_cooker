# slow_cooker
A load tester for tenderizing your servers.

Most load testers work by sending as much traffic as possible to a
backend. We wanted a different approach, we wanted to be able to test
a service with a predictable load and concurrency level for a long
period of time. Instead of getting a report at the end, we wanted
periodic reports of qps and latency.

# Flags

```-qps 1```

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

# Using multiple Host headers

If you want to send multiple Host headers to a backend, run multiple
`slow_cooker` processes and manually calculate the qps split.

```$ slow_cooker -host web_a -qps 100```

```$ slow_cooker -host web_b -qps 200```

Would send 300 qps total to the default URL (`http://localhost:4140/`)
with 100 qps sent with `Host: web_a` and 200 qps sent with `Host: web_b`

# Example usage

```
$ slow_cooker -qps 200 -concurrency 10
2016-05-06T21:37:29Z     20010/0 requests       11959 kilobytes 10s [2/3/4/11]
2016-05-06T21:37:39Z     19986/0 requests       11944 kilobytes 10s [2/3/5/12]
2016-05-06T21:37:49Z     20012/0 requests       11960 kilobytes 10s [2/3/4/11]
2016-05-06T21:37:59Z     19949/0 requests       11922 kilobytes 10s [2/3/6/13]
2016-05-06T21:38:09Z     19999/0 requests       11952 kilobytes 10s [2/3/4/14]
2016-05-06T21:38:19Z     20009/0 requests       11958 kilobytes 10s [2/3/4/11]
```

# Log format

```
timestamp\t good/bad requests\t size kilobytes\t interval\t[p50/p95/p99/p999] latency
```

