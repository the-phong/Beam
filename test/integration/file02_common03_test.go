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

// TestFile02Common03FixedLegitimate is bounded QA, not a confirmatory File 02 case.
// Each pair reuses one fixture and the exact legitimate signed bytes and offsets
// on two fresh loopback networks; only the attack network offers extra invalid txs.
func TestFile02Common03FixedLegitimate(t *testing.T) {
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("FILE02_COMMON03_ROOT")
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
			t.Fatal("FILE02_COMMON03_ROOT must be an existing private directory directly below ignored research/artifacts/file02")
		}
		root = abs
	}
	profiles := []struct {
		name string
		code researchv2.Profile
	}{
		{"C", researchv2.ProfileC}, {"CP", researchv2.ProfileCP},
		{"P", researchv2.ProfileP}, {"H", researchv2.ProfileH},
	}
	for i, profile := range profiles {
		i, profile := i, profile
		t.Run(profile.name, func(t *testing.T) {
			fixture, err := load.NewPilotFixture("pluto-file02-common03", 16)
			if err != nil {
				t.Fatal(err)
			}
			attack, err := load.BuildPilotPlan(fixture, profile.code, load.FixedLegitimatePlusAdversary,
				"constant", fmt.Sprintf("common03-%s-attack", profile.name), int64(1103+i), c, configHash, 0)
			if err != nil {
				t.Fatal(err)
			}
			benign := attack
			benign.RunID = fmt.Sprintf("common03-%s-benign", profile.name)
			benign.Regime = load.EqualTxRate
			benign.TargetTxPS, benign.TargetBytesPS, benign.TargetLegitTPS, benign.TargetInvalidTPS = 1, 0, 0, 0
			benign.Events = nil
			for _, event := range attack.Events {
				if event.Traffic == "legitimate" {
					benign.Events = append(benign.Events, event)
				}
			}
			if err := benign.Validate(c, configHash); err != nil {
				t.Fatal(err)
			}
			for _, pair := range []struct {
				name string
				plan load.Plan
			}{{"benign", benign}, {"attack", attack}} {
				pair := pair
				t.Run(pair.name, func(t *testing.T) {
					run := filepath.Join(root, "profile-"+profile.name, pair.name)
					if err := os.MkdirAll(filepath.Dir(run), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(run, 0700); err != nil {
						t.Fatal(err)
					}
					planPath := filepath.Join(run, "plan.json")
					if err := load.WritePlan(planPath, pair.plan, c, configHash); err != nil {
						t.Fatal(err)
					}
					genesisPath := filepath.Join(run, "genesis.bin")
					if err := os.WriteFile(genesisPath, fixture.Genesis, 0600); err != nil {
						t.Fatal(err)
					}
					pilot := startS06NetworkAt(t, c, fixture, profile.code, t.TempDir())
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
					if err != nil || json.Unmarshal(raw, &result) != nil || result.Status != load.StatusOK || result.ActualSendCount != len(pair.plan.Events) {
						t.Fatalf("invalid load result: %v status=%s", err, result.Status)
					}
					pilot.n.waitAll(8, 30*time.Second)
					oracle := s07Oracle(t, pilot, fixture, pair.plan, result)
					if err := s07WriteJSON(filepath.Join(run, "oracle.json"), oracle); err != nil {
						t.Fatal(err)
					}
					if err := s07WriteJSON(filepath.Join(run, "metadata.json"), map[string]any{
						"schema": "pluto.file02.common03_run.v1", "scope": "QA_ONLY", "run_id": pair.plan.RunID,
						"profile": profile.name, "condition": pair.name,
						"source_commit": os.Getenv("FILE02_SOURCE_COMMIT"), "source_tree": os.Getenv("FILE02_SOURCE_TREE"),
						"config_sha256": configHash, "network": "fresh_four_validator_single_host_loopback",
					}); err != nil {
						t.Fatal(err)
					}
					t.Logf("COMMON03 QA profile=%s condition=%s valid=%d invalid=%d height=%d", profile.name, pair.name, oracle.ValidCommitted, oracle.InvalidRejected, oracle.CommonHeight)
				})
			}
		})
	}
}
