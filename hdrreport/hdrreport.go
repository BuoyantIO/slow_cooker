package hdrreport

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/HdrHistogram/hdrhistogram-go"
)

// Quantiles contains common latency quantiles (p50, p95, p999)
type Quantiles struct {
	Quantile50  int64 `json:"p50"`
	Quantile75  int64 `json:"p75"`
	Quantile90  int64 `json:"p90"`
	Quantile95  int64 `json:"p95"`
	Quantile99  int64 `json:"p99"`
	Quantile999 int64 `json:"p999"`
}

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
	latency := Quantiles{
		Quantile50:  hist.ValueAtQuantile(50),
		Quantile75:  hist.ValueAtQuantile(75),
		Quantile90:  hist.ValueAtQuantile(90),
		Quantile95:  hist.ValueAtQuantile(95),
		Quantile99:  hist.ValueAtQuantile(99),
		Quantile999: hist.ValueAtQuantile(999),
	}

	if data, err := json.MarshalIndent(latency, "", "  "); err != nil {
		log.Fatal("Unable to generate report: ", err)
	} else {
		fmt.Println(string(data))
	}
}
