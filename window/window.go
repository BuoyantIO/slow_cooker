package window

import (
	"math"
)

// Mean returns the mean of a slice of int64.
func Mean(data []int) int {
	sum := 0

	for _, n := range data {
		sum += n
	}

	count := len(data)
	if count > 0 {
		return sum / count
	}
	return 0

}

// CalculateChangeIndicator determines if a Change
// Indicator should be generated from a window of recent latencies.
//
// For each 10x over the mean the latest item is, we add a single plus
// sign up to 3.
//
// For each 10x under the mean the latest item is, we add a single
// minus sign up to 3.
//
// Otherwise we return no change indicator.
func CalculateChangeIndicator(data []int, latest int) string {
	mean := Mean(data)

	if len(data) > 0 {
		// Log10 doesn't play well with 0 so we do some
		// special casing.
		if mean == 0 && latest == 0 {
			return ""
		}

		if mean == 0 && latest > 0 {
			return "+"
		}
		if latest == 0 && mean > 0 {
			return "-"
		}

		diff := int(math.Log10(float64(latest)) - math.Log10(float64(mean)))

		// Keep diff between 3 and -3
		diff = int(math.Min(float64(diff), 3))
		diff = int(math.Max(float64(diff), -3))

		switch diff {
		case 1:
			return "+"
		case 2:
			return "++"
		case 3:
			return "+++"
		case -1:
			return "-"
		case -2:
			return "--"
		case -3:
			return "---"
		default:
			return ""
		}
	}

	return ""
}
