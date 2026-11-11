package b1state_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"

	"example.com/pluto/internal/classical/state"

	"example.com/pluto/internal/hybrid/state"
	"example.com/pluto/internal/hybrid/testutil"
	"example.com/pluto/internal/hybrid/tx"
)

// The committed vector contains only public keys, genesis bytes, transaction
// bytes and hashes. The signing keys were generated ephemerally and discarded.
func TestHybridPublicGoldenVector(t *testing.T) {
	raw, err := os.ReadFile("../tx/testdata/hybrid_public_vector.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector struct {
		ChainID        string `json:"chain_id"`
		Genesis        string `json:"genesis_hex"`
		Transaction    string `json:"transaction_hex"`
		TransactionSHA string `json:"transaction_sha256"`
		TransactionID  string `json:"transaction_id_hex"`
		Digest         string `json:"signed_digest_hex"`
		H0             string `json:"h0_apphash_hex"`
		After          string `json:"after_transfer_candidate_apphash_hex"`
	}
	if err := json.Unmarshal(raw, &vector); err != nil {
		t.Fatal(err)
	}
	genesis, err := hex.DecodeString(vector.Genesis)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := hex.DecodeString(vector.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	checksum := sha256.Sum256(wire)
	if hex.EncodeToString(checksum[:]) != vector.TransactionSHA {
		t.Fatal("golden transaction checksum changed")
	}
	id, err := b1tx.TxID(wire)
	if err != nil || hex.EncodeToString(id[:]) != vector.TransactionID {
		t.Fatal("golden transaction ID changed")
	}
	state, err := b1state.ParseGenesis(genesis, vector.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	h0 := state.AppHash()
	if hex.EncodeToString(h0[:]) != vector.H0 {
		t.Fatal("golden H0 AppHash changed")
	}
	tx, err := b1tx.Parse(wire)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := b1tx.SigningDigest(tx)
	if err != nil || hex.EncodeToString(digest[:]) != vector.Digest {
		t.Fatal("golden signed digest changed")
	}
	next, code, err := b1state.EvaluateTrusted(state, wire)
	if err != nil || code != b1tx.CodeOK {
		t.Fatalf("golden proofs failed code=%d err=%v", code, err)
	}
	hash := next.AppHash()
	if hex.EncodeToString(hash[:]) != vector.After {
		t.Fatal("golden post-transfer AppHash changed")
	}
	sender, _ := next.Account(b0state.Address(tx.Sender))
	receiver, _ := next.Account(b0state.Address(tx.Receiver))
	if sender.Balance != 75 || sender.Nonce != 1 || receiver.Balance != 25 || receiver.Nonce != 0 || next.TotalSupply() != 100 {
		t.Fatal("golden account state changed")
	}
	t.Logf("B1 public golden vector tx_sha256=%x H0=%x after=%x", checksum, h0, hash)
}

func TestHybridH01ToH18(t *testing.T) {
	fixture, err := b1test.New("pluto-b1-test")
	if err != nil {
		t.Fatal(err)
	}
	state, err := b1state.ParseGenesis(fixture.Genesis, fixture.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := fixture.Signed(25, 0)
	if err != nil {
		t.Fatal(err)
	}
	other, err := fixture.Signed(26, 0)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := b1tx.Parse(valid)
	if err != nil {
		t.Fatal(err)
	}
	otherParsed, err := b1tx.Parse(other)
	if err != nil {
		t.Fatal(err)
	}
	encode := func(tx b1tx.Transaction) []byte {
		t.Helper()
		raw, err := b1tx.Encode(tx)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	mutate := func(fn func(*b1tx.Transaction)) []byte { copy := parsed; fn(&copy); return encode(copy) }
	foreign, err := b1test.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	var cases = []struct {
		id                     string
		raw                    []byte
		want                   b1tx.Code
		reachedSecp, reachedML bool
	}{
		{"H01", valid, b1tx.CodeOK, true, true},
		{"H02", mutate(func(x *b1tx.Transaction) { x.SecpSignature = [64]byte{} }), b1tx.CodeBadSecpSignature, true, false},
		{"H03", mutate(func(x *b1tx.Transaction) { x.MLSignature = [len(parsed.MLSignature)]byte{} }), b1tx.CodeBadMLSignature, true, true},
		{"H04", mutate(func(x *b1tx.Transaction) { x.SecpSignature[0] ^= 0x80 }), b1tx.CodeBadSecpSignature, true, false},
		{"H05", mutate(func(x *b1tx.Transaction) { x.MLSignature[0] ^= 0x80 }), b1tx.CodeBadMLSignature, true, true},
		{"H06", mutate(func(x *b1tx.Transaction) { x.SecpSignature = otherParsed.SecpSignature }), b1tx.CodeBadSecpSignature, true, false},
		{"H07", mutate(func(x *b1tx.Transaction) { x.MLSignature = otherParsed.MLSignature }), b1tx.CodeBadMLSignature, true, true},
		{"H08", mutate(func(x *b1tx.Transaction) { x.MLKey = foreign.Binding.MLKey }), b1tx.CodeBadBinding, false, false},
		{"H09", mutate(func(x *b1tx.Transaction) { x.SecpKey = foreign.Binding.SecpKey }), b1tx.CodeBadBinding, false, false},
		{"malformed-secp-key", mutate(func(x *b1tx.Transaction) { x.SecpKey[0] = 4 }), b1tx.CodeBadSecpKey, false, false},
		{"unregistered-ml-bytes", mutate(func(x *b1tx.Transaction) {
			for i := range x.MLKey {
				x.MLKey[i] = 0xff
			}
		}), b1tx.CodeBadBinding, false, false},
		{"H10", mutate(func(x *b1tx.Transaction) { x.ChainID = "other-chain" }), b1tx.CodeWrongChain, false, false},
		{"H11", append(bytes.Clone(valid[:10]), append([]byte{2}, valid[11:]...)...), b1tx.CodeMalformed, false, false},
		{"H11-b0-magic", func() []byte { raw := bytes.Clone(valid); copy(raw[:8], "PLUTOTX1"); return raw }(), b1tx.CodeMalformed, false, false},
		{"H12", func() []byte { raw := bytes.Clone(valid); raw[9]++; return raw }(), b1tx.CodeMalformed, false, false},
		{"H13", append(bytes.Clone(valid), 0), b1tx.CodeMalformed, false, false},
		{"H13-conflicting-length", func() []byte { raw := bytes.Clone(valid); raw[11]++; return raw }(), b1tx.CodeMalformed, false, false},
		{"H13-invalid-chain-byte", func() []byte { raw := bytes.Clone(valid); raw[12] = '/'; return raw }(), b1tx.CodeMalformed, false, false},
		{"H14-truncated", bytes.Clone(valid[:len(valid)-1]), b1tx.CodeMalformed, false, false},
		{"H14-oversized", make([]byte, b1tx.MaxWireSize+1), b1tx.CodeMalformed, false, false},
		{"H15", mutate(func(x *b1tx.Transaction) { x.Nonce = 1 }), b1tx.CodeBadSecpSignature, true, false},
		{"H16", mutate(func(x *b1tx.Transaction) { x.Amount = 101 }), b1tx.CodeBadSecpSignature, true, false},
	}
	// H15/H16 must use valid proofs for their changed intents to reach state.
	cases[len(cases)-2].raw, err = fixture.Signed(25, 1)
	if err != nil {
		t.Fatal(err)
	}
	cases[len(cases)-2].want = b1tx.CodeInvalidNonce
	cases[len(cases)-2].reachedML = true
	cases[len(cases)-1].raw, err = fixture.Signed(101, 0)
	if err != nil {
		t.Fatal(err)
	}
	cases[len(cases)-1].want = b1tx.CodeInsufficientBalance
	cases[len(cases)-1].reachedML = true
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			before := state.AppHash()
			next, code, err := b1state.EvaluateTrusted(state, tc.raw)
			if err != nil || code != tc.want {
				t.Fatalf("code=%d want=%d err=%v", code, tc.want, err)
			}
			if code == b1tx.CodeOK {
				sender, _ := next.Account(fixture.Sender.Binding.Address)
				receiver, _ := next.Account(fixture.Receiver.Binding.Address)
				if sender.Balance != 75 || sender.Nonce != 1 || receiver.Balance != 25 || receiver.Nonce != 0 || next.TotalSupply() != 100 {
					t.Fatalf("bad state sender=%+v receiver=%+v", sender, receiver)
				}
			} else if next.AppHash() != before {
				t.Fatal("invalid transaction mutated candidate")
			}
			t.Logf("%s raw=%x code=%d secp_verify_reached=%t ml_verify_reached=%t candidate_changed=%t", tc.id, tc.raw, code, tc.reachedSecp, tc.reachedML, code == b1tx.CodeOK)
		})
	}
	unregisteredReceiver, err := fixture.SignedBy(fixture.Sender, foreign.Binding.Address, 25, 0)
	if err != nil {
		t.Fatal(err)
	}
	unregisteredState, unregisteredCode, err := b1state.EvaluateTrusted(state, unregisteredReceiver)
	if err != nil || unregisteredCode != b1tx.CodeBadBinding || unregisteredState.AppHash() != state.AppHash() {
		t.Fatalf("unregistered recipient accepted code=%d err=%v", unregisteredCode, err)
	}
	for _, tc := range []struct {
		name string
		raw  []byte
		want b1tx.Code
	}{
		{"chain-before-proof", mutate(func(x *b1tx.Transaction) {
			x.ChainID = "other-chain"
			x.SecpSignature = [64]byte{}
			x.MLSignature = [len(parsed.MLSignature)]byte{}
		}), b1tx.CodeWrongChain},
		{"binding-before-proof", mutate(func(x *b1tx.Transaction) { x.MLKey = foreign.Binding.MLKey; x.SecpSignature = [64]byte{} }), b1tx.CodeBadBinding},
		{"classical-before-ml", mutate(func(x *b1tx.Transaction) {
			x.SecpSignature = [64]byte{}
			x.MLSignature = [len(parsed.MLSignature)]byte{}
		}), b1tx.CodeBadSecpSignature},
	} {
		_, code, err := b1state.EvaluateTrusted(state, tc.raw)
		if err != nil || code != tc.want {
			t.Fatalf("precedence %s got=%d want=%d err=%v", tc.name, code, tc.want, err)
		}
	}
	first, code, err := b1state.EvaluateTrusted(state, valid)
	if err != nil || code != b1tx.CodeOK {
		t.Fatalf("H17 first: %d %v", code, err)
	}
	second, code, err := b1state.EvaluateTrusted(first, valid)
	if err != nil || code != b1tx.CodeInvalidNonce || second.AppHash() != first.AppHash() {
		t.Fatalf("H17 replay: %d %v", code, err)
	}
	t.Logf("H17 raw=%x first=%d replay=%d effect_once=true", valid, b1tx.CodeOK, code)
	sequence, err := fixture.Signed(10, 2)
	if err != nil {
		t.Fatal(err)
	}
	final, codes, err := b1state.SimulateOrderedTrusted(state, [][]byte{valid, sequence})
	if err != nil || len(codes) != 2 || codes[0] != b1tx.CodeOK || codes[1] != b1tx.CodeInvalidNonce || final.AppHash() != first.AppHash() {
		t.Fatalf("H18 sequence codes=%v err=%v", codes, err)
	}
	t.Logf("H18 raw_first=%x raw_second=%x codes=%v one_transition=true", valid, sequence, codes)
	// Pure ML-DSA may create distinct valid proofs for one intent. TxIDs can
	// differ, but the committed nonce still permits the effect only once.
	variant := parsed
	digest, err := b1tx.SigningDigest(variant)
	if err != nil {
		t.Fatal(err)
	}
	if err := mldsa44.SignTo(fixture.Sender.ML, digest[:], []byte(b1tx.MLContext), true, variant.MLSignature[:]); err != nil {
		t.Fatal(err)
	}
	variantRaw := encode(variant)
	if bytes.Equal(variantRaw, valid) {
		t.Fatal("randomized ML proof did not change bytes")
	}
	originalID, err := b1tx.TxID(valid)
	if err != nil {
		t.Fatal(err)
	}
	variantID, err := b1tx.TxID(variantRaw)
	if err != nil || variantID == originalID {
		t.Fatal("malleated transaction ID not distinct")
	}
	_, variantCode, err := b1state.EvaluateTrusted(state, variantRaw)
	if err != nil || variantCode != b1tx.CodeOK {
		t.Fatalf("valid randomized proof rejected code=%d err=%v", variantCode, err)
	}
	variantFinal, variantCodes, err := b1state.SimulateOrderedTrusted(state, [][]byte{valid, variantRaw})
	if err != nil || len(variantCodes) != 2 || variantCodes[0] != b1tx.CodeOK || variantCodes[1] != b1tx.CodeInvalidNonce || variantFinal.AppHash() != first.AppHash() {
		t.Fatalf("malleated replay codes=%v err=%v", variantCodes, err)
	}
	t.Logf("hybrid proof malleability distinct_txids=%x/%x sequence_codes=%v effect_once=true", originalID, variantID, variantCodes)
}

func TestHybridGenesisAndRecordIdentity(t *testing.T) {
	fixture, err := b1test.New("pluto-b1-test")
	if err != nil {
		t.Fatal(err)
	}
	state, err := b1state.ParseGenesis(fixture.Genesis, fixture.ChainID)
	if err != nil || state.Height() != 0 || state.TotalSupply() != 100 {
		t.Fatalf("genesis: %v", err)
	}
	if _, err := b1state.ParseGenesis(fixture.Genesis, "other-chain"); err == nil {
		t.Fatal("cross-chain genesis accepted")
	}
	bad := bytes.Clone(fixture.Genesis)
	bad[len(bad)-1] ^= 0x80
	if _, err := b1state.ParseGenesis(bad, fixture.ChainID); err == nil {
		t.Fatal("key/address conflict accepted")
	}
}

func FuzzHybridDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte(b1tx.Magic))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > b1tx.MaxWireSize+1 {
			return
		}
		parsed, err := b1tx.Parse(data)
		if err != nil {
			return
		}
		reencoded, err := b1tx.Encode(parsed)
		if err != nil || !bytes.Equal(data, reencoded) {
			t.Fatal("noncanonical round trip")
		}
	})
}

func FuzzHybridEvaluate(f *testing.F) {
	raw, err := os.ReadFile("../tx/testdata/hybrid_public_vector.json")
	if err != nil {
		f.Fatal(err)
	}
	var vector struct {
		ChainID     string `json:"chain_id"`
		Genesis     string `json:"genesis_hex"`
		Transaction string `json:"transaction_hex"`
	}
	if err := json.Unmarshal(raw, &vector); err != nil {
		f.Fatal(err)
	}
	genesis, err := hex.DecodeString(vector.Genesis)
	if err != nil {
		f.Fatal(err)
	}
	wire, err := hex.DecodeString(vector.Transaction)
	if err != nil {
		f.Fatal(err)
	}
	state, err := b1state.ParseGenesis(genesis, vector.ChainID)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(wire)
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, candidate []byte) {
		if len(candidate) > b1tx.MaxWireSize+1 {
			return
		}
		next, code, err := b1state.EvaluateTrusted(state, candidate)
		if err != nil {
			t.Fatalf("public input caused internal error: %v", err)
		}
		if code != b1tx.CodeOK && next.AppHash() != state.AppHash() {
			t.Fatal("rejected mutation changed state")
		}
		if code == b1tx.CodeOK && next.TotalSupply() != state.TotalSupply() {
			t.Fatal("accepted mutation changed supply")
		}
	})
}
