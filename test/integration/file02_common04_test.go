//go:build integration && fourval

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// TestFile02Common04MatchedRates exercises equal-tx and equal-byte regimes
// across all profiles on fresh isolated networks. This is bounded QA only.
func TestFile02Common04MatchedRates(t *testing.T) {
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("FILE02_COMMON04_ROOT")
	if root == "" {
		root = t.TempDir()
	} else {
		allowed, err := filepath.Abs("../../research/artifacts/file02")
		if err != nil {
			t.Fatal(err)
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			t.Fatal(err)
		}
		parent, parentErr := os.Stat(filepath.Dir(abs))
		base, baseErr := os.Stat(allowed)
		info, rootErr := os.Lstat(abs)
		if parentErr != nil || baseErr != nil || rootErr != nil || !os.SameFile(parent, base) || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("FILE02_COMMON04_ROOT must be a private directory directly below ignored research/artifacts/file02")
		}
		root = abs
	}
	fixture, err := load.NewPilotFixture("pluto-file02-common04", 90)
	if err != nil {
		t.Fatal(err)
	}
	profiles := []struct {
		name string
		code researchv2.Profile
	}{
		{"C", researchv2.ProfileC}, {"CP", researchv2.ProfileCP},
		{"P", researchv2.ProfileP}, {"H", researchv2.ProfileH},
	}
	for regimeIndex, regime := range []struct {
		name, code string
		rate       int
	}{
		{"equal-tx", load.EqualTxRate, 2}, {"equal-byte", load.EqualByteRate, 0},
	} {
		regimeIndex, regime := regimeIndex, regime
		t.Run(regime.name, func(t *testing.T) {
			for i, profile := range profiles {
				i, profile := i, profile
				t.Run(profile.name, func(t *testing.T) {
					plan, err := load.BuildPilotPlan(fixture, profile.code, regime.code, "constant",
						fmt.Sprintf("common04-%s-%s", regime.name, profile.name),
						int64(1204+10*regimeIndex+i), c, configHash, regime.rate)
					if err != nil {
						t.Fatal(err)
					}
					run := filepath.Join(root, regime.name, profile.name)
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
					pilot := startS06NetworkAt(t, c, fixture, profile.code, t.TempDir())
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
					binary, _ := filepath.Abs("../../build/profile-load")
					cmd := exec.Command(binary, "-config", configPath, "-plan", planPath,
						"-genesis", genesisPath, "-nodes", filepath.Join(run, "nodes.json"),
						"-rpc", pilot.nodes[0].RPC, "-data-root", pilot.n.root,
						"-kill-file", filepath.Join(run, "STOP"), "-result", filepath.Join(run, "result.json"),
						"-allocated-cores", fmt.Sprint(runtime.NumCPU()))
					output, runErr := cmd.CombinedOutput()
					if err := s07WriteExclusive(filepath.Join(run, "load.log"), output); err != nil {
						t.Fatal(err)
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
					if err := s07WriteJSON(filepath.Join(run, "metadata.json"), map[string]any{
						"schema": "pluto.file02.common04_run.v1", "scope": "QA_ONLY", "run_id": plan.RunID,
						"profile": profile.name, "regime": regime.name,
						"source_commit": os.Getenv("FILE02_SOURCE_COMMIT"), "source_tree": os.Getenv("FILE02_SOURCE_TREE"),
						"config_sha256": configHash, "network": "fresh_four_validator_single_host_loopback",
					}); err != nil {
						t.Fatal(err)
					}
					t.Logf("COMMON04 QA regime=%s profile=%s valid=%d height=%d", regime.name, profile.name, oracle.ValidCommitted, oracle.CommonHeight)
				})
			}
		})
	}
}
