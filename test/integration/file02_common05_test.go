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

// TestFile02Common05MatchedPressure holds both offered tx/s and wire bytes/s
// constant while changing only the added pressure's validity and crypto path.
func TestFile02Common05MatchedPressure(t *testing.T) {
	configPath, err := filepath.Abs("../../research/config/s06_pilot.json")
	if err != nil {
		t.Fatal(err)
	}
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	root := os.Getenv("FILE02_COMMON05_ROOT")
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
			t.Fatal("FILE02_COMMON05_ROOT must be a private directory directly below ignored research/artifacts/file02")
		}
		root = abs
	}
	profiles := []struct {
		name string
		code researchv2.Profile
	}{
		{"C", researchv2.ProfileC}, {"CP", researchv2.ProfileCP},
		{"P", researchv2.ProfileP}, {"H", researchv2.ProfileH},
	}
	for i, profile := range profiles {
		i, profile := i, profile
		t.Run(profile.name, func(t *testing.T) {
			fixture, err := load.NewPilotFixture("pluto-file02-common05", 16)
			if err != nil {
				t.Fatal(err)
			}
			attack, err := load.BuildPilotPlan(fixture, profile.code, load.FixedLegitimatePlusAdversary,
				"constant", fmt.Sprintf("common05-%s-attack", profile.name), int64(1305+i), c, configHash, 0)
			if err != nil {
				t.Fatal(err)
			}
			benign, err := load.BuildPilotPlan(fixture, profile.code, load.EqualTxRate,
				"constant", fmt.Sprintf("common05-%s-benign", profile.name), int64(1305+i), c, configHash, 2)
			if err != nil {
				t.Fatal(err)
			}
			if len(benign.Events) != len(attack.Events) {
				t.Fatal("pair schedule length differs")
			}
			for j, event := range attack.Events {
				if benign.Events[j].OffsetNS != event.OffsetNS || benign.Events[j].AccountSlot != event.AccountSlot {
					t.Fatal("pair schedule differs")
				}
				if event.Traffic == "legitimate" {
					benign.Events[j] = event
				}
			}
			if err := benign.Validate(c, configHash); err != nil {
				t.Fatal(err)
			}
			for _, pair := range []struct {
				name string
				plan load.Plan
			}{{"benign", benign}, {"attack", attack}} {
				pair := pair
				t.Run(pair.name, func(t *testing.T) {
					file02ExecuteQA(t, c, configPath, configHash, fixture, profile.code,
						filepath.Join(root, "profile-"+profile.name, pair.name), pair.plan, map[string]any{
							"schema": "pluto.file02.common05_run.v1", "profile": profile.name, "condition": pair.name,
						})
				})
			}
		})
	}
}
