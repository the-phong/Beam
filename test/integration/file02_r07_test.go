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

func TestFile02R07(t *testing.T) {
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("FILE02_R07_ROOT")
	if root == "" {
		t.Fatal("FILE02_R07_ROOT is required")
	}
	allowed, err := filepath.Abs("../../research/artifacts/file02")
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	parent, parentErr := os.Stat(filepath.Dir(filepath.Dir(abs)))
	base, baseErr := os.Stat(allowed)
	info, rootErr := os.Lstat(abs)
	if parentErr != nil || baseErr != nil || rootErr != nil || !os.SameFile(parent, base) || filepath.Base(abs) != "raw" || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("R07 root must be a private raw directory below ignored research/artifacts/file02")
	}
	raw, err := os.ReadFile(filepath.Join(root, "schedule.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schedule struct {
		Schema string `json:"schema"`
		Jobs   []struct {
			Block   int    `json:"block"`
			Order   int    `json:"order"`
			Profile string `json:"profile"`
			Seed    int64  `json:"seed"`
			RunID   string `json:"run_id"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(raw, &schedule); err != nil || schedule.Schema != "pluto.file02.r07.schedule.v1" || len(schedule.Jobs) != 48 {
		t.Fatal("invalid R07 schedule")
	}
	profiles := map[string]researchv2.Profile{"C": researchv2.ProfileC, "CP": researchv2.ProfileCP, "P": researchv2.ProfileP, "H": researchv2.ProfileH}
	for block := 0; block < 12; block++ {
		fixture, err := load.NewPilotFixture("pluto-file02-rq1", 70)
		if err != nil {
			t.Fatal(err)
		}
		for order := 0; order < 4; order++ {
			job := schedule.Jobs[4*block+order]
			p, ok := profiles[job.Profile]
			if !ok || job.Block != block || job.Order != order || job.Seed <= 0 || job.RunID == "" {
				t.Fatal("invalid scheduled job")
			}
			if passed := t.Run(fmt.Sprintf("b%02d-%s", block, job.Profile), func(t *testing.T) {
				plan, err := load.BuildPilotPlan(fixture, p, load.EqualByteRate, "constant", job.RunID, job.Seed, c, configHash, 0)
				if err != nil {
					t.Fatal(err)
				}
				file02ExecuteQA(t, c, configPath, configHash, fixture, p, filepath.Join(root, "runs", job.RunID), plan, map[string]any{
					"schema": "pluto.file02.r07_run.v1", "scope": "R_CONFIRMATORY", "case_id": "R07",
					"profile": job.Profile, "block": job.Block, "order": job.Order, "seed": job.Seed,
					"recorder_mode": "ON", "commit_observer_mode": "block_scan",
				})
			}); !passed {
				t.Fatal("R07 stopped after failed arm; no replacement")
			}
			time.Sleep(2 * time.Second)
		}
	}
}
