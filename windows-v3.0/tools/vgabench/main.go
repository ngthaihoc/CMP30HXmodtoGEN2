package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	bdf := flag.String("bdf", "", "Target PCI BDF (e.g. 0000:01:00.0)")
	sizeMB := flag.Int("size-mb", 64, "Pinned host transfer buffer size in MiB (4..512)")
	iters := flag.Int("iters", 5, "Number of timed H2D/D2H copy iterations")
	timeoutMs := flag.Int("timeout-ms", 15000, "Watchdog timeout in milliseconds")
	durationMs := flag.Int("duration-ms", 0, "Optional sustained load target duration in milliseconds (0..60000)")
	flag.Parse()

	res := RunBenchmark(NewRealCudaDriver(), BenchConfig{
		BDF:        *bdf,
		SizeMB:     *sizeMB,
		Iterations: *iters,
		TimeoutMs:  *timeoutMs,
		DurationMs: *durationMs,
	})

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(res); err != nil {
		fmt.Fprintf(os.Stderr, "{\"status\":\"error\",\"reason\":%q}\n", err.Error())
		os.Exit(1)
	}
	if res.Status != "ok" {
		os.Exit(2)
	}
}
