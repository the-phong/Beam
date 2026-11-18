//go:build integration && fourval

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// Source-locked, balanced X07 queue-trace QA; no service-effect inference.
func TestFile02X07QueueV2QA(t *testing.T) {
	root := os.Getenv("FILE02_X07_V2_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	info, err := os.Lstat(abs)
	if root == "" || err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		filepath.Base(abs) != "raw" || filepath.Dir(filepath.Dir(abs)) != allowed {
		t.Fatal("private ignored X07 v2 raw root required")
	}
	configPath, _ := filepath.Abs("../../configs/experiments/s06_pilot.json")
	config, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for block := 0; block < 8; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-x07", 241)
		if err != nil {
			t.Fatal(err)
		}
		_, _, lockedPlan, err := load.BuildX10TripletPlans(fixture, block, config, hash)
		if err != nil {
			t.Fatal(err)
		}
		if len(lockedPlan.Events) != 241 {
			t.Fatal("X07 v2 workload count drift")
		}
		for _, event := range lockedPlan.Events {
			if len(event.BytesHex) != 2*2568 {
				t.Fatal("X07 v2 wire bytes differ from locked 2,568-byte H")
			}
		}
		modes := []string{"OFF", "ON"}
		if block%2 == 1 {
			modes = []string{"ON", "OFF"}
		}
		for order, mode := range modes {
			block, order, mode := block, order, mode
			t.Run(fmt.Sprintf("b%02d-%s", block, mode), func(t *testing.T) {
				plan := lockedPlan
				plan.RunID = fmt.Sprintf("X07-V2-b%02d-%s", block, mode)
				file02ExecuteQA(t, config, configPath, hash, fixture, researchv2.ProfileH,
					filepath.Join(abs, "runs", plan.RunID), plan, map[string]any{
						"schema": "pluto.file02.x07_v2_run.v1", "scope": "QA_ONLY",
						"case_id": "X07", "profile": "H", "condition": "I",
						"block": block, "order": order, "seed": plan.Seed,
						"trace_mode": mode, "x07_queue_v2": true,
						"recorder_mode": "ON", "commit_observer_mode": "block_scan",
					})
			})
		}
	}
}
