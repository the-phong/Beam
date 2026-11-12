//go:build integration && fourval

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

func TestFile02T06PrecisionPilot(t *testing.T) {
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("FILE02_T06_PRECISION_ROOT")
	if root == "" {
		t.Fatal("FILE02_T06_PRECISION_ROOT is required")
	}
	allowed, err := filepath.Abs("../../research/artifacts/file02")
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	parent, parentErr := os.Stat(filepath.Dir(abs))
	base, baseErr := os.Stat(allowed)
	info, rootErr := os.Lstat(abs)
	if parentErr != nil || baseErr != nil || rootErr != nil || !os.SameFile(parent, base) || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("FILE02_T06_PRECISION_ROOT must be a private directory directly below ignored research/artifacts/file02")
	}
	raw, err := os.ReadFile(filepath.Join(root, "schedule.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schedule struct {
		Schema string              `json:"schema"`
		Jobs   []file02Common06Job `json:"jobs"`
	}
	if err := json.Unmarshal(raw, &schedule); err != nil || schedule.Schema != "pluto.file02.t06_precision_schedule.v1" || len(schedule.Jobs) != 48 {
		t.Fatal("invalid 48-run precision schedule")
	}
	profiles := map[string]researchv2.Profile{"C": researchv2.ProfileC, "CP": researchv2.ProfileCP,
		"P": researchv2.ProfileP, "H": researchv2.ProfileH}
	for index, job := range schedule.Jobs {
		index, job := index, job
		p, ok := profiles[job.Profile]
		if !ok || job.Block != index/4 || job.Order != index%4 || job.Seed == 0 || job.RunID == "" {
			t.Fatal("invalid QA order or seed")
		}
		if passed := t.Run(fmt.Sprintf("%02d-%s", index, job.Profile), func(t *testing.T) {
			fixture, err := load.NewPilotFixture("pluto-file02-t06-precision", 241)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := load.BuildPrecisionPilotPlan(fixture, p, job.RunID, job.Seed, c, configHash)
			if err != nil {
				t.Fatal(err)
			}
			file02ExecuteQA(t, c, configPath, configHash, fixture, p,
				filepath.Join(root, "runs", job.RunID), plan, map[string]any{
					"schema": "pluto.file02.t06_precision_run.v1", "profile": job.Profile,
					"block": job.Block, "order": job.Order, "seed": job.Seed,
					"commit_observer_mode": "block_scan",
				})
		}); !passed {
			t.Fatal("precision pilot stopped after failed run")
		}
	}
}
