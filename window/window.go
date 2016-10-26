package window

// Returns the mean of a slice of int64.
func Mean(data []int) int {
	sum := 0

	for _, n := range data {
		sum += n
	}

	count := len(data)
	if count > 0 {
		return sum / count
	} else {
		return 0
	}
}

// Given a window of recent latencies, determine if a Change
// Indicator should be generated.
//
// For each 10x over the mean the latest item is, we add a single plus
// sign up to 3.
//
// For each 10x under the mean the latest item is, we add a single
// minus sign up to 3.
//
// Otherwise we return no change indicator.
func CalculateChangeIndicator(data []int, latest int) string {
	mad := Mean(data)

	if len(data) > 0 {
		if latest >= (mad * 1000) {
			return "+++"
		}

		if latest >= (mad * 100) {
			return "++"
		}

		if latest >= (mad * 10) {
			return "+"
		}

		if latest <= (mad / 1000) {
			return "---"
		}

		if latest <= (mad / 100) {
			return "--"
		}

		if latest <= (mad / 10) {
			return "-"
		}
	}

	return ""
}
