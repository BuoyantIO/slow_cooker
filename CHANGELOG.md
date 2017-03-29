# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## Next release
### Added
- Users can now check request bodies against an fnv-1a hash value with a configurable sample rate.
### Changed
- Renamed `good%` column to `goal%`, and started counting bad requests toward goal.

## [1.1.0] - 2017-01-26
### Added
- Added Prometheus `/metrics` endpoint, using new `metric-addr` param

## [1.0.1] - 2017-01-11
### Added
- Added basic CONTRIBUTORS file to help guide PRs
- Added flag to set HTTP request body payload
- Added flag to set HTTP request headers
- Added flag to set HTTP method used for requests

### Changed
- Modified Dockerfile to build project from fully-qualified package location

## [1.0.0] - 2016-12-8
### Added
- Added percent success to the output.
- Added target traffic per interval to the output.
- Optional latency histogram report to stdout.
- Adds a change indicator to the end of the line showing how many
  orders of magnitude this line's p99 is over the previous 5.
- Optional full latency CSV report to a given filename.
- Respect `http_proxy` environment variable.
- Added `-totalRequests` flag for exiting after the given number of requests are issued.
- Added a header line at the beginning of the test run.

### Changed
- We no longer generate the i386 linux binaries for release.
- Removed `-reuse` deprecation warning.
- Removed bytes received from the output.
- Removed `-url`, instead use the first argument from ARGV.
- Use the new `/net/http/httptrace` package and measure latency as time to first byte.
- No longer exit with an error code of 1 after cleanup

## [0.7.0] - 2016-07-21
### Added
- We now output min and max latency numbers bracketing the latency percentiles. (Fixes #13)
- You can now pass a CSV of Host headers and it will fairly split traffic with each Host header.
- Each request has a header called Sc-Req-Id with a unique numeric value to help debug proxy interactions.

### Changed
- Output now shows good/bad/failed requests
- Improved qps calculation when fractional milliseconds are involved. (Fixed #5)
- -reuse is now on by default. If you want to turn reuse off, use -noreuse (Fixes #11)

## [0.6.0] - 2016-05-23
### Changed
- compression turned off by default. re-enable it with `-compress`
- better error reporting by adding a few strategic newlines
- compression, etc settings were not set when client reuse was disabled
- tie maxConns to concurrency to avoid FD exhaustion

### Added
- TLS automatically used if https urls are passed into `-url
- Release script now builds darwin binaries
- Dockerfile
- Marathon config file


## [0.5.0] - 2016-05-18
### Changed
- better output lines using padding rather than tabs

### Added
- reuse connections with the `-reuse` flag
- static binaries available in the Releases page
