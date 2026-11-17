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

// The balanced probe QA uses the same signed I schedule within each pair.
func TestFile02X16ProbeOverheadQA(t *testing.T) {
	root := os.Getenv("FILE02_X16_OVERHEAD_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	info, err := os.Lstat(abs)
	if root == "" || err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		filepath.Base(abs) != "raw" || filepath.Dir(filepath.Dir(abs)) != allowed {
		t.Fatal("private ignored X16 overhead raw root required")
	}
	configPath, _ := filepath.Abs("../../research/config/s06_pilot.json")
	config, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for block := 0; block < 8; block++ {
		// Chain-ID length is part of the signed wire size. Keep the previously
		// verified 2,568-byte H fixture while drawing fresh identities per block.
		fixture, err := load.NewPilotFixture("pluto-file02-x16", 241)
		if err != nil {
			t.Fatal(err)
		}
		_, _, lockedPlan, err := load.BuildX10TripletPlans(fixture, block, config, hash)
		if err != nil {
			t.Fatal(err)
		}
		if len(lockedPlan.Events) != 241 {
			t.Fatal("X16 overhead workload count drift")
		}
		for _, event := range lockedPlan.Events {
			if len(event.BytesHex) != 2*2568 {
				t.Fatal("X16 overhead wire bytes differ from locked 2,568-byte H")
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
				plan.RunID = fmt.Sprintf("X16-OH-b%02d-%s", block, mode)
				file02ExecuteQA(t, config, configPath, hash, fixture, researchv2.ProfileH,
					filepath.Join(abs, "runs", plan.RunID), plan, map[string]any{
						"schema": "pluto.file02.x16_overhead_run.v1", "scope": "QA_ONLY",
						"case_id": "X16", "profile": "H", "condition": "I",
						"block": block, "order": order, "seed": plan.Seed,
						"trace_mode": "ON", "stage_cpu_mode": mode,
						"recorder_mode": "ON", "commit_observer_mode": "block_scan",
					})
			})
		}
	}
}
