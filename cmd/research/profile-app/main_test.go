package main

import (
	"strings"
	"testing"
)

func TestParseRequiresLoopbackAndKnownProfile(t *testing.T) {
	base := []string{"-db", "/private/db", "-genesis", "/private/genesis.bin",
		"-chain-id", "pluto-x05", "-profile", "H", "-listen", "tcp://127.0.0.1:26758",
		"-resources", "/private/resources.jsonl", "-events", "/private/events.jsonl"}
	if _, err := parse(base); err != nil {
		t.Fatal(err)
	}
	if _, err := parse(append(append([]string(nil), base...), "-stage-cpu")); err == nil {
		t.Fatal("stage CPU profiling without recorder accepted")
	}
	if _, err := parse(append(append([]string(nil), base...), "-recorder", "-stage-cpu")); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]string{
		{"-listen", "tcp://0.0.0.0:26758"},
		{"-listen", "tcp://127.0.0.1:0"},
		{"-profile", "UNKNOWN"},
	} {
		args := append([]string(nil), base...)
		for i := 0; i < len(args); i += 2 {
			if args[i] == bad[0] {
				args[i+1] = bad[1]
			}
		}
		if _, err := parse(args); err == nil {
			t.Fatalf("accepted invalid argument %v", bad)
		}
	}
}

func TestSampleIdentifiesDedicatedProcess(t *testing.T) {
	s := sample(3)
	if s.PID < 1 || s.Schema == "" || s.UserCPUNS < 0 || s.SystemCPUNS < 0 || s.Threads < 1 || s.RecorderDrops != 3 || !strings.HasSuffix(s.UTC, "Z") {
		t.Fatalf("invalid resource sample: %+v", s)
	}
}
