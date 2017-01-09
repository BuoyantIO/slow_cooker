package scripting

import (
	"flag"
	"log"
	"testing"
)

var (
	testScript  = flag.String("script", "", "lua script to benchmark")
	testMethod  = flag.String("method", "GET", "value of method arg passed to slow_cooker.generate_data")
	testUrl     = flag.String("url", "localhost", "value of url arg passed to slow_cooker.generate_data")
	testHost    = flag.String("host", "", "value of host arg passed to slow_cooker.generate_data")
	concurrency = flag.Int("concurrency", 10, "number of concurrent lua VMs")
)

func init() {
	flag.Parse()
}

func BenchmarkDataGeneratorScript(b *testing.B) {
	if *testScript == "" {
		log.Fatal("Missing required -dataGeneratorScript flag")
	}
	lPool := NewLStatePool(*testScript, *concurrency, DefaultModuleLoaders)
	dataGenerator := NewDataGenerator(lPool)
	for i := 0; i < b.N; i++ {
		lReqData := &LReqData{
			Method: *testMethod,
			Url:    *testUrl,
			Host:   *testHost,
		}
		dataGenerator(lReqData, uint64(i))
	}
}
