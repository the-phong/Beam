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

func TestFile02AttackCase(t *testing.T) {
	caseID := os.Getenv("FILE02_A_CASE")
	if !(caseID >= "A01" && caseID <= "A05" || caseID == "A08") || len(caseID) != 3 {
		t.Fatal("A01–A05 or A08 case ID required")
	}
	root := os.Getenv("FILE02_A_ROOT")
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	parent, pe := os.Stat(filepath.Dir(filepath.Dir(abs)))
	base, be := os.Stat(allowed)
	info, ie := os.Lstat(abs)
	if root == "" || pe != nil || be != nil || ie != nil || !os.SameFile(parent, base) ||
		filepath.Base(abs) != "raw" || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("private ignored attack raw root required")
	}
	configPath, _ := filepath.Abs("../../research/config/s06_pilot.json")
	config, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var schedule struct {
		Schema string `json:"schema"`
		CaseID string `json:"case_id"`
		Jobs   []struct {
			Pair      int    `json:"pair"`
			Order     int    `json:"order"`
			Condition string `json:"condition"`
			RunID     string `json:"run_id"`
		} `json:"jobs"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "schedule.json"))
	if err != nil || json.Unmarshal(raw, &schedule) != nil || schedule.CaseID != caseID ||
		schedule.Schema != "pluto.file02.attack_schedule.v1" || len(schedule.Jobs) != 24 {
		t.Fatal("invalid locked attack schedule")
	}
	for pair := 0; pair < 12; pair++ {
		fixture, err := load.NewPilotFixture("pluto-file02-"+caseID, 241)
		if err != nil {
			t.Fatal(err)
		}
		profile := researchv2.ProfileH
		benign, attack, err := load.BuildAttackPairPlans(fixture, profile, caseID, pair, config, hash)
		if err != nil {
			t.Fatal(err)
		}
		for order := 0; order < 2; order++ {
			job := schedule.Jobs[2*pair+order]
			if job.Pair != pair || job.Order != order || job.RunID == "" {
				t.Fatal("attack pair order drift")
			}
			plan := benign
			if job.Condition == "attack" {
				plan = attack
			} else if job.Condition != "benign" {
				t.Fatal("unknown attack arm")
			}
			if job.RunID != plan.RunID {
				t.Fatal("scheduled run ID differs from plan")
			}
			if passed := t.Run(fmt.Sprintf("p%02d-%s", pair, job.Condition), func(t *testing.T) {
				file02ExecuteQA(t, config, configPath, hash, fixture, profile,
					filepath.Join(root, "runs", job.RunID), plan, map[string]any{
						"schema": "pluto.file02.attack_run.v1", "scope": "R_CONFIRMATORY",
						"case_id": caseID, "profile": "H", "pair": pair, "order": order,
						"condition": job.Condition, "seed": plan.Seed, "recorder_mode": "ON",
						"commit_observer_mode": "block_scan",
					})
			}); !passed {
				t.Fatal("attack case stopped after failed arm; no replacement")
			}
			time.Sleep(2 * time.Second)
		}
	}
}
