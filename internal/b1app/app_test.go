package b1app_test

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/b1app"
	"example.com/pluto/internal/b1genesis"
	"example.com/pluto/internal/b1store"
	"example.com/pluto/internal/b1test"
	"example.com/pluto/internal/b1tx"
)

func TestHybridABCIProposalAndCommit(t *testing.T) {
	fixture, err := b1test.New("pluto-b1-test")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "b1")
	store, err := b1store.Create(dir, fixture.Genesis, fixture.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	a, err := b1app.Open(dir, fixture.ChainID, fixture.Genesis)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if a == nil {
			return
		}
		if err := a.Close(); err != nil && !errors.Is(err, b1store.ErrClosed) {
			t.Errorf("close reopened B1 application: %v", err)
		}
	})
	ctx := context.Background()
	transport, err := b1genesis.Encode(fixture.Genesis, fixture.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	init := &abci.RequestInitChain{ChainId: fixture.ChainID, InitialHeight: 1, AppStateBytes: transport}
	first, err := a.InitChain(ctx, init)
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.InitChain(ctx, init)
	if err != nil || !bytes.Equal(first.AppHash, second.AppHash) {
		t.Fatalf("H0 retry: %v", err)
	}
	h0, err := a.Committed()
	h0Hash := h0.AppHash()
	if err != nil || h0.Height() != 0 || !bytes.Equal(first.AppHash, h0Hash[:]) {
		t.Fatalf("InitChain rewrote or advanced H0: %v", err)
	}
	badInit := *init
	badInit.ChainId = "other-chain"
	if _, err := a.InitChain(ctx, &badInit); err == nil {
		t.Fatal("conflicting InitChain accepted")
	}
	other, err := b1test.New(fixture.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	otherTransport, err := b1genesis.Encode(other.Genesis, fixture.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	badInit = *init
	badInit.AppStateBytes = otherTransport
	if _, err := a.InitChain(ctx, &badInit); err == nil {
		t.Fatal("different Genesis V3 accepted at H0")
	}
	badInit.AppStateBytes = []byte(`{"genesis":"noncanonical"}`)
	if _, err := a.InitChain(ctx, &badInit); err == nil {
		t.Fatal("noncanonical B1 transport accepted at H0")
	}
	valid, err := fixture.Signed(25, 0)
	if err != nil {
		t.Fatal(err)
	}
	bad := bytes.Clone(valid)
	bad[len(bad)-1] ^= 1
	t.Logf("H19/H20 direct proposal raw_valid=%x raw_bad=%x", valid, bad)
	for _, kind := range []abci.CheckTxType{abci.CheckTxType_New, abci.CheckTxType_Recheck} {
		resp, err := a.CheckTx(ctx, &abci.RequestCheckTx{Tx: valid, Type: kind})
		if err != nil || resp.Code != 0 {
			t.Fatalf("H01 CheckTx type=%v code=%d err=%v", kind, resp.Code, err)
		}
		resp, err = a.CheckTx(ctx, &abci.RequestCheckTx{Tx: bad, Type: kind})
		if err != nil || resp.Code != uint32(b1tx.CodeBadMLSignature) {
			t.Fatalf("H05 CheckTx type=%v code=%d err=%v", kind, resp.Code, err)
		}
	}
	prepared, err := a.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 1048576, Txs: [][]byte{bad, valid}})
	if err != nil || len(prepared.Txs) != 1 || !bytes.Equal(prepared.Txs[0], valid) {
		t.Fatalf("PrepareProposal: %v %d", err, len(prepared.Txs))
	}
	for _, proposal := range [][][]byte{{bad}, {valid, bad}, {valid, valid}} {
		resp, err := a.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: proposal})
		if err != nil || resp.Status != abci.ResponseProcessProposal_REJECT {
			t.Fatalf("H19/H20 invalid proposal accepted: %v %+v", err, resp)
		}
		t.Logf("H19/H20 proposal tx lengths=%v rejected without commit", func() []int {
			n := []int{}
			for _, raw := range proposal {
				n = append(n, len(raw))
			}
			return n
		}())
	}
	accepted, err := a.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: [][]byte{valid}})
	if err != nil || accepted.Status != abci.ResponseProcessProposal_ACCEPT {
		t.Fatalf("valid proposal: %v", err)
	}
	before, err := a.Committed()
	if err != nil {
		t.Fatal(err)
	}
	final, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{valid, bad}})
	if err != nil || len(final.TxResults) != 2 || final.TxResults[0].Code != 0 || final.TxResults[1].Code != uint32(b1tx.CodeBadMLSignature) {
		t.Fatalf("fallback finalization: %v %+v", err, final)
	}
	still, err := a.Committed()
	if err != nil || still.AppHash() != before.AppHash() {
		t.Fatal("FinalizeBlock changed committed state")
	}
	if _, err := a.Commit(ctx, &abci.RequestCommit{}); err != nil {
		t.Fatal(err)
	}
	committed, err := a.Committed()
	if err != nil || committed.Height() != 1 {
		t.Fatalf("commit height: %v", err)
	}
	sender, _ := committed.Account(fixture.Sender.Binding.Address)
	receiver, _ := committed.Account(fixture.Receiver.Binding.Address)
	if sender.Balance != 75 || sender.Nonce != 1 || receiver.Balance != 25 || receiver.Nonce != 0 || committed.TotalSupply() != 100 {
		t.Fatalf("balances: %+v %+v", sender, receiver)
	}
	query, err := a.Query(ctx, &abci.RequestQuery{Path: "b1/account/v1", Data: fixture.Sender.Binding.Address[:]})
	if err != nil || query.Code != 0 || len(query.Value) != 17 {
		t.Fatalf("query: %v %+v", err, query)
	}
	info, err := a.Info(ctx, &abci.RequestInfo{})
	if err != nil || info.LastBlockHeight != 1 || !bytes.Equal(info.LastBlockAppHash, final.AppHash) {
		t.Fatalf("Info: %v %+v", err, info)
	}
	if _, err := a.InitChain(ctx, init); err == nil {
		t.Fatal("advanced application accepted InitChain")
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	a, err = b1app.Open(dir, fixture.ChainID, fixture.Genesis)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := a.Committed()
	if err != nil || replayed.AppHash() != committed.AppHash() {
		t.Fatalf("H21 replay: %v", err)
	}
	_, code, err := b1storeReplay(a, ctx, valid)
	if err != nil || code != b1tx.CodeInvalidNonce {
		t.Fatalf("H17 replay code=%d err=%v", code, err)
	}
	t.Logf("H01/H19/H20/H21 committed_height=%d AppHash=%x sender=%d/%d receiver=%d/%d raw=%x", replayed.Height(), replayed.AppHash(), sender.Balance, sender.Nonce, receiver.Balance, receiver.Nonce, valid)
}

func b1storeReplay(a *b1app.App, ctx context.Context, raw []byte) (*abci.ResponseCheckTx, b1tx.Code, error) {
	resp, err := a.CheckTx(ctx, &abci.RequestCheckTx{Tx: raw, Type: abci.CheckTxType_New})
	if err != nil {
		return nil, 0, err
	}
	return resp, b1tx.Code(resp.Code), nil
}

func TestHybridPostSyncResponseLoss(t *testing.T) {
	fixture, err := b1test.New("pluto-b1-test")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "b1")
	store, err := b1store.Create(dir, fixture.Genesis, fixture.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	a, err := b1app.OpenWithStoreOptions(dir, fixture.ChainID, fixture.Genesis, b1store.Options{AfterSync: func() error { return errors.New("injected response loss after Pebble Sync") }})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := fixture.Signed(25, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.FinalizeBlock(context.Background(), &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{raw}}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Commit(context.Background(), &abci.RequestCommit{}); err == nil {
		t.Fatal("post-sync response loss not reached")
	}
	if _, err := a.Committed(); !errors.Is(err, b1store.ErrUncertain) {
		t.Fatalf("uncertain read: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	a, err = b1app.Open(dir, fixture.ChainID, fixture.Genesis)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	state, err := a.Committed()
	if err != nil || state.Height() != 1 {
		t.Fatalf("reopen height: %v", err)
	}
	sender, _ := state.Account(fixture.Sender.Binding.Address)
	receiver, _ := state.Account(fixture.Receiver.Binding.Address)
	if sender.Balance != 75 || sender.Nonce != 1 || receiver.Balance != 25 {
		t.Fatalf("duplicate or missing effect: %+v %+v", sender, receiver)
	}
	t.Logf("H23 simulated post-sync response loss A=%d AppHash=%x sender=%d/%d receiver=%d/%d exact_once=true", state.Height(), state.AppHash(), sender.Balance, sender.Nonce, receiver.Balance, receiver.Nonce)
}
