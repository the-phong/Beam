package researchv2_test

import (
	"bytes"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/profiles"
	"example.com/pluto/internal/profiles/testfixture"
)

func fixture(t *testing.T) *researchv2test.Fixture {
	t.Helper()
	f, err := researchv2test.New("rv2-profile-test")
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestFourProfilesSameTransferAndExactSize(t *testing.T) {
	f := fixture(t)
	profiles := []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH}
	var reference [32]byte
	var sizes [5]int
	for _, p := range profiles {
		s, err := researchv2.ParseGenesis(f.Genesis, f.ChainID)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.Signed(p, 7, 0)
		if err != nil {
			t.Fatal(err)
		}
		wantSize, err := p.WireSize(f.ChainID)
		if err != nil || len(raw) != wantSize {
			t.Fatalf("profile %d wire size %d want %d: %v", p, len(raw), wantSize, err)
		}
		sizes[p] = len(raw)
		tx, err := researchv2.Parse(raw)
		if err != nil {
			t.Fatalf("profile %d parse: %v", p, err)
		}
		reencoded, err := researchv2.Encode(tx)
		if err != nil || !bytes.Equal(raw, reencoded) {
			t.Fatalf("profile %d noncanonical encode: %v", p, err)
		}
		next, result, err := researchv2.Evaluate(s, p, raw)
		if err != nil || result != (researchv2.Result{Code: researchv2.CodeOK, LastStage: researchv2.StageAccepted}) {
			t.Fatalf("profile %d result %+v err %v", p, result, err)
		}
		sender, _ := next.Account(f.Sender.Binding.Account)
		receiver, _ := next.Account(f.Receiver.Binding.Account)
		if sender.Balance != 93 || sender.Nonce != 1 || receiver.Balance != 7 || next.TotalSupply() != 100 {
			t.Fatalf("profile %d state mismatch: %+v %+v", p, sender, receiver)
		}
		if reference == ([32]byte{}) {
			reference = next.AppHash()
		} else if next.AppHash() != reference {
			t.Fatalf("profile %d AppHash differs for same transfer", p)
		}
		id, err := researchv2.TxID(raw)
		if err != nil || id == ([32]byte{}) {
			t.Fatalf("profile %d tx ID", p)
		}
		again, result, err := researchv2.EvaluateTrusted(next, p, raw)
		if err != nil || result != (researchv2.Result{Code: researchv2.CodeInvalidNonce, LastStage: researchv2.StageState}) || again.AppHash() != next.AppHash() {
			t.Fatalf("profile %d duplicate mutated state: %+v %v", p, result, err)
		}
	}
	if sizes[researchv2.ProfileCP] != sizes[researchv2.ProfileH] || sizes[researchv2.ProfileC] >= sizes[researchv2.ProfileCP] || sizes[researchv2.ProfileP] >= sizes[researchv2.ProfileH] {
		t.Fatalf("wrong size relationships: %+v", sizes)
	}
}

func TestCPAndHExactBytesAcrossChainIDBounds(t *testing.T) {
	for _, chain := range []string{"a", string(bytes.Repeat([]byte{'z'}, researchv2.MaxChainID))} {
		cp, err := researchv2.Encode(researchv2.Transaction{Profile: researchv2.ProfileCP, ChainID: chain})
		if err != nil {
			t.Fatal(err)
		}
		h, err := researchv2.Encode(researchv2.Transaction{Profile: researchv2.ProfileH, ChainID: chain})
		if err != nil {
			t.Fatal(err)
		}
		if len(cp) != len(h) {
			t.Fatalf("chain length %d: CP=%d H=%d", len(chain), len(cp), len(h))
		}
		if _, err := researchv2.Parse(cp); err != nil {
			t.Fatal(err)
		}
		if _, err := researchv2.Parse(h); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProfileNegativeBoundariesAndReachedStages(t *testing.T) {
	f := fixture(t)
	state, err := researchv2.ParseGenesis(f.Genesis, f.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH} {
		t.Run(string(rune('0'+p)), func(t *testing.T) {
			valid, err := f.Signed(p, 1, 0)
			if err != nil {
				t.Fatal(err)
			}
			check := func(name string, raw []byte, want researchv2.Result) {
				t.Helper()
				next, result, err := researchv2.EvaluateTrusted(state, p, raw)
				if err != nil || result != want || next.AppHash() != state.AppHash() {
					t.Fatalf("%s profile %d got %+v err %v; rejected state changed=%v", name, p, result, err, next.AppHash() != state.AppHash())
				}
			}
			check("truncated", valid[:len(valid)-1], researchv2.Result{Code: researchv2.CodeMalformed, LastStage: researchv2.StageStructural})
			check("trailing", append(bytes.Clone(valid), 0), researchv2.Result{Code: researchv2.CodeMalformed, LastStage: researchv2.StageStructural})
			badMagic := bytes.Clone(valid)
			badMagic[0] ^= 1
			check("magic", badMagic, researchv2.Result{Code: researchv2.CodeMalformed, LastStage: researchv2.StageStructural})
			wrongChain := *f
			wrongChain.ChainID = "other-chain"
			wrongRaw, err := wrongChain.Signed(p, 1, 0)
			if err != nil {
				t.Fatal(err)
			}
			check("chain", wrongRaw, researchv2.Result{Code: researchv2.CodeWrongChain, LastStage: researchv2.StageChain})
			zero, err := f.Signed(p, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			check("amount", zero, researchv2.Result{Code: researchv2.CodeZeroAmount, LastStage: researchv2.StageAmount})
			unknown, err := researchv2test.NewIdentity()
			if err != nil {
				t.Fatal(err)
			}
			wrongBinding, err := f.SignedBy(p, unknown, unknown.Binding.Account, f.Receiver.Binding.Account, 1, 0)
			if err != nil {
				t.Fatal(err)
			}
			check("binding", wrongBinding, researchv2.Result{Code: researchv2.CodeBadBinding, LastStage: researchv2.StageBinding})
			wrongKey, err := f.SignedBy(p, unknown, f.Sender.Binding.Account, f.Receiver.Binding.Account, 1, 0)
			if err != nil {
				t.Fatal(err)
			}
			keyStage := researchv2.StageSecp
			keyCode := researchv2.CodeBadSecpSignature
			if p == researchv2.ProfileP {
				keyStage, keyCode = researchv2.StageML, researchv2.CodeBadMLSignature
			}
			check("unregistered signing key", wrongKey, researchv2.Result{Code: keyCode, LastStage: keyStage})
			wrongNonce, err := f.Signed(p, 1, 1)
			if err != nil {
				t.Fatal(err)
			}
			check("nonce", wrongNonce, researchv2.Result{Code: researchv2.CodeInvalidNonce, LastStage: researchv2.StageState})
			tooMuch, err := f.Signed(p, 101, 0)
			if err != nil {
				t.Fatal(err)
			}
			check("balance", tooMuch, researchv2.Result{Code: researchv2.CodeInsufficientBalance, LastStage: researchv2.StageState})
			if p.UsesSecp() {
				bad := bytes.Clone(valid)
				bad[len(bad)-pProofTail(p)-1] ^= 1
				check("secp proof", bad, researchv2.Result{Code: researchv2.CodeBadSecpSignature, LastStage: researchv2.StageSecp})
				tx, err := researchv2.Parse(valid)
				if err != nil {
					t.Fatal(err)
				}
				var s secp256k1.ModNScalar
				if s.SetByteSlice(tx.SecpSignature[32:]) || s.IsZero() || s.IsOverHalfOrder() {
					t.Fatal("fixture secp signature is not canonical")
				}
				s.Negate()
				if !s.IsOverHalfOrder() {
					t.Fatal("negated signature is not high-S")
				}
				var high [32]byte
				s.PutBytes(&high)
				copy(tx.SecpSignature[32:], high[:])
				highRaw, err := researchv2.Encode(tx)
				if err != nil {
					t.Fatal(err)
				}
				check("high-S secp proof", highRaw, researchv2.Result{Code: researchv2.CodeBadSecpSignature, LastStage: researchv2.StageSecp})
			}
			if p.UsesML() {
				bad := bytes.Clone(valid)
				bad[len(bad)-1] ^= 1
				check("ML proof", bad, researchv2.Result{Code: researchv2.CodeBadMLSignature, LastStage: researchv2.StageML})
			}
			if p == researchv2.ProfileCP {
				bad := bytes.Clone(valid)
				bad[len(bad)-1] = 1
				check("padding", bad, researchv2.Result{Code: researchv2.CodeMalformed, LastStage: researchv2.StageStructural})
			}
			other := researchv2.ProfileH
			if p == researchv2.ProfileH {
				other = researchv2.ProfileC
			}
			check("profile", validAsProfile(t, f, other), researchv2.Result{Code: researchv2.CodeWrongProfile, LastStage: researchv2.StageProfile})
		})
	}
}

func pProofTail(p researchv2.Profile) int {
	if p == researchv2.ProfileCP || p == researchv2.ProfileH {
		return researchv2.MLProofSize
	}
	return 0
}
func validAsProfile(t *testing.T, f *researchv2test.Fixture, p researchv2.Profile) []byte {
	t.Helper()
	raw, err := f.Signed(p, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestGenesisRejectsBindingAndPreservesAccountIDs(t *testing.T) {
	f := fixture(t)
	base, err := researchv2.ParseGenesis(f.Genesis, f.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if base.RegistryHash() == ([32]byte{}) {
		t.Fatal("missing registry commitment")
	}
	allocations := []researchv2.Allocation{{Binding: f.Sender.Binding, Balance: 100}, {Binding: f.Receiver.Binding, Balance: 0}}
	if bytes.Compare(allocations[0].Binding.Account[:], allocations[1].Binding.Account[:]) > 0 {
		allocations[0], allocations[1] = allocations[1], allocations[0]
	}
	bad := append([]researchv2.Allocation(nil), allocations...)
	bad[1].Binding.SecpKey = bad[0].Binding.SecpKey
	if _, err := researchv2.EncodeGenesis(f.ChainID, bad); err == nil {
		t.Fatal("duplicate secp key accepted")
	}
	bad = append([]researchv2.Allocation(nil), allocations...)
	bad[1].Binding.MLKey = bad[0].Binding.MLKey
	if _, err := researchv2.EncodeGenesis(f.ChainID, bad); err == nil {
		t.Fatal("duplicate ML key accepted")
	}
	bad = append([]researchv2.Allocation(nil), allocations...)
	bad[0].Binding.Account = b0state.Address{}
	changedGenesis, err := researchv2.EncodeGenesis(f.ChainID, bad)
	if err != nil {
		t.Fatalf("account ID unexpectedly derived from keys: %v", err)
	}
	changed, err := researchv2.ParseGenesis(changedGenesis, f.ChainID)
	if err != nil || changed.RegistryHash() == base.RegistryHash() {
		t.Fatal("account ID did not affect registry commitment")
	}
	corrupt := bytes.Clone(f.Genesis)
	corrupt[len(corrupt)-len(f.Receiver.Binding.MLKey)-33] = 0
	if _, err := researchv2.ParseGenesis(corrupt, f.ChainID); err == nil {
		t.Fatal("corrupt key accepted")
	}
}
