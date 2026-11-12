package researchv2app_test

import (
	"encoding/hex"
	"path/filepath"
	"reflect"
	"testing"

	researchv2corpus "example.com/pluto/internal/experiment/corpus/profiles"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

// TestFile02T03LateInvalidStageOrdering ensures late ML rejection on H
// actually passes the secp stage, while earlier failures stop before it.
func TestFile02T03LateInvalidStageOrdering(t *testing.T) {
	corpus, err := researchv2corpus.Read(filepath.Join("..", "..", "..", "..", "research", "corpus", "s04_public_corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := hex.DecodeString(corpus.GenesisHex)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]researchv2.Stage{
		"p4-bad-ml": {researchv2.StageStructural, researchv2.StageProfile, researchv2.StageChain,
			researchv2.StageAmount, researchv2.StageBinding, researchv2.StageSecp, researchv2.StageML},
		"p4-bad-proof": {researchv2.StageStructural, researchv2.StageProfile, researchv2.StageChain,
			researchv2.StageAmount, researchv2.StageBinding, researchv2.StageSecp},
		"p4-unbound-sender": {researchv2.StageStructural, researchv2.StageProfile, researchv2.StageChain,
			researchv2.StageAmount, researchv2.StageBinding},
		"p2-nonzero-pad": {researchv2.StageStructural},
	}
	seen := make(map[string]bool)
	for _, item := range corpus.Cases {
		stages, ok := want[item.CaseID]
		if !ok {
			continue
		}
		state, err := researchv2.ParseGenesis(genesis, corpus.ChainID)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := hex.DecodeString(item.BytesHex)
		if err != nil {
			t.Fatal(err)
		}
		var visited []researchv2.Stage
		next, result, err := researchv2.EvaluateTrustedRecorded(state, item.Profile, raw,
			func(stage researchv2.Stage) { visited = append(visited, stage) })
		if err != nil || result.Code != item.ExpectedCode || result.LastStage != item.Stage ||
			!reflect.DeepEqual(visited, stages) || next.AppHash() != state.AppHash() {
			t.Fatalf("%s: result=%+v visited=%v error=%v", item.CaseID, result, visited, err)
		}
		seen[item.CaseID] = true
	}
	if len(seen) != len(want) {
		t.Fatalf("missing corpus cases: got %v", seen)
	}
}
