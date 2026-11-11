// profile-load sends a pre-signed S06 plan only to an isolated loopback RPC.
// Signing keys and validator homes are never passed to this process.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"example.com/pluto/internal/profiles/load"
)

func main() {
	configPath := flag.String("config", "", "pinned S06 JSON configuration")
	planPath := flag.String("plan", "", "private pre-signed JSON plan")
	genesisPath := flag.String("genesis", "", "binary genesis used by the isolated network")
	nodesPath := flag.String("nodes", "", "private four-node observer/PID JSON")
	rpc := flag.String("rpc", "", "one permitted 127.0.0.1 Comet RPC origin")
	dataRoot := flag.String("data-root", "", "isolated network root for free-disk monitoring")
	killFile := flag.String("kill-file", "", "operator-created file that aborts this run")
	resultPath := flag.String("result", "", "new private JSON result path; never overwritten")
	cores := flag.Int("allocated-cores", 0, "logical CPU allocation for this isolated host")
	flag.Parse()
	if flag.NArg() != 0 || *configPath == "" || *planPath == "" || *genesisPath == "" || *nodesPath == "" || *rpc == "" || *dataRoot == "" || *killFile == "" || *resultPath == "" || *cores < 1 || *cores > runtime.NumCPU() {
		fmt.Fprintln(os.Stderr, "all isolated load and safety arguments are required")
		os.Exit(2)
	}
	config, configHash, err := load.ReadConfig(*configPath)
	if err != nil {
		fatal(err)
	}
	plan, planHash, err := load.ReadPlan(*planPath, config, configHash)
	if err != nil {
		fatal(err)
	}
	genesis, err := os.ReadFile(*genesisPath)
	if err != nil {
		fatal(err)
	}
	h := sha256.Sum256(genesis)
	if hex.EncodeToString(h[:]) != plan.GenesisSHA256 {
		fatal(errors.New("plan genesis hash differs from isolated network genesis"))
	}
	nodes, err := readNodes(*nodesPath)
	if err != nil {
		fatal(err)
	}
	var endpointFound bool
	for _, n := range nodes {
		if n.RPC == *rpc {
			endpointFound = true
		}
	}
	if !endpointFound {
		fatal(errors.New("RPC endpoint is outside watched validator allowlist"))
	}
	sender, err := load.NewHTTPSender(*rpc, time.Duration(config.Safety.RequestTimeoutMS)*time.Millisecond)
	if err != nil {
		fatal(err)
	}
	guard, err := load.NewHostGuard(config, nodes, *dataRoot, *killFile, *cores)
	if err != nil {
		fatal(err)
	}
	resultFile, err := os.OpenFile(*resultPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		fatal(fmt.Errorf("result already exists or is unwritable: %w", err))
	}
	defer resultFile.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := load.Run(ctx, plan, config, configHash, planHash, sender, guard)
	if err != nil {
		result = load.Result{Version: 1, RunID: plan.RunID, ConfigSHA256: configHash, PlanSHA256: planHash, GenesisSHA256: plan.GenesisSHA256,
			Profile: byte(plan.Profile), Regime: plan.Regime, Status: load.StatusSafetyAbort, Reason: "preflight refused run: " + err.Error(), CompletedUTC: time.Now().UTC().Format(time.RFC3339Nano)}
	}
	if err := json.NewEncoder(resultFile).Encode(result); err != nil {
		fatal(err)
	}
	if err := resultFile.Sync(); err != nil {
		fatal(err)
	}
	if result.Status != load.StatusOK {
		fmt.Fprintf(os.Stderr, "run %s: %s: %s\n", plan.RunID, result.Status, result.Reason)
		os.Exit(1)
	}
	fmt.Printf("run=%s status=OK planned=%d actual=%d\n", plan.RunID, result.PlannedCount, result.ActualSendCount)
}

func readNodes(path string) ([4]load.WatchedNode, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return [4]load.WatchedNode{}, err
	}
	var nodes [4]load.WatchedNode
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&nodes); err != nil {
		return nodes, err
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nodes, errors.New("invalid trailing node document")
	}
	return nodes, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
