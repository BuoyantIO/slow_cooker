package main

import (
	"testing"
	"time"
)

func TestQpsCalc(t *testing.T) {
	// At 100 qps, we expect to wait 10 milliseconds
	checkDuration(100, 10, t)
	// At 1000 qps, we expect to wait 1 milliseconds
	checkDuration(1000, 1, t)
	// At 150 qps, we expect to wait 6.666 milliseconds
	checkDuration(150, 6.666666, t)
	// At 134 qps, we expect to wait 7.462 milliseconds
	checkDuration(134, 7.462686, t)
}

func checkDuration(targetQPS int, expectedWaitTimeMs float64, t *testing.T) {
	expected := time.Duration(expectedWaitTimeMs * float64(time.Millisecond))
	got := CalcTimeToWait(&targetQPS)
	if expected != got {
		t.Errorf("For %d qps, expected to wait %s, instead we wait %s",
			targetQPS, expected, got)
	}
}
