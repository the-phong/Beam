//go:build integration && fourval

package integration

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	"example.com/pluto/internal/experiment/load"
	researchv2 "example.com/pluto/internal/experiment/profiles"
)

func file02A06Oracle(t *testing.T, pilot *s06Network, fixture *load.PilotFixture,
	plan load.Plan, result load.Result) s07OracleRecord {
	t.Helper()
	if result.RunID != plan.RunID || result.ConfigSHA256 != plan.ConfigSHA256 ||
		result.GenesisSHA256 != plan.GenesisSHA256 || len(plan.Events) != 81 ||
		len(result.Events) != 81 || result.ActualSendCount != 81 {
		t.Fatal("A06 plan/result identity or count")
	}
	common := int64(1 << 62)
	for i := range pilot.n.nodes {
		h, _, err := pilot.n.height(i)
		if err != nil || h < 8 {
			t.Fatalf("A06 validator %d height %d: %v", i, h, err)
		}
		if h < common {
			common = h
		}
	}
	var reference map[string]int
	validatorHashes := make([]map[string]int, 0, 4)
	var blockID, appHash string
	for i := range pilot.n.nodes {
		seen := map[string]int{}
		for height := int64(1); height <= common; height++ {
			b, a, txs, err := pilot.n.block(i, height)
			if err != nil {
				t.Fatal(err)
			}
			if height == common {
				if i == 0 {
					blockID, appHash = b, a
				} else if b != blockID || a != appHash {
					t.Fatal("A06 same-height BlockID/AppHash divergence")
				}
			}
			for _, encoded := range txs {
				raw, err := base64.StdEncoding.DecodeString(encoded)
				if err != nil {
					t.Fatal(err)
				}
				hash := sha256.Sum256(raw)
				seen[hex.EncodeToString(hash[:])]++
			}
		}
		if i == 0 {
			reference = seen
		} else if !s07EqualCounts(reference, seen) {
			t.Fatal("A06 validator committed sets diverge")
		}
		validatorHashes = append(validatorHashes, seen)
	}
	expected := map[string]bool{}
	validSlots := map[int]bool{}
	var rejected int
	for i, event := range plan.Events {
		actual := result.Events[i]
		if actual.ID != event.ID || actual.ActualOffsetNS == nil || actual.ActualSendMonotonicNS == nil {
			t.Fatalf("A06 send missing %s", event.ID)
		}
		switch event.Scenario {
		case "fixed_legitimate", "benign_background":
			if actual.ActualCode == nil || *actual.ActualCode != 0 || actual.SubmissionError != "" {
				t.Fatalf("A06 valid event %s rejected: %+v", event.ID, actual)
			}
			expected[event.SHA256] = true
			validSlots[event.AccountSlot] = true
		case "A06_valid_replay":
			if !(actual.ActualCode != nil && (*actual.ActualCode == 0 || *actual.ActualCode == uint32(researchv2.CodeInvalidNonce)) ||
				actual.Reason == "duplicate_cache") {
				t.Fatalf("A06 replay result unclassified %s: %+v", event.ID, actual)
			}
			if actual.ActualCode == nil || *actual.ActualCode != 0 {
				rejected++
			}
			expected[event.SHA256] = true
			validSlots[event.AccountSlot] = true
		case "A06_bad_secp_distinct", "A06_bad_secp_repeated":
			if !(actual.ActualCode != nil && *actual.ActualCode == uint32(researchv2.CodeBadSecpSignature) ||
				event.Scenario == "A06_bad_secp_repeated" && actual.Reason == "duplicate_cache") {
				t.Fatalf("A06 invalid result unclassified %s: %+v", event.ID, actual)
			}
			rejected++
		default:
			t.Fatalf("A06 unknown scenario %s", event.Scenario)
		}
	}
	if len(reference) != len(expected) {
		t.Fatalf("A06 committed hash count %d expected %d", len(reference), len(expected))
	}
	for hash := range expected {
		if reference[hash] != 1 {
			t.Fatalf("A06 hash %s committed %d times", hash, reference[hash])
		}
	}
	for i := range pilot.n.nodes {
		balance, nonce := s04Account(t, pilot.n, i, fixture.Receiver.Binding.Account)
		if balance != uint64(len(expected)) || nonce != 0 {
			t.Fatalf("A06 validator %d receiver state %d/%d expected %d", i, balance, nonce, len(expected))
		}
		for slot, sender := range fixture.Senders {
			balance, nonce = s04Account(t, pilot.n, i, sender.Binding.Account)
			if validSlots[slot] && (balance != 1 || nonce != 1) || !validSlots[slot] && (balance != 2 || nonce != 0) {
				t.Fatalf("A06 validator %d sender %d state %d/%d", i, slot, balance, nonce)
			}
		}
	}
	return s07OracleRecord{Schema: "pluto.file02.a06.oracle.v1", RunID: plan.RunID,
		Status: "OK", ObservedUTC: time.Now().UTC().Format(time.RFC3339Nano),
		CommonHeight: common, BlockID: blockID, AppHash: appHash,
		ValidCommitted: len(expected), InvalidRejected: rejected,
		CommittedHashes: reference, ValidatorCommittedHashes: validatorHashes}
}
