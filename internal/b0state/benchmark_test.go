package b0state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"testing"

	"example.com/pluto/internal/tx"
)

func TestTrustedEvaluatorMatchesPhase03CReference(t *testing.T) {
	sender, receiver, valid := baseFixture(t)
	state, err := NewState(chain, 0, 100, []Account{{Address: sender, Balance: 100}})
	if err != nil {
		t.Fatal(err)
	}
	wrong := append([]byte(nil), valid...)
	wrong[12] = 'x'
	zero := signed(t, 1, receiver, 0, 0)
	noncanonical := append([]byte(nil), valid...)
	for i := len(noncanonical) - 64; i < len(noncanonical)-32; i++ {
		noncanonical[i] = 0
	}
	invalidECDSA := append([]byte(nil), valid...)
	invalidECDSA[len(invalidECDSA)-80] ^= 1
	for _, raw := range [][]byte{{1}, wrong, zero, noncanonical, invalidECDSA, valid,
		signed(t, 1, sender, 25, 0), signed(t, 1, receiver, 200, 0), signed(t, 1, receiver, 25, 1)} {
		reference, referenceCode, referenceErr := Evaluate(state, raw)
		integrated, integratedCode, integratedErr := EvaluateTrusted(state, raw)
		if referenceErr != nil || integratedErr != nil || referenceCode != integratedCode || reference.AppHash() != integrated.AppHash() {
			t.Fatalf("different result for %x: reference=%d/%v integrated=%d/%v", raw, referenceCode, referenceErr, integratedCode, integratedErr)
		}
		a, _ := reference.LogicalBytes()
		b, _ := integrated.LogicalBytes()
		if !bytes.Equal(a, b) {
			t.Fatal("different logical state")
		}
	}
	if _, _, err := EvaluateTrusted(State{}, []byte{1}); err != ErrState {
		t.Fatalf("untrusted state accepted: %v", err)
	}
}

// The Phase 03C Evaluate method remains the untouched reference path. Each
// subbenchmark uses the same immutable snapshot and test-only signed bytes.
func benchmarkState(b *testing.B, k int, sender, receiver Address) State {
	b.Helper()
	accounts := []Account{{Address: sender, Balance: 1_000_000}}
	if k > 1 {
		accounts = append(accounts, Account{Address: receiver, Balance: 1_000_000})
	}
	for i := 0; len(accounts) < k; i++ {
		var addr Address
		binary.BigEndian.PutUint32(addr[16:], uint32(i))
		if addr == sender || addr == receiver {
			continue
		}
		accounts = append(accounts, Account{Address: addr, Balance: 1_000_000})
	}
	sort.Slice(accounts, func(i, j int) bool { return string(accounts[i].Address[:]) < string(accounts[j].Address[:]) })
	s, err := NewState(chain, 0, uint64(k)*1_000_000, accounts)
	if err != nil {
		b.Fatal(err)
	}
	return s
}

func BenchmarkB0StateCost(b *testing.B) {
	// baseFixture uses the approved Phase 03B V01 vector; signed uses only
	// explicit public test scalars 1 and 2.
	sender, receiver, valid := baseFixture(b)
	self := signed(b, 1, sender, 25, 0)
	zero := signed(b, 1, receiver, 0, 0)
	wrong := append([]byte(nil), valid...)
	wrong[12] = 'x'
	noncanonical := append([]byte(nil), valid...)
	for i := len(noncanonical) - 64; i < len(noncanonical)-32; i++ {
		noncanonical[i] = 0
	}
	invalidECDSA := append([]byte(nil), valid...)
	invalidECDSA[len(invalidECDSA)-64-16] ^= 1 // signed amount/nonce region
	workloads := []struct {
		name string
		raw  []byte
		code tx.Code
	}{
		{"Malformed", []byte{1}, 1}, {"WrongChain", wrong, 2},
		{"ZeroAmount", zero, 5}, {"NoncanonicalScalar", noncanonical, 4},
		{"InvalidECDSA", invalidECDSA, 4}, {"SelfTransfer", self, 0},
		{"DistinctTransfer", valid, 0},
	}
	for _, k := range []int{1, 64, 4096, 65536} {
		state := benchmarkState(b, k, sender, receiver)
		logicalBytes := 26 + len(chain) + 36*k
		for _, path := range []struct {
			name     string
			evaluate func(State, []byte) (State, tx.Code, error)
		}{
			{"Reference", Evaluate}, {"Trusted", EvaluateTrusted},
		} {
			for _, work := range workloads {
				b.Run(fmt.Sprintf("K=%d/%s/%s", k, path.name, work.name), func(b *testing.B) {
					_, code, err := path.evaluate(state, work.raw)
					if err != nil || code != work.code {
						b.Fatalf("fixture code=%d want=%d err=%v", code, work.code, err)
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, code, err := path.evaluate(state, work.raw)
						if err != nil || code != work.code {
							b.Fatal(code, err)
						}
					}
					b.StopTimer()
					b.ReportMetric(float64(logicalBytes), "logical-bytes/op")
				})
			}
		}
		b.Run(fmt.Sprintf("K=%d/EmptyBlock", k), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := state.AdvanceBlock(); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(logicalBytes), "logical-bytes/op")
		})
		b.Run(fmt.Sprintf("K=%d/SerializeAndHash", k), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := state.LogicalBytes(); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(logicalBytes), "logical-bytes/op")
		})
	}
}
