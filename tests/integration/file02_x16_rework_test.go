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

// Engineering QA only. The two fresh networks share one signed fixture.
func TestFile02X16StageCPUQA(t *testing.T) {
	root := os.Getenv("FILE02_X16_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	info, err := os.Lstat(abs)
	if root == "" || err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		filepath.Base(abs) != "raw" || filepath.Dir(filepath.Dir(abs)) != allowed {
		t.Fatal("private ignored X16 raw root required")
	}
	configPath, _ := filepath.Abs("../../configs/experiments/s06_pilot.json")
	config, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := load.NewPilotFixture("pluto-file02-x16", 241)
	if err != nil {
		t.Fatal(err)
	}
	_, _, lockedPlan, err := load.BuildX10TripletPlans(fixture, 0, config, hash)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"OFF", "ON"} {
		t.Run(mode, func(t *testing.T) {
			plan := lockedPlan
			plan.RunID = fmt.Sprintf("X16-QA-I-%s", mode)
			file02ExecuteQA(t, config, configPath, hash, fixture, researchv2.ProfileH,
				filepath.Join(abs, "runs", plan.RunID), plan, map[string]any{
					"schema": "pluto.file02.x16_qa_run.v1", "scope": "QA_ONLY",
					"case_id": "X16", "profile": "H", "condition": "I",
					"seed": plan.Seed, "trace_mode": "ON", "stage_cpu_mode": mode,
					"recorder_mode": "ON", "commit_observer_mode": "block_scan",
				})
		})
	}
}
