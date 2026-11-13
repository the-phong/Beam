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

func TestFile02R16(t *testing.T) {
	root := os.Getenv("FILE02_R16_ROOT")
	if root == "" {
		t.Fatal("FILE02_R16_ROOT required")
	}
	allowed, _ := filepath.Abs("../../research/artifacts/file02")
	abs, _ := filepath.Abs(root)
	parent, pe := os.Stat(filepath.Dir(filepath.Dir(abs)))
	base, be := os.Stat(allowed)
	info, ie := os.Lstat(abs)
	if pe != nil || be != nil || ie != nil || !os.SameFile(parent, base) || filepath.Base(abs) != "raw" || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("private R16 raw root required")
	}
	var schedule struct {
		Schema string `json:"schema"`
		Jobs   []struct {
			Block   int    `json:"block"`
			Order   int    `json:"order"`
			Profile string `json:"profile"`
			Arm     string `json:"arm"`
			Seed    int64  `json:"seed"`
			RunID   string `json:"run_id"`
		} `json:"jobs"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "schedule.json"))
	if err != nil || json.Unmarshal(raw, &schedule) != nil || schedule.Schema != "pluto.file02.r16.schedule.v1" || len(schedule.Jobs) != 48 {
		t.Fatal("invalid R16 schedule")
	}
	configPath := filepath.Join(root, "config.json")
	c, hash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	profiles := map[string]researchv2.Profile{"C": researchv2.ProfileC, "CP": researchv2.ProfileCP, "P": researchv2.ProfileP, "H": researchv2.ProfileH}
	for block := 0; block < 4; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-r16", 41)
		if err != nil {
			t.Fatal(err)
		}
		for order := 0; order < 12; order++ {
			job := schedule.Jobs[12*block+order]
			p, ok := profiles[job.Profile]
			if !ok || job.Block != block || job.Order != order || job.Seed <= 0 || job.RunID == "" ||
				(job.Arm != "constant" && job.Arm != "ramp" && job.Arm != "burst") {
				t.Fatal("invalid R16 job")
			}
			if passed := t.Run(fmt.Sprintf("b%02d-%s-%s", block, job.Profile, job.Arm), func(t *testing.T) {
				plan, err := load.BuildPilotPlan(fixture, p, load.EqualTxRate, job.Arm, job.RunID, job.Seed, c, hash, 10)
				if err != nil {
					t.Fatal(err)
				}
				file02ExecuteQA(t, c, configPath, hash, fixture, p, filepath.Join(root, "runs", job.RunID), plan, map[string]any{
					"schema": "pluto.file02.r16_run.v1", "scope": "R_CONFIRMATORY", "case_id": "R16",
					"profile": job.Profile, "arm": job.Arm, "block": job.Block, "order": job.Order, "seed": job.Seed,
					"recorder_mode": "ON", "commit_observer_mode": "block_scan",
				})
			}); !passed {
				t.Fatal("R16 stopped after failed arm; no replacement")
			}
			time.Sleep(2 * time.Second)
		}
	}
}
