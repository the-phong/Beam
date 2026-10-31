package bench

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/experiment/corpus"
	b1app "example.com/pluto/internal/hybrid/app"
	b1store "example.com/pluto/internal/hybrid/store"
)

func TestMeasurementDoesNotChangeB1Result(t *testing.T) {
	c, _, err := corpus.Load("../../../research/corpus/phase_05/public_corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	p, _ := c.Profile("b1")
	genesis, _ := p.Genesis()
	valid, _ := p.Valid[0].Bytes()
	bad, _ := p.Case("P04")
	invalid, _ := bad.Bytes()
	ctx := context.Background()
	type outcome struct {
		codes     []uint32
		proposal  abci.ResponseProcessProposal_ProposalStatus
		hash      []byte
		committed [32]byte
	}
	run := func(enabled bool) outcome {
		dir := filepath.Join(t.TempDir(), "store")
		store, e := b1store.Create(dir, genesis, p.ChainID)
		if e != nil {
			t.Fatal(e)
		}
		if e = store.Close(); e != nil {
			t.Fatal(e)
		}
		app, e := b1app.Open(dir, p.ChainID, genesis)
		if e != nil {
			t.Fatal(e)
		}
		defer app.Close()
		recorder := &Recorder{Enabled: enabled, Capacity: 16}
		var out outcome
		for _, raw := range [][]byte{valid, invalid} {
			var reply *abci.ResponseCheckTx
			e = recorder.Do(S7, func() error { var err error; reply, err = app.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw}); return err })
			if e != nil {
				t.Fatal(e)
			}
			out.codes = append(out.codes, reply.Code)
		}
		var proposal *abci.ResponseProcessProposal
		e = recorder.Do(S8, func() error {
			var err error
			proposal, err = app.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: [][]byte{valid}})
			return err
		})
		if e != nil {
			t.Fatal(e)
		}
		out.proposal = proposal.Status
		var finalized *abci.ResponseFinalizeBlock
		e = recorder.Do(S9, func() error {
			var err error
			finalized, err = app.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{valid}})
			return err
		})
		if e != nil {
			t.Fatal(e)
		}
		out.hash = bytes.Clone(finalized.AppHash)
		e = recorder.Do(S10, func() error { _, err := app.Commit(ctx, &abci.RequestCommit{}); return err })
		if e != nil {
			t.Fatal(e)
		}
		state, e := app.Committed()
		if e != nil {
			t.Fatal(e)
		}
		out.committed = state.AppHash()
		if enabled && (len(recorder.Samples) != 5 || recorder.Dropped != 0) {
			t.Fatal("missing or dropped callback samples")
		}
		if !enabled && (len(recorder.Samples) != 0 || recorder.Dropped != 0) {
			t.Fatal("disabled recorder emitted samples")
		}
		return out
	}
	a, b := run(false), run(true)
	if !bytes.Equal(a.hash, b.hash) || a.committed != b.committed || a.proposal != b.proposal || len(a.codes) != len(b.codes) {
		t.Fatal("measurement changed B1 result")
	}
	for i := range a.codes {
		if a.codes[i] != b.codes[i] {
			t.Fatal("measurement changed transaction code")
		}
	}
}

func TestRecorderDropsOnlySamples(t *testing.T) {
	r := &Recorder{Enabled: true, Capacity: 1}
	called := 0
	for i := 0; i < 3; i++ {
		if err := r.Do(S7, func() error { called++; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if called != 3 || len(r.Samples) != 1 || r.Dropped != 2 {
		t.Fatal("recorder changed operation execution")
	}
}
