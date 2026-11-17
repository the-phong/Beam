//go:build integration && fourval

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

func TestFile02X16MatchedBI(t *testing.T) {
	root := os.Getenv("FILE02_X16_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	info, err := os.Lstat(abs)
	if root == "" || err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		filepath.Base(abs) != "raw" || filepath.Dir(filepath.Dir(abs)) != allowed {
		t.Fatal("private ignored X16 raw root required")
	}
	configPath, _ := filepath.Abs("../../research/config/s06_pilot.json")
	config, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var schedule struct {
		Schema string `json:"schema"`
		Jobs   []struct {
			Block     int    `json:"block"`
			Order     int    `json:"order"`
			Condition string `json:"condition"`
			RunID     string `json:"run_id"`
		} `json:"jobs"`
	}
	data, err := os.ReadFile(filepath.Join(abs, "schedule.json"))
	if err != nil || json.Unmarshal(data, &schedule) != nil ||
		schedule.Schema != "pluto.file02.x16_schedule.v1" || len(schedule.Jobs) != 48 {
		t.Fatal("invalid X16 locked schedule")
	}
	failed := 0
	for block := 0; block < 24; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-x16", 241)
		if err != nil {
			t.Fatal(err)
		}
		_, b, i, err := load.BuildX10TripletPlans(fixture, block, config, hash)
		if err != nil {
			t.Fatal(err)
		}
		plans := map[string]load.Plan{"B": b, "I": i}
		seen := map[string]bool{}
		for order := 0; order < 2; order++ {
			job := schedule.Jobs[2*block+order]
			plan, ok := plans[job.Condition]
			if !ok || job.Block != block || job.Order != order ||
				job.RunID != fmt.Sprintf("X16-b%02d-%s", block, job.Condition) || seen[job.Condition] {
				t.Fatal("X16 arm/order drift")
			}
			seen[job.Condition] = true
			plan.RunID = job.RunID
			if passed := t.Run(job.RunID, func(t *testing.T) {
				file02ExecuteQA(t, config, configPath, hash, fixture, researchv2.ProfileH,
					filepath.Join(abs, "runs", plan.RunID), plan, map[string]any{
						"schema": "pluto.file02.x16_run.v1", "scope": "X_CONFIRMATORY",
						"case_id": "X16", "profile": "H", "block": block, "order": order,
						"condition": job.Condition, "seed": plan.Seed,
						"trace_mode": "ON", "stage_cpu_mode": "ON", "recorder_mode": "ON",
						"commit_observer_mode": "block_scan",
					})
			}); !passed {
				failed++
				t.Logf("X16 retained failed run %s; block excluded from inference", plan.RunID)
			}
			time.Sleep(2 * time.Second)
		}
	}
	if failed != 0 {
		t.Errorf("X16 completed schedule with %d failed runs", failed)
	}
}
