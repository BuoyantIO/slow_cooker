package main

import "testing"

func TestHashSampling(t *testing.T) {
	// With a samplingRate of 0.0 we never check.
	checkDriver(0.0, 100000, 0, t)
	// With a samplingRate of 0.01 we check 1% of the values
	checkDriver(0.01, 100000, 1000, t)
	// With a samplingRate of 0.1 we check 10% of the values
	checkDriver(0.1, 100000, 10000, t)
	// With a samplingRate of 0.2 we check 20% of the values
	checkDriver(0.2, 100000, 20000, t)
	// With a samplingRate of 0.5 we check 50% of the values
	checkDriver(0.5, 100000, 50000, t)
	// With a samplingRate of 0.9 we check 90% of the values
	checkDriver(0.9, 100000, 90000, t)
	// With a samplingRate of 0.999 we check 99.9% of the values
	checkDriver(0.999, 100000, 99900, t)
	// With a samplingRate of 1.0 we check 100% of the values
	checkDriver(1.0, 100000, 100000, t)
	// Somebody giving us a high sampleRate will still get 100% of the values
	checkDriver(1.1, 100000, 100000, t)
}

func checkDriver(sampleRate float64, iterations uint64, expectedChecks uint64, t *testing.T) {
	actuallyChecked, _ := shouldCheckHashTest(sampleRate, iterations, t)
	if !deltaCheck(actuallyChecked, expectedChecks, 10.0, t) {
		t.Errorf("Sample Rate of %f should have resulted in no greater than %d checks, instead we saw %d checks",
			sampleRate, expectedChecks, actuallyChecked)
	}
}

func shouldCheckHashTest(samplingRate float64, iterations uint64, t *testing.T) (uint64, uint64) {
	checked := uint64(0)
	unchecked := uint64(0)
	for i := uint64(0); i < iterations; i++ {
		if shouldCheckHash(samplingRate) {
			checked++
		} else {
			unchecked++
		}
	}

	return checked, unchecked
}

// Unit tests for the deltaCheck helper function.
func TestDeltaCheck(t *testing.T) {
	if !deltaCheck(80, 100, 20.0, t) {
		t.Errorf("80 is within 20%% of 100")
	}

	if !deltaCheck(91, 100, 10.0, t) {
		t.Errorf("91 is within 10%% of 100")
	}

	if !deltaCheck(100, 100, 100.0, t) {
		t.Errorf("100 is 100, for reals")
	}
}

/// Checks that an expected value and an actual value are within deltaPercentage percent of each other.
/// deltaPercentage must be a float between (0.0, 100.0]
/// Returns true if the delta check succeeds.
func deltaCheck(actualValue uint64, expectedValue uint64, deltaPercentage float64, t *testing.T) bool {
	if deltaPercentage <= 0.0 {
		t.Fatal("deltaPercentage cannot be 0.0 or negative")
	}

	if deltaPercentage > 100.0 {
		t.Fatal("deltaPercentage cannot be greater than 100.0.")
	}

	delta := uint64(float64(expectedValue) * (0.01 * deltaPercentage))
	top := delta + expectedValue
	bottom := expectedValue - delta

	if actualValue >= bottom && actualValue <= top {
		return true
	}

	return false
}
