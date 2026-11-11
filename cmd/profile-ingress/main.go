package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/pluto/internal/telemetry"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:26757", "loopback listener for controlled transaction RPC")
	backend := flag.String("backend", "http://127.0.0.1:26657", "loopback Comet RPC origin")
	capacity := flag.Int("capacity", 4096, "bounded pending telemetry events")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments")
		os.Exit(2)
	}
	host, _, err := net.SplitHostPort(*listen)
	if err != nil || host != "127.0.0.1" {
		fmt.Fprintln(os.Stderr, "listener must be 127.0.0.1")
		os.Exit(2)
	}
	proxy, err := telemetry.NewIngressProxy(*backend, *capacity)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Handler: proxy, ReadHeaderTimeout: 3 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	enc := json.NewEncoder(os.Stdout)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	flush := func() error {
		events, dropped := proxy.Drain()
		for _, event := range events {
			if err := enc.Encode(event); err != nil {
				return err
			}
		}
		if dropped > 0 {
			return enc.Encode(map[string]any{"schema": telemetry.Schema, "kind": "rpc_proxy_drops", "observed_utc": time.Now().UTC().Format(time.RFC3339Nano), "dropped": dropped})
		}
		return nil
	}
	for {
		select {
		case <-ticker.C:
			if err := flush(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		case err := <-done:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = server.Shutdown(shutdownCtx)
			cancel()
			if err := flush(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}
}
