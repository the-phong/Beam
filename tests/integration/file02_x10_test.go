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

func TestFile02X10ThreeArm(t *testing.T) {
	root := os.Getenv("FILE02_X10_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	info, err := os.Lstat(abs)
	if root == "" || err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		filepath.Base(abs) != "raw" || filepath.Dir(filepath.Dir(abs)) != allowed {
		t.Fatal("private ignored X10 raw root required")
	}
	configPath, _ := filepath.Abs("../../configs/experiments/s06_pilot.json")
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
	data, err := os.ReadFile(filepath.Join(root, "schedule.json"))
	if err != nil || json.Unmarshal(data, &schedule) != nil ||
		schedule.Schema != "pluto.file02.x10_schedule.v1" || len(schedule.Jobs) != 72 {
		t.Fatal("invalid X10 locked schedule")
	}
	blocks := 24
	scope := "X_CONFIRMATORY"
	if os.Getenv("FILE02_X10_QA_LIMIT") == "1" {
		blocks, scope = 1, "QA_ONLY"
	} else if os.Getenv("FILE02_X10_QA_LIMIT") != "" {
		t.Fatal("unsupported X10 QA limit")
	}
	failedArms := 0
	for block := 0; block < blocks; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-x10", 241)
		if err != nil {
			t.Fatal(err)
		}
		l, b, i, err := load.BuildX10TripletPlans(fixture, block, config, hash)
		if err != nil {
			t.Fatal(err)
		}
		plans := map[string]load.Plan{"L": l, "B": b, "I": i}
		seen := map[string]bool{}
		for order := 0; order < 3; order++ {
			job := schedule.Jobs[3*block+order]
			plan, ok := plans[job.Condition]
			if !ok || job.Block != block || job.Order != order || job.RunID != plan.RunID || seen[job.Condition] {
				t.Fatal("X10 arm/order drift")
			}
			seen[job.Condition] = true
			if passed := t.Run(fmt.Sprintf("b%02d-%s", block, job.Condition), func(t *testing.T) {
				file02ExecuteQA(t, config, configPath, hash, fixture, researchv2.ProfileH,
					filepath.Join(root, "runs", job.RunID), plan, map[string]any{
						"schema": "pluto.file02.x10_run.v1", "scope": scope,
						"case_id": "X10", "profile": "H", "block": block, "order": order,
						"condition": job.Condition, "seed": plan.Seed, "recorder_mode": "ON",
						"commit_observer_mode": "block_scan",
					})
			}); !passed {
				failedArms++
				t.Logf("X10 retained failed arm %s; block excluded if QC fails; no replacement", job.RunID)
			}
			time.Sleep(2 * time.Second)
		}
	}
	if failedArms != 0 {
		t.Errorf("X10 completed schedule with %d failed arms retained for QC", failedArms)
	}
}
