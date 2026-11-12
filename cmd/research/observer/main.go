package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/pluto/internal/experiment/telemetry"
)

func main() {
	rpc := flag.String("rpc", "", "loopback Comet RPC origin, for example http://127.0.0.1:26657")
	metrics := flag.String("metrics", "", "loopback Comet Prometheus origin")
	cometPID := flag.Int("comet-pid", 0, "Comet process PID")
	appPID := flag.Int("app-pid", 0, "profile application process PID")
	interval := flag.Duration("interval", time.Second, "sample interval")
	count := flag.Int("count", 1, "number of samples; 0 runs until interrupted")
	flag.Parse()
	if flag.NArg() != 0 || *interval <= 0 || *count < 0 {
		fmt.Fprintln(os.Stderr, "invalid observer arguments")
		os.Exit(2)
	}
	c, err := telemetry.NewCollector(*rpc, *metrics, *cometPID, *appPID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	enc := json.NewEncoder(os.Stdout)
	for i := 0; *count == 0 || i < *count; i++ {
		if err := enc.Encode(c.Sample(ctx)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if *count != 0 && i+1 == *count {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(*interval):
		}
	}
}
