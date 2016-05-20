# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]
### Changed
- compression turned off by default. re-enable it with `-compress`
- better error reporting by adding a few strategic newlines.

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
