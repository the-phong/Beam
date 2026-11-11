package b0app

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sort"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/classical/state"
)

func BenchmarkB0CheckTxCost(b *testing.B) {
	sender, receiver := testAddresses(b)
	valid := signedB0(b, "pluto-b0-v1", receiver, 25, 0)
	self := signedB0(b, "pluto-b0-v1", sender, 25, 0)
	wrong := signedB0(b, "pluto-other-1", receiver, 25, 0)
	zero := signedB0(b, "pluto-b0-v1", receiver, 0, 0)
	badScalar := bytes.Clone(valid)
	clear(badScalar[len(badScalar)-64 : len(badScalar)-32])
	badECDSA := bytes.Clone(valid)
	badECDSA[len(badECDSA)-80] ^= 1
	workloads := []struct {
		name string
		raw  []byte
		code uint32
	}{
		{"Malformed", []byte{1}, 1}, {"WrongChain", wrong, 2}, {"ZeroAmount", zero, 5},
		{"NoncanonicalScalar", badScalar, 4}, {"InvalidECDSA", badECDSA, 4},
		{"SelfTransfer", self, 0}, {"DistinctTransfer", valid, 0},
	}
	for _, k := range []int{1, 64, 4096, 65536} {
		accounts := []b0state.Account{{Address: sender, Balance: 1_000_000}}
		if k > 1 {
			accounts = append(accounts, b0state.Account{Address: receiver, Balance: 1_000_000})
		}
		for i := 0; len(accounts) < k; i++ {
			var addr b0state.Address
			binary.BigEndian.PutUint32(addr[16:], uint32(i))
			if addr == sender || addr == receiver {
				continue
			}
			accounts = append(accounts, b0state.Account{Address: addr, Balance: 1_000_000})
		}
		sort.Slice(accounts, func(i, j int) bool { return bytes.Compare(accounts[i].Address[:], accounts[j].Address[:]) < 0 })
		state, err := b0state.NewState("pluto-b0-v1", 1, uint64(k)*1_000_000, accounts)
		if err != nil {
			b.Fatal(err)
		}
		a := syntheticApp(b, uint64(k)*1_000_000, state)
		for _, work := range workloads {
			b.Run(fmt.Sprintf("K=%d/%s", k, work.name), func(b *testing.B) {
				req := &abci.RequestCheckTx{Tx: work.raw}
				response, err := a.CheckTx(context.Background(), req)
				if err != nil || response.Code != work.code {
					b.Fatalf("fixture: %+v %v", response, err)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					response, err := a.CheckTx(context.Background(), req)
					if err != nil || response.Code != work.code {
						b.Fatal(response, err)
					}
				}
				b.StopTimer()
				b.ReportMetric(float64(26+len("pluto-b0-v1")+36*k), "state-logical-bytes/op")
			})
		}
		if err := a.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
