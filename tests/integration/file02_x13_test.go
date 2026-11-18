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

func TestFile02X13Sensitivity(t *testing.T) {
	root := os.Getenv("FILE02_X13_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	info, err := os.Lstat(abs)
	if root == "" || err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		filepath.Base(abs) != "raw" || filepath.Dir(filepath.Dir(abs)) != allowed {
		t.Fatal("private ignored X13 raw root required")
	}
	configPath, _ := filepath.Abs("../../configs/experiments/s06_pilot.json")
	config, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var schedule struct {
		Schema string `json:"schema"`
		Jobs   []struct {
			Block int    `json:"block"`
			Order int    `json:"order"`
			Rate  int    `json:"rate"`
			RunID string `json:"run_id"`
		} `json:"jobs"`
	}
	data, err := os.ReadFile(filepath.Join(root, "schedule.json"))
	if err != nil || json.Unmarshal(data, &schedule) != nil ||
		schedule.Schema != "pluto.file02.x13_schedule.v1" || len(schedule.Jobs) != 36 {
		t.Fatal("invalid X13 locked schedule")
	}
	blocks, scope := 12, "X_CONFIRMATORY"
	if os.Getenv("FILE02_X13_QA_LIMIT") == "1" {
		blocks, scope = 1, "QA_ONLY"
	} else if os.Getenv("FILE02_X13_QA_LIMIT") != "" {
		t.Fatal("unsupported X13 QA limit")
	}
	failed := 0
	for block := 0; block < blocks; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-x13", 301)
		if err != nil {
			t.Fatal(err)
		}
		plans := map[int]load.Plan{}
		for _, rate := range []int{20, 23, 25} {
			plans[rate], err = load.BuildX13InvalidPlan(fixture, block, rate, config, hash)
			if err != nil {
				t.Fatal(err)
			}
		}
		seen := map[int]bool{}
		for order := 0; order < 3; order++ {
			job := schedule.Jobs[3*block+order]
			plan, ok := plans[job.Rate]
			if !ok || job.Block != block || job.Order != order || job.RunID != plan.RunID || seen[job.Rate] {
				t.Fatal("X13 rate/order drift")
			}
			seen[job.Rate] = true
			if passed := t.Run(fmt.Sprintf("b%02d-r%d", block, job.Rate), func(t *testing.T) {
				file02ExecuteQA(t, config, configPath, hash, fixture, researchv2.ProfileH,
					filepath.Join(root, "runs", job.RunID), plan, map[string]any{
						"schema": "pluto.file02.x13_run.v1", "scope": scope,
						"case_id": "X13", "profile": "H", "block": block, "order": order,
						"rate": job.Rate, "seed": plan.Seed, "recorder_mode": "ON",
						"commit_observer_mode": "block_scan",
					})
			}); !passed {
				failed++
				t.Logf("X13 retained failed run %s", job.RunID)
			}
			time.Sleep(2 * time.Second)
		}
	}
	if failed != 0 {
		t.Errorf("X13 completed schedule with %d failed runs retained", failed)
	}
}
