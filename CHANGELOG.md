# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]
### Added
- Added percent success to the output.
- Added target traffic per interval to the output.
### Changed
- Removed `-reuse` deprecation warning.
- Removed bytes received from the output.

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
