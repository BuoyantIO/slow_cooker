package hdrreport

import (
	"fmt"
	"github.com/codahale/hdrhistogram"
	"os"
)

func WriteReportCSV(filename *string, hist *hdrhistogram.Histogram) error {
	f, err := os.Create(*filename)

	if err != nil {
		return err
	}

	for _, bar := range hist.Distribution() {
		_, err := f.Write([]byte(bar.String()))

		if err != nil {
			return err
		}
	}

	err = f.Sync()

	if err != nil {
		return err
	}

	err = f.Close()

	if err != nil {
		return err
	}

	return nil
}

func PrintLatencySummary(hist *hdrhistogram.Histogram) {
	fmt.Printf("FROM    TO #REQUESTS\n")
	fmt.Printf("   0     2 %d\n", SumBars(0, 2, hist.Distribution()))
	fmt.Printf("   2     8 %d\n", SumBars(2, 8, hist.Distribution()))
	fmt.Printf("   8    32 %d\n", SumBars(8, 32, hist.Distribution()))
	fmt.Printf("  32    64 %d\n", SumBars(32, 64, hist.Distribution()))
	fmt.Printf("  64   128 %d\n", SumBars(64, 128, hist.Distribution()))
	fmt.Printf(" 128   256 %d\n", SumBars(128, 256, hist.Distribution()))
	fmt.Printf(" 256   512 %d\n", SumBars(256, 512, hist.Distribution()))
	fmt.Printf(" 512  1024 %d\n", SumBars(512, 1024, hist.Distribution()))
	fmt.Printf("1024  4096 %d\n", SumBars(1024, 4096, hist.Distribution()))
	fmt.Printf("4096 16384 %d\n", SumBars(4096, 16384, hist.Distribution()))
}

// Given a sorted `[]hdrhistogram.Bar`, return the sum of every `Bar` in the
// Range of (from, to]. Inclusive of from, exclusive of to.
func SumBars(from int64, to int64, bars []hdrhistogram.Bar) int64 {
	count := int64(0)
	for _, bar := range bars {
		if bar.To >= to {
			// short circuit if we've passed the item
			// we're interested in.
			break
		}
		if bar.From >= from && bar.To < to {
			count = count + bar.Count
		}
	}
	return count
}
