package researchv2app_test

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/experiment/profiles"
	"example.com/pluto/internal/experiment/profiles/app"
	"example.com/pluto/internal/experiment/profiles/testfixture"
)

func TestOneABCIPathForFourProfiles(t *testing.T) {
	f, err := researchv2test.New("rv2-abci-test")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	var reference [32]byte
	for _, p := range []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH} {
		app, err := researchv2app.New(f.Genesis, f.ChainID, p)
		if err != nil {
			t.Fatal(err)
		}
		init, err := app.InitChain(ctx, &abci.RequestInitChain{ChainId: f.ChainID, InitialHeight: 1, AppStateBytes: f.Genesis})
		initialHash := app.Committed().AppHash()
		if err != nil || !bytes.Equal(init.AppHash, initialHash[:]) {
			t.Fatalf("profile %d init: %v", p, err)
		}
		raw, err := f.Signed(p, 7, 0)
		if err != nil {
			t.Fatal(err)
		}
		check, err := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw})
		if err != nil || check.Code != 0 {
			t.Fatalf("profile %d CheckTx: %+v %v", p, check, err)
		}
		prepared, err := app.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 100000, Txs: [][]byte{raw}})
		if err != nil || len(prepared.Txs) != 1 {
			t.Fatalf("profile %d prepare: %+v %v", p, prepared, err)
		}
		process, err := app.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: prepared.Txs})
		if err != nil || process.Status != abci.ResponseProcessProposal_ACCEPT {
			t.Fatalf("profile %d process: %+v %v", p, process, err)
		}
		initial := app.Committed().AppHash()
		final, err := app.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: prepared.Txs})
		if err != nil || len(final.TxResults) != 1 || final.TxResults[0].Code != 0 {
			t.Fatalf("profile %d finalize: %+v %v", p, final, err)
		}
		if app.Committed().AppHash() != initial {
			t.Fatalf("profile %d FinalizeBlock wrote committed state", p)
		}
		if _, err := app.Commit(ctx, &abci.RequestCommit{}); err != nil {
			t.Fatalf("profile %d commit: %v", p, err)
		}
		state := app.Committed()
		committedHash := state.AppHash()
		if state.Height() != 1 || !bytes.Equal(final.AppHash, committedHash[:]) {
			t.Fatalf("profile %d AppHash/height mismatch", p)
		}
		if reference == ([32]byte{}) {
			reference = state.AppHash()
		} else if state.AppHash() != reference {
			t.Fatalf("profile %d state commitment differs", p)
		}
		query, err := app.Query(ctx, &abci.RequestQuery{Path: "rv2/account/v1", Data: f.Sender.Binding.Account[:]})
		if err != nil || query.Code != 0 || len(query.Value) != 17 || query.Value[0] != 1 {
			t.Fatalf("profile %d account query: %+v %v", p, query, err)
		}
		duplicate, err := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw})
		if err != nil || duplicate.Code != uint32(researchv2.CodeInvalidNonce) {
			t.Fatalf("profile %d duplicate check: %+v %v", p, duplicate, err)
		}
	}
}

func TestProposalRejectsWrongProfileWithoutMutation(t *testing.T) {
	f, err := researchv2test.New("rv2-abci-negative")
	if err != nil {
		t.Fatal(err)
	}
	app, err := researchv2app.New(f.Genesis, f.ChainID, researchv2.ProfileH)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	wrong, err := f.Signed(researchv2.ProfileC, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := f.Signed(researchv2.ProfileH, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	check, err := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: wrong})
	if err != nil || check.Code != uint32(researchv2.CodeWrongProfile) {
		t.Fatalf("wrong profile CheckTx: %+v %v", check, err)
	}
	initial := app.Committed().AppHash()
	process, err := app.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: [][]byte{wrong}})
	if err != nil || process.Status != abci.ResponseProcessProposal_REJECT {
		t.Fatalf("wrong profile proposal: %+v %v", process, err)
	}
	prepared, err := app.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 100000, Txs: [][]byte{wrong, valid}})
	if err != nil || len(prepared.Txs) != 1 || !bytes.Equal(prepared.Txs[0], valid) {
		t.Fatalf("wrong profile prepared: %+v %v", prepared, err)
	}
	if app.Committed().AppHash() != initial {
		t.Fatal("rejected proposal changed state")
	}
}

func TestConcurrentAdmissionAcrossCommit(t *testing.T) {
	f, err := researchv2test.New("rv2-abci-concurrent")
	if err != nil {
		t.Fatal(err)
	}
	app, err := researchv2app.New(f.Genesis, f.ChainID, researchv2.ProfileH)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.Signed(researchv2.ProfileH, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	var wg sync.WaitGroup
	errors := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				resp, err := app.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw})
				if err != nil {
					errors <- err
					return
				}
				if resp.Code != 0 && resp.Code != uint32(researchv2.CodeInvalidNonce) {
					errors <- fmt.Errorf("unexpected CheckTx code %d", resp.Code)
					return
				}
				if _, err := app.Info(ctx, &abci.RequestInfo{}); err != nil {
					errors <- err
					return
				}
			}
		}()
	}
	if _, err := app.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{raw}}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Commit(ctx, &abci.RequestCommit{}); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
}
