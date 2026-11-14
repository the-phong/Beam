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

func TestFile02A14ProcessFault(t *testing.T) {
	root := os.Getenv("FILE02_A14_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	parent, pe := os.Stat(filepath.Dir(filepath.Dir(abs)))
	base, be := os.Stat(allowed)
	info, ie := os.Lstat(abs)
	if root == "" || pe != nil || be != nil || ie != nil || !os.SameFile(parent, base) ||
		filepath.Base(abs) != "raw" || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("private ignored A14 raw root required")
	}
	configPath, _ := filepath.Abs("../../research/config/s06_pilot.json")
	c, hash, err := load.ReadConfig(configPath)
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
	raw, err := os.ReadFile(filepath.Join(root, "schedule.json"))
	if err != nil || json.Unmarshal(raw, &schedule) != nil || schedule.Schema != "pluto.file02.a14.schedule.v1" || len(schedule.Jobs) != 8 {
		t.Fatal("invalid locked A14 schedule")
	}
	for block := 0; block < 4; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-a14", 31)
		if err != nil {
			t.Fatal(err)
		}
		for order := 0; order < 2; order++ {
			job := schedule.Jobs[2*block+order]
			if job.Block != block || job.Order != order || (job.Condition != "control" && job.Condition != "fault") {
				t.Fatal("A14 schedule mismatch")
			}
			plan, err := load.BuildPilotPlan(fixture, researchv2.ProfileH, load.EqualTxRate,
				"constant", job.RunID, int64(2026111400+block), c, hash, 6)
			if err != nil || len(plan.Events) != 25 {
				t.Fatalf("A14 plan: %v events=%d", err, len(plan.Events))
			}
			if passed := t.Run(fmt.Sprintf("b%02d-%s", block, job.Condition), func(t *testing.T) {
				file02ExecuteQA(t, c, configPath, hash, fixture, researchv2.ProfileH,
					filepath.Join(root, "runs", job.RunID), plan, map[string]any{
						"schema": "pluto.file02.a14_run.v1", "scope": "A_CONFIRMATORY",
						"case_id": "A14", "profile": "H", "block": block,
						"order": order, "condition": job.Condition, "seed": plan.Seed,
						"recorder_mode": "ON", "commit_observer_mode": "block_scan",
					})
			}); !passed {
				t.Fatal("A14 stopped after failed arm; no replacement")
			}
			time.Sleep(2 * time.Second)
		}
	}
}
