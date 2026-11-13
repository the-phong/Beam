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

// TestFile02Common04MatchedRates exercises equal-tx and equal-byte regimes
// across all profiles on fresh isolated networks. This is bounded QA only.
func TestFile02Common04MatchedRates(t *testing.T) {
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("FILE02_COMMON04_ROOT")
	if root == "" {
		root = t.TempDir()
	} else {
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
			t.Fatal("FILE02_COMMON04_ROOT must be a private directory directly below ignored research/artifacts/file02")
		}
		root = abs
	}
	fixture, err := load.NewPilotFixture("pluto-file02-common04", 90)
	if err != nil {
		t.Fatal(err)
	}
	profiles := []struct {
		name string
		code researchv2.Profile
	}{
		{"C", researchv2.ProfileC}, {"CP", researchv2.ProfileCP},
		{"P", researchv2.ProfileP}, {"H", researchv2.ProfileH},
	}
	for regimeIndex, regime := range []struct {
		name, code string
		rate       int
	}{
		{"equal-tx", load.EqualTxRate, 2}, {"equal-byte", load.EqualByteRate, 0},
	} {
		regimeIndex, regime := regimeIndex, regime
		t.Run(regime.name, func(t *testing.T) {
			for i, profile := range profiles {
				i, profile := i, profile
				t.Run(profile.name, func(t *testing.T) {
					plan, err := load.BuildPilotPlan(fixture, profile.code, regime.code, "constant",
						fmt.Sprintf("common04-%s-%s", regime.name, profile.name),
						int64(1204+10*regimeIndex+i), c, configHash, regime.rate)
					if err != nil {
						t.Fatal(err)
					}
					file02ExecuteQA(t, c, configPath, configHash, fixture, profile.code,
						filepath.Join(root, regime.name, profile.name), plan, map[string]any{
							"schema": "pluto.file02.common04_run.v1", "profile": profile.name, "regime": regime.name,
						})
				})
			}
		})
	}
}
