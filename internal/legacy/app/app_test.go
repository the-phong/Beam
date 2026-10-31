package app

import (
	"context"
	"path/filepath"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
)

func newTestApp(t *testing.T) (*App, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "app")
	a, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a, dir
}

func initApp(t *testing.T, a *App) []byte {
	t.Helper()
	r, err := a.InitChain(context.Background(), &abci.RequestInitChain{ChainId: "pluto-test", InitialHeight: 1})
	if err != nil {
		t.Fatal(err)
	}
	return r.AppHash
}

func TestLifecycleAndDeterministicReplay(t *testing.T) {
	ctx := context.Background()
	a, _ := newTestApp(t)
	b, _ := newTestApp(t)
	for _, x := range []*App{a, b} {
		info, err := x.Info(ctx, &abci.RequestInfo{})
		if err != nil || info.LastBlockHeight != 0 || len(info.LastBlockAppHash) != 0 {
			t.Fatalf("before init: %v %v", info, err)
		}
	}
	genA, genB := initApp(t, a), initApp(t, b)
	if string(genA) != string(genB) {
		t.Fatal("genesis hashes differ")
	}
	if _, err := a.InitChain(ctx, &abci.RequestInitChain{ChainId: "wrong", InitialHeight: 1}); err == nil {
		t.Fatal("accepted conflicting chain")
	}
	for h := int64(1); h <= 3; h++ {
		var hashes [][]byte
		for _, x := range []*App{a, b} {
			before := x.Committed()
			check, err := x.CheckTx(ctx, &abci.RequestCheckTx{Tx: []byte("tx")})
			if err != nil || check.Code != CodeUnsupportedTx {
				t.Fatalf("CheckTx: %v %v", check, err)
			}
			prep, err := x.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: h, Txs: [][]byte{[]byte("bad")}})
			if err != nil || len(prep.Txs) != 0 {
				t.Fatalf("PrepareProposal: %v %v", prep, err)
			}
			reject, err := x.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: h, Txs: [][]byte{[]byte("bad")}})
			if err != nil || reject.Status != abci.ResponseProcessProposal_REJECT {
				t.Fatalf("bad proposal: %v %v", reject, err)
			}
			accept, err := x.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: h})
			if err != nil || accept.Status != abci.ResponseProcessProposal_ACCEPT {
				t.Fatalf("good proposal: %v %v", accept, err)
			}
			if *x.Committed() != *before {
				t.Fatal("admission/proposal mutated committed state")
			}
			if _, err := x.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: h, Txs: [][]byte{[]byte("bad")}}); err == nil {
				t.Fatal("finalized unsupported transaction")
			}
			final, err := x.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: h})
			if err != nil {
				t.Fatal(err)
			}
			hashes = append(hashes, final.AppHash)
			info, _ := x.Info(ctx, &abci.RequestInfo{})
			if info.LastBlockHeight != h-1 {
				t.Fatal("FinalizeBlock exposed uncommitted state")
			}
			if _, err := x.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: h}); err == nil {
				t.Fatal("accepted duplicate FinalizeBlock")
			}
			if _, err := x.Commit(ctx, &abci.RequestCommit{}); err != nil {
				t.Fatal(err)
			}
			info, _ = x.Info(ctx, &abci.RequestInfo{})
			if info.LastBlockHeight != h || string(info.LastBlockAppHash) != string(final.AppHash) {
				t.Fatal("Info not committed state")
			}
		}
		if string(hashes[0]) != string(hashes[1]) {
			t.Fatalf("replay diverged at %d", h)
		}
	}
	if _, err := a.Commit(ctx, &abci.RequestCommit{}); err == nil {
		t.Fatal("duplicate commit succeeded")
	}
	if r, _ := a.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 3}); r.Status != abci.ResponseProcessProposal_REJECT {
		t.Fatal("accepted stale height")
	}
	if _, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 5}); err == nil {
		t.Fatal("accepted skipped height")
	}
}

func TestFinalizeWithoutCommitAndFailure(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "app")
	a, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	initApp(t, a)
	genesis := *a.Committed()
	if _, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1}); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	a, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if *a.Committed() != genesis {
		t.Fatal("uncommitted FinalizeBlock persisted")
	}
	if _, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1}); err != nil {
		t.Fatal(err)
	}
	// A corrupted pending record makes persistence reject the write before it
	// can be applied. Commit must not promote pending state.
	a.pending.AppHash[0] ^= 1
	if _, err := a.Commit(ctx, &abci.RequestCommit{}); err == nil {
		t.Fatal("failed persistence reported success")
	}
	if *a.Committed() != genesis {
		t.Fatal("failed persistence promoted state")
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	a, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if *a.Committed() != genesis {
		t.Fatal("failed commit changed durable state")
	}
}
