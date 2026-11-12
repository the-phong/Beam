//go:build integration && fourval

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// file02ExecuteQA runs one pre-signed bounded plan on a fresh local network.
// Only keyless run evidence is persisted under the caller's ignored root.
func file02ExecuteQA(t *testing.T, c load.Config, configPath, configHash string, fixture *load.PilotFixture,
	profile researchv2.Profile, run string, plan load.Plan, metadata map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(run), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(run, 0700); err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(run, "plan.json")
	if err := load.WritePlan(planPath, plan, c, configHash); err != nil {
		t.Fatal(err)
	}
	genesisPath := filepath.Join(run, "genesis.bin")
	if err := os.WriteFile(genesisPath, fixture.Genesis, 0600); err != nil {
		t.Fatal(err)
	}
	scope, ok := metadata["scope"].(string)
	if !ok {
		scope = "QA_ONLY"
	}
	if scope != "QA_ONLY" && scope != "T_PREFLIGHT" {
		t.Fatal("unsupported File 02 run scope")
	}
	metadata["scope"] = scope
	metadata["run_id"] = plan.RunID
	metadata["source_commit"] = os.Getenv("FILE02_SOURCE_COMMIT")
	metadata["source_tree"] = os.Getenv("FILE02_SOURCE_TREE")
	metadata["config_sha256"] = configHash
	metadata["network"] = "fresh_four_validator_single_host_loopback"
	if err := s07WriteJSON(filepath.Join(run, "metadata.json"), metadata); err != nil {
		t.Fatal(err)
	}
	pilot := startS06NetworkAt(t, c, fixture, profile, t.TempDir())
	pilot.startObservers(t, c)
	deadline := time.Now().Add(10 * time.Second)
	for {
		ready := true
		for _, watched := range pilot.nodes {
			info, err := os.Stat(watched.ObserverFile)
			if err != nil || info.Size() == 0 {
				ready = false
				break
			}
		}
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("four observers did not become ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	nodes, err := json.Marshal(pilot.nodes)
	if err != nil {
		t.Fatal(err)
	}
	if err := s07WriteExclusive(filepath.Join(run, "nodes.json"), nodes); err != nil {
		t.Fatal(err)
	}
	captureCommits := metadata["capture_commit_observations"] != false
	var observed chan []s07CommitObservation
	if captureCommits {
		observed = make(chan []s07CommitObservation, 1)
		stopObservation := make(chan struct{})
		if metadata["commit_observer_mode"] == "block_scan" {
			go file02ObserveCommitsByBlock(pilot, plan, stopObservation, observed)
		} else {
			go s07ObserveCommits(pilot, plan, stopObservation, observed)
		}
		defer close(stopObservation)
	}
	binary, _ := filepath.Abs("../../build/profile-load")
	if metadata["inject_stop_before_load"] == true {
		if err := os.WriteFile(filepath.Join(run, "STOP"), []byte("QA safety stop\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(binary, "-config", configPath, "-plan", planPath,
		"-genesis", genesisPath, "-nodes", filepath.Join(run, "nodes.json"),
		"-rpc", pilot.nodes[0].RPC, "-data-root", pilot.n.root,
		"-kill-file", filepath.Join(run, "STOP"), "-result", filepath.Join(run, "result.json"),
		"-allocated-cores", fmt.Sprint(runtime.NumCPU()))
	start := time.Now()
	output, runErr := cmd.CombinedOutput()
	wallNS := time.Since(start).Nanoseconds()
	if err := s07WriteExclusive(filepath.Join(run, "load.log"), output); err != nil {
		t.Fatal(err)
	}
	// Preserve the independent recorder streams even when the bounded load
	// aborts. The run directory is private and remains available for QC.
	for i, watched := range pilot.nodes {
		observerRaw, err := os.ReadFile(watched.ObserverFile)
		if err != nil {
			t.Fatalf("read observer %d: %v", i, err)
		}
		if err := s07WriteExclusive(filepath.Join(run, fmt.Sprintf("observer-%d.jsonl", i)), observerRaw); err != nil {
			t.Fatal(err)
		}
	}
	if cmd.ProcessState != nil {
		resources := map[string]any{
			"schema": "pluto.file02.generator_resources.v1", "run_id": plan.RunID,
			"wall_ns": wallNS, "user_cpu_ns": cmd.ProcessState.UserTime().Nanoseconds(),
			"system_cpu_ns": cmd.ProcessState.SystemTime().Nanoseconds(),
		}
		if usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
			resources["peak_rss_bytes"] = usage.Maxrss * 1024
		}
		if err := s07WriteJSON(filepath.Join(run, "generator_resources.json"), resources); err != nil {
			t.Fatal(err)
		}
	}
	if runErr != nil {
		t.Fatalf("bounded load failed: %v (private load.log)", runErr)
	}
	var result load.Result
	raw, err := os.ReadFile(filepath.Join(run, "result.json"))
	if err != nil || json.Unmarshal(raw, &result) != nil || result.Status != load.StatusOK || result.ActualSendCount != len(plan.Events) {
		t.Fatalf("invalid load result: %v status=%s", err, result.Status)
	}
	pilot.n.waitAll(8, 30*time.Second)
	oracle := s07Oracle(t, pilot, fixture, plan, result)
	if err := s07WriteJSON(filepath.Join(run, "oracle.json"), oracle); err != nil {
		t.Fatal(err)
	}
	if captureCommits {
		var commits []s07CommitObservation
		select {
		case commits = <-observed:
		case <-time.After(10 * time.Second):
			t.Fatal("commit observations incomplete")
		}
		if len(commits) != oracle.ValidCommitted {
			t.Fatalf("commit observations %d differ from valid count %d", len(commits), oracle.ValidCommitted)
		}
		if err := s07WriteJSON(filepath.Join(run, "commit_observations.json"), map[string]any{
			"schema": "pluto.s07.commits.v1", "run_id": plan.RunID, "poll_interval_ms": 100,
			"observation_semantics": "first RPC tx-index observation on all four validators; upper bound on inclusion time",
			"events":                commits,
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("File02 QA run=%s valid=%d invalid=%d height=%d", plan.RunID, oracle.ValidCommitted, oracle.InvalidRejected, oracle.CommonHeight)
}
