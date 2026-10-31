package bench

import (
	"context"
	"path/filepath"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	b0app "example.com/pluto/internal/classical/app"
	b0state "example.com/pluto/internal/classical/state"
	b0store "example.com/pluto/internal/classical/store"
	"example.com/pluto/internal/classical/tx"
	"example.com/pluto/internal/experiment/corpus"
	b1app "example.com/pluto/internal/hybrid/app"
	b1state "example.com/pluto/internal/hybrid/state"
	b1store "example.com/pluto/internal/hybrid/store"
	b1tx "example.com/pluto/internal/hybrid/tx"
)

var sinkCode uint32
var sinkDigest [32]byte

func fixture(b *testing.B, name string) (corpus.Profile, []byte, []byte, []byte, []byte) {
	b.Helper()
	c, _, err := corpus.Load("../../../research/corpus/phase_05/public_corpus.json")
	if err != nil {
		b.Fatal(err)
	}
	p, ok := c.Profile(name)
	if !ok {
		b.Fatal("missing profile")
	}
	valid, _ := p.Valid[0].Bytes()
	early, _ := p.Cases[0].Bytes()
	lateID := "A03"
	if name == "b1" {
		lateID = "P04"
	}
	lateCase, _ := p.Case(lateID)
	late, _ := lateCase.Bytes()
	genesis, _ := p.Genesis()
	return p, valid, early, late, genesis
}

func BenchmarkPhase05Crypto(b *testing.B) {
	p0, v0, _, _, g0 := fixture(b, "b0")
	p1, v1, _, _, g1 := fixture(b, "b1")
	s0, err := b0state.ParseGenesis(g0, p0.ChainID)
	if err != nil {
		b.Fatal(err)
	}
	s1, err := b1state.ParseGenesis(g1, p1.ChainID)
	if err != nil {
		b.Fatal(err)
	}
	t1, err := b1tx.Parse(v1)
	if err != nil {
		b.Fatal(err)
	}
	d1, err := b1tx.SigningDigest(t1)
	if err != nil {
		b.Fatal(err)
	}
	for _, spec := range []struct {
		name string
		fn   func() uint32
	}{
		{"b0_secp_validate", func() uint32 { return uint32(tx.Validate(v0, p0.ChainID)) }},
		{"b1_secp_verify", func() uint32 { return uint32(b1tx.VerifySecp(t1, d1)) }},
		{"b1_ml_verify", func() uint32 { return uint32(b1tx.VerifyML(t1, d1)) }},
		{"b1_parse_only", func() uint32 {
			_, e := b1tx.Parse(v1)
			if e != nil {
				return 1
			}
			return 0
		}},
		{"b1_chain_profile", func() uint32 {
			if t1.ChainID != p1.ChainID {
				return 1
			}
			return 0
		}},
		{"b1_decode_digest", func() uint32 {
			t, e := b1tx.Parse(v1)
			if e != nil {
				return 1
			}
			sinkDigest, e = b1tx.SigningDigest(t)
			if e != nil {
				return 1
			}
			return 0
		}},
		{"b1_key_binding", func() uint32 {
			a, e := b1tx.SenderAddress(t1.SecpKey, t1.MLKey)
			if e != nil {
				return 1
			}
			if a != t1.Sender {
				return 2
			}
			return 0
		}},
		{"b1_state_rules_after_auth", func() uint32 {
			_, code, e := b0state.ApplyAuthenticatedTrusted(s1.Base(), b0state.Address(t1.Sender), b0state.Address(t1.Receiver), t1.Amount, t1.Nonce)
			if e != nil {
				return 255
			}
			return uint32(code)
		}},
		{"b0_full_validation", func() uint32 {
			_, code, e := b0state.EvaluateTrusted(s0, v0)
			if e != nil {
				return 255
			}
			return uint32(code)
		}},
		{"b1_full_validation", func() uint32 {
			_, code, e := b1state.EvaluateTrusted(s1, v1)
			if e != nil {
				return 255
			}
			return uint32(code)
		}},
	} {
		b.Run(spec.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sinkCode = spec.fn()
			}
			if sinkCode != 0 {
				b.Fatal(sinkCode)
			}
		})
	}
}

func BenchmarkPhase05Application(b *testing.B) {
	ctx := context.Background()
	for _, name := range []string{"b0", "b1"} {
		p, valid, early, late, genesis := fixture(b, name)
		root := filepath.Join(b.TempDir(), name)
		if name == "b0" {
			store, err := b0store.Create(root, genesis, p.ChainID)
			if err != nil {
				b.Fatal(err)
			}
			if err := store.Close(); err != nil {
				b.Fatal(err)
			}
			app, err := b0app.Open(root, p.ChainID, genesis)
			if err != nil {
				b.Fatal(err)
			}
			defer app.Close()
			b.Run("b0_CheckTx_valid", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, e := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: valid})
					if e != nil {
						b.Fatal(e)
					}
					sinkCode = r.Code
				}
			})
			b.Run("b0_CheckTx_early", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, e := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: early})
					if e != nil {
						b.Fatal(e)
					}
					sinkCode = r.Code
				}
			})
			b.Run("b0_CheckTx_late", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, e := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: late})
					if e != nil {
						b.Fatal(e)
					}
					sinkCode = r.Code
				}
			})
			b.Run("b0_ProcessProposal", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, e := app.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: [][]byte{valid}})
					if e != nil {
						b.Fatal(e)
					}
					sinkCode = uint32(r.Status)
				}
			})
			b.Run("b0_PrepareProposal", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, e := app.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 1 << 20, Txs: [][]byte{valid}})
					if e != nil {
						b.Fatal(e)
					}
					sinkCode = uint32(len(r.Txs))
				}
			})
		} else {
			store, err := b1store.Create(root, genesis, p.ChainID)
			if err != nil {
				b.Fatal(err)
			}
			if err := store.Close(); err != nil {
				b.Fatal(err)
			}
			app, err := b1app.Open(root, p.ChainID, genesis)
			if err != nil {
				b.Fatal(err)
			}
			defer app.Close()
			b.Run("b1_CheckTx_valid", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, e := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: valid})
					if e != nil {
						b.Fatal(e)
					}
					sinkCode = r.Code
				}
			})
			b.Run("b1_CheckTx_early", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, e := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: early})
					if e != nil {
						b.Fatal(e)
					}
					sinkCode = r.Code
				}
			})
			b.Run("b1_CheckTx_late_ml", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, e := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: late})
					if e != nil {
						b.Fatal(e)
					}
					sinkCode = r.Code
				}
			})
			b.Run("b1_ProcessProposal", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, e := app.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: [][]byte{valid}})
					if e != nil {
						b.Fatal(e)
					}
					sinkCode = uint32(r.Status)
				}
			})
			b.Run("b1_PrepareProposal", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, e := app.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 1 << 20, Txs: [][]byte{valid}})
					if e != nil {
						b.Fatal(e)
					}
					sinkCode = uint32(len(r.Txs))
				}
			})
		}
	}
}

func BenchmarkPhase05FinalizeCommit(b *testing.B) {
	ctx := context.Background()
	for _, profile := range []string{"b0", "b1"} {
		p, _, _, _, genesis := fixture(b, profile)
		for _, stage := range []string{"FinalizeBlock", "Commit"} {
			b.Run(profile+"_"+stage, func(b *testing.B) {
				root := filepath.Join(b.TempDir(), "state")
				var finalize func(int64) error
				var commit func() error
				if profile == "b0" {
					store, err := b0store.Create(root, genesis, p.ChainID)
					if err != nil {
						b.Fatal(err)
					}
					if err := store.Close(); err != nil {
						b.Fatal(err)
					}
					app, err := b0app.Open(root, p.ChainID, genesis)
					if err != nil {
						b.Fatal(err)
					}
					defer app.Close()
					finalize = func(height int64) error {
						_, e := app.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: height})
						return e
					}
					commit = func() error { _, e := app.Commit(ctx, &abci.RequestCommit{}); return e }
				} else {
					store, err := b1store.Create(root, genesis, p.ChainID)
					if err != nil {
						b.Fatal(err)
					}
					if err := store.Close(); err != nil {
						b.Fatal(err)
					}
					app, err := b1app.Open(root, p.ChainID, genesis)
					if err != nil {
						b.Fatal(err)
					}
					defer app.Close()
					finalize = func(height int64) error {
						_, e := app.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: height})
						return e
					}
					commit = func() error { _, e := app.Commit(ctx, &abci.RequestCommit{}); return e }
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if stage == "FinalizeBlock" {
						if err := finalize(int64(i + 1)); err != nil {
							b.Fatal(err)
						}
						b.StopTimer()
						if err := commit(); err != nil {
							b.Fatal(err)
						}
						b.StartTimer()
					} else {
						b.StopTimer()
						if err := finalize(int64(i + 1)); err != nil {
							b.Fatal(err)
						}
						b.StartTimer()
						if err := commit(); err != nil {
							b.Fatal(err)
						}
					}
				}
			})
		}
	}
}

func BenchmarkPhase05RecorderOverhead(b *testing.B) {
	p, valid, _, _, genesis := fixture(b, "b1")
	state, err := b1state.ParseGenesis(genesis, p.ChainID)
	if err != nil {
		b.Fatal(err)
	}
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		b.Run(name, func(b *testing.B) {
			recorder := &Recorder{Enabled: enabled, Capacity: 1}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := recorder.Do(S6, func() error {
					_, code, e := b1state.EvaluateTrusted(state, valid)
					sinkCode = uint32(code)
					return e
				}); err != nil {
					b.Fatal(err)
				}
			}
			if sinkCode != 0 {
				b.Fatal(sinkCode)
			}
		})
	}
}

func BenchmarkPhase05InvalidClasses(b *testing.B) {
	for _, profile := range []string{"b0", "b1"} {
		p, _, _, _, genesis := fixture(b, profile)
		ids := []string{"S01", "P03", "A03"}
		if profile == "b1" {
			ids = []string{"S01", "I01", "P03", "P04", "A03"}
		}
		for _, id := range ids {
			cs, ok := p.Case(id)
			if !ok {
				b.Fatal(id)
			}
			raw, err := cs.Bytes()
			if err != nil {
				b.Fatal(err)
			}
			if profile == "b0" {
				state, err := b0state.ParseGenesis(genesis, p.ChainID)
				if err != nil {
					b.Fatal(err)
				}
				b.Run(profile+"_"+id, func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						_, code, e := b0state.EvaluateTrusted(state, raw)
						if e != nil {
							b.Fatal(e)
						}
						sinkCode = uint32(code)
					}
					if sinkCode != cs.ExpectedCode {
						b.Fatal(sinkCode)
					}
				})
			} else {
				state, err := b1state.ParseGenesis(genesis, p.ChainID)
				if err != nil {
					b.Fatal(err)
				}
				b.Run(profile+"_"+id, func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						_, code, e := b1state.EvaluateTrusted(state, raw)
						if e != nil {
							b.Fatal(e)
						}
						sinkCode = uint32(code)
					}
					if sinkCode != cs.ExpectedCode {
						b.Fatal(sinkCode)
					}
				})
			}
		}
	}
}
