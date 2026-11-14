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

func TestFile02A13Dynamics(t *testing.T) {
	root := os.Getenv("FILE02_A13_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	parent, pe := os.Stat(filepath.Dir(filepath.Dir(abs)))
	base, be := os.Stat(allowed)
	info, ie := os.Lstat(abs)
	if root == "" || pe != nil || be != nil || ie != nil || !os.SameFile(parent, base) ||
		filepath.Base(abs) != "raw" || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("private ignored A13 raw root required")
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
	if err != nil || json.Unmarshal(raw, &schedule) != nil || schedule.Schema != "pluto.file02.a13.schedule.v1" || len(schedule.Jobs) != 8 {
		t.Fatal("invalid locked A13 schedule")
	}
	for block := 0; block < 4; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-a13", 241)
		if err != nil {
			t.Fatal(err)
		}
		benign, attack, err := load.BuildAttackDynamicsPairPlans(fixture, block, c, hash)
		if err != nil {
			t.Fatal(err)
		}
		for order := 0; order < 2; order++ {
			job := schedule.Jobs[2*block+order]
			plan := benign
			if job.Condition == "attack" {
				plan = attack
			} else if job.Condition != "benign" {
				t.Fatal("unknown A13 condition")
			}
			if job.Block != block || job.Order != order || job.RunID != plan.RunID {
				t.Fatal("A13 job/plan mismatch")
			}
			if passed := t.Run(fmt.Sprintf("b%02d-%s", block, job.Condition), func(t *testing.T) {
				file02ExecuteQA(t, c, configPath, hash, fixture, researchv2.ProfileH,
					filepath.Join(root, "runs", job.RunID), plan, map[string]any{
						"schema": "pluto.file02.a13_run.v1", "scope": "A_CONFIRMATORY",
						"case_id": "A13", "profile": "H", "block": block,
						"order": order, "condition": job.Condition, "seed": plan.Seed,
						"recorder_mode": "ON", "commit_observer_mode": "block_scan",
					})
			}); !passed {
				t.Fatal("A13 stopped after failed arm; no replacement")
			}
			time.Sleep(2 * time.Second)
		}
	}
}
