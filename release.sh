#!/bin/bash

set -ex

GOOS=linux GOARCH=amd64  go build -o slow_cooker_linux_amd64 github.com/BuoyantIO/slow_cooker
GOOS=linux GOARCH=arm    go build -o slow_cooker_linux_arm   github.com/BuoyantIO/slow_cooker
GOOS=darwin GOARCH=amd64 go build -o slow_cooker_darwin      github.com/BuoyantIO/slow_cooker
echo "releases built:"
ls slow_cooker_*
