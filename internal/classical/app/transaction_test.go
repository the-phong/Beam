package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	b0genesis "example.com/pluto/internal/classical/genesis"
	b0state "example.com/pluto/internal/classical/state"
	b0store "example.com/pluto/internal/classical/store"
	"example.com/pluto/internal/classical/tx"
)

// Test-only scalar 1 is public and never used as a node or production key.
func signedB0(t testing.TB, chain string, receiver [20]byte, amount, nonce uint64) []byte {
	return signedB0Key(t, 1, chain, receiver, amount, nonce)
}

func signedB0Key(t testing.TB, keyByte byte, chain string, receiver [20]byte, amount, nonce uint64) []byte {
	t.Helper()
	var secret [32]byte
	secret[31] = keyByte
	key := secp256k1.PrivKeyFromBytes(secret[:])
	record := tx.Transaction{ChainID: chain, Receiver: receiver, Amount: amount, Nonce: nonce}
	copy(record.Sender[:], key.PubKey().SerializeCompressed())
	digest, err := tx.SigningDigest(record)
	if err != nil {
		t.Fatal(err)
	}
	signature := ecdsa.Sign(key, digest[:])
	r, s := signature.R(), signature.S()
	r.PutBytes(&record.R)
	s.PutBytes(&record.S)
	wire, err := tx.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func testAddresses(t testing.TB) (b0state.Address, b0state.Address) {
	t.Helper()
	var secretA, secretB [32]byte
	secretA[31], secretB[31] = 1, 2
	a, b := secp256k1.PrivKeyFromBytes(secretA[:]), secp256k1.PrivKeyFromBytes(secretB[:])
	var ka, kb [33]byte
	copy(ka[:], a.PubKey().SerializeCompressed())
	copy(kb[:], b.PubKey().SerializeCompressed())
	addrA, err := tx.SenderAddress(ka)
	if err != nil {
		t.Fatal(err)
	}
	addrB, err := tx.SenderAddress(kb)
	if err != nil {
		t.Fatal(err)
	}
	return addrA, addrB
}

func appWithBalance(t *testing.T, chain string, balance uint64) (*App, string, b0state.Address, b0state.Address) {
	t.Helper()
	sender, receiver := testAddresses(t)
	genesis, err := b0state.EncodeGenesis(chain, []b0state.Allocation{{Address: sender, Balance: balance}})
	if err != nil {
		t.Fatal(err)
	}
	transport, err := b0genesis.Encode(genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "b0")
	created, err := b0store.Create(path, genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	if err := created.Close(); err != nil {
		t.Fatal(err)
	}
	a, err := Open(path, chain, genesis)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.InitChain(context.Background(), &abci.RequestInitChain{InitialHeight: 1, ChainId: chain, AppStateBytes: transport}); err != nil {
		t.Fatal(err)
	}
	return a, path, sender, receiver
}

func TestB0ABCITransactionLifecycle(t *testing.T) {
	const chain = "pluto-b0-v1"
	a, path, sender, receiver := appWithBalance(t, chain, 100)
	ctx := context.Background()
	first := signedB0(t, chain, receiver, 25, 0)
	second := signedB0(t, chain, receiver, 10, 1)
	bad := bytes.Clone(first)
	bad[0] = 0
	for _, test := range []struct {
		raw  []byte
		code uint32
		log  string
	}{
		{first, 0, "ok"}, {second, 6, "invalid_nonce"}, {bad, 1, "malformed"},
	} {
		resp, err := a.CheckTx(ctx, &abci.RequestCheckTx{Tx: test.raw})
		if err != nil || resp.Code != test.code || resp.Log != test.log || resp.Codespace != resultCodespace || resp.GasWanted != 0 || resp.GasUsed != 0 {
			t.Fatalf("CheckTx: %+v %v", resp, err)
		}
	}
	proposed, err := a.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 1000, Txs: [][]byte{bad, second, first, second}})
	if err != nil || len(proposed.Txs) != 2 || !bytes.Equal(proposed.Txs[0], first) || !bytes.Equal(proposed.Txs[1], second) {
		t.Fatalf("PrepareProposal: %+v %v", proposed, err)
	}
	for _, raws := range [][][]byte{{bad}, {second}, {first, first}} {
		resp, err := a.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: raws})
		if err != nil || resp.Status != abci.ResponseProcessProposal_REJECT {
			t.Fatalf("ProcessProposal invalid: %+v %v", resp, err)
		}
	}
	accepted, err := a.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: proposed.Txs})
	if err != nil || accepted.Status != abci.ResponseProcessProposal_ACCEPT {
		t.Fatalf("ProcessProposal valid: %+v %v", accepted, err)
	}
	final, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{bad, first, first, second}})
	if err != nil || len(final.TxResults) != 4 {
		t.Fatalf("FinalizeBlock: %+v %v", final, err)
	}
	for i, want := range []uint32{1, 0, 6, 0} {
		if final.TxResults[i].Code != want || final.TxResults[i].Codespace != resultCodespace {
			t.Fatalf("FinalizeBlock tx %d: %+v", i, final.TxResults[i])
		}
	}
	before, _, err := a.Committed()
	if err != nil || before.Height() != 0 {
		t.Fatalf("pending became committed: %+v %v", before, err)
	}
	if _, err := a.Commit(ctx, &abci.RequestCommit{}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Commit(ctx, &abci.RequestCommit{}); err == nil {
		t.Fatal("duplicate Commit")
	}
	for _, check := range []struct {
		addr           b0state.Address
		balance, nonce uint64
	}{
		{sender, 65, 2}, {receiver, 35, 0},
	} {
		q, err := a.Query(ctx, &abci.RequestQuery{Path: "b0/account/v1", Data: check.addr[:]})
		if err != nil || q.Code != 0 || q.Height != 1 || len(q.Value) != 17 || q.Value[0] != 1 || binary.BigEndian.Uint64(q.Value[1:9]) != check.balance || binary.BigEndian.Uint64(q.Value[9:17]) != check.nonce {
			t.Fatalf("Query: %+v %v", q, err)
		}
	}
	info, err := a.Info(ctx, &abci.RequestInfo{})
	if err != nil || info.LastBlockHeight != 1 || !bytes.Equal(info.LastBlockAppHash, final.AppHash) {
		t.Fatalf("Info: %+v %v", info, err)
	}
	replay, err := a.CheckTx(ctx, &abci.RequestCheckTx{Tx: first, Type: abci.CheckTxType_Recheck})
	if err != nil || replay.Code != 6 {
		t.Fatalf("replay CheckTx: %+v %v", replay, err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	genesis, err := b0state.EncodeGenesis(chain, []b0state.Allocation{{Address: sender, Balance: 100}})
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(path, chain, genesis)
	if err != nil {
		t.Fatalf("post-transfer restart: %v", err)
	}
	defer restarted.Close()
	state, exists, err := restarted.Committed()
	recoveredHash := state.AppHash()
	if err != nil || !exists || state.Height() != 1 || state.TotalSupply() != 100 || !bytes.Equal(recoveredHash[:], final.AppHash) {
		t.Fatalf("recovered state: %+v %v", state, err)
	}
	third := signedB0(t, chain, receiver, 5, 2)
	response, err := restarted.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 2, Txs: [][]byte{first, third}})
	if err != nil || response.TxResults[0].Code != 6 || response.TxResults[1].Code != 0 {
		t.Fatalf("post-restart execution: %+v %v", response, err)
	}
}

func TestB0ProposalBudgetAndLimitFallback(t *testing.T) {
	a, _, _, receiver := appWithBalance(t, "c", 6145)
	defer a.Close()
	ctx := context.Background()
	first := signedB0(t, "c", receiver, 1, 0)
	second := signedB0(t, "c", receiver, 1, 1)
	if len(first) != 146 {
		t.Fatalf("fixture width %d", len(first))
	}
	resp, err := a.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 146, Txs: [][]byte{bytes.Repeat([]byte{1}, 195), first, second}})
	if err != nil || len(resp.Txs) != 1 || !bytes.Equal(resp.Txs[0], first) {
		t.Fatalf("skip oversized candidate: %+v %v", resp, err)
	}
	empty, err := a.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 0, Txs: [][]byte{first}})
	if err != nil || len(empty.Txs) != 0 {
		t.Fatalf("zero budget: %+v %v", empty, err)
	}
	// Every V20 record is signed, funded, and carries a sequential nonce.
	countBatch := make([][]byte, 6145)
	for i := range countBatch {
		countBatch[i] = signedB0(t, "c", receiver, 1, uint64(i))
	}
	if len(countBatch)*len(first) != 897170 {
		t.Fatal("count fixture byte total")
	}
	countInitial, _, err := a.Committed()
	if err != nil {
		t.Fatal(err)
	}
	_, countCodes, err := b0state.SimulateOrderedTrusted(countInitial, countBatch)
	if err != nil {
		t.Fatal(err)
	}
	for i, code := range countCodes {
		if code != 0 {
			t.Fatalf("count fixture tx %d invalid: %d", i, code)
		}
	}
	countDigest := sha256.New()
	for _, wire := range countBatch {
		_, _ = countDigest.Write(wire)
	}
	t.Logf("V20 count chain=c K=1 genesis_supply=6145 tx_count=%d aggregate_bytes=%d sha256_concatenated=%x first_wire=%s last_wire=%s", len(countBatch), len(countBatch)*146, countDigest.Sum(nil), hex.EncodeToString(countBatch[0]), hex.EncodeToString(countBatch[len(countBatch)-1]))
	decision, err := a.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: countBatch})
	if err != nil || decision.Status != abci.ResponseProcessProposal_REJECT {
		t.Fatalf("count ProcessProposal: %+v %v", decision, err)
	}
	final, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: countBatch})
	if err != nil || len(final.TxResults) != len(countBatch) {
		t.Fatalf("count FinalizeBlock: %v", err)
	}
	for _, result := range final.TxResults {
		if result.Code != 10 || result.Log != "block_limit" {
			t.Fatal("count fallback result")
		}
	}
	if _, err := a.Commit(ctx, &abci.RequestCommit{}); err != nil {
		t.Fatal(err)
	}
	state, _, err := a.Committed()
	if err != nil || state.Height() != 1 || state.TotalSupply() != 6145 || len(state.Accounts()) != 1 {
		t.Fatalf("count fallback state: %+v %v", state, err)
	}
	wantCount, err := countInitial.AdvanceBlock()
	if err != nil || state.AppHash() != wantCount.AppHash() {
		t.Fatalf("count fallback AppHash: %v", err)
	}
	longChain := strings.Repeat("c", 50)
	b, _, _, longReceiver := appWithBalance(t, longChain, 5378)
	defer b.Close()
	byteBatch := make([][]byte, 5378)
	for i := range byteBatch {
		byteBatch[i] = signedB0(t, longChain, longReceiver, 1, uint64(i))
	}
	if len(byteBatch[0]) != 195 || len(byteBatch)*195 != 1048710 {
		t.Fatal("byte fixture size")
	}
	byteInitial, _, err := b.Committed()
	if err != nil {
		t.Fatal(err)
	}
	_, byteCodes, err := b0state.SimulateOrderedTrusted(byteInitial, byteBatch)
	if err != nil {
		t.Fatal(err)
	}
	for i, code := range byteCodes {
		if code != 0 {
			t.Fatalf("byte fixture tx %d invalid: %d", i, code)
		}
	}
	byteDigest := sha256.New()
	for _, wire := range byteBatch {
		_, _ = byteDigest.Write(wire)
	}
	t.Logf("V20 bytes chain_length=50 K=1 genesis_supply=5378 tx_count=%d aggregate_bytes=%d sha256_concatenated=%x first_wire=%s last_wire=%s", len(byteBatch), len(byteBatch)*195, byteDigest.Sum(nil), hex.EncodeToString(byteBatch[0]), hex.EncodeToString(byteBatch[len(byteBatch)-1]))
	decision, err = b.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: byteBatch})
	if err != nil || decision.Status != abci.ResponseProcessProposal_REJECT {
		t.Fatalf("byte ProcessProposal: %+v %v", decision, err)
	}
	final, err = b.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: byteBatch})
	if err != nil || len(final.TxResults) != len(byteBatch) {
		t.Fatalf("byte FinalizeBlock: %v", err)
	}
	for _, result := range final.TxResults {
		if result.Code != 10 {
			t.Fatal("byte fallback result")
		}
	}
	wantBytes, err := byteInitial.AdvanceBlock()
	if err != nil {
		t.Fatal(err)
	}
	pending, exists, err := b.store.Pending()
	if err != nil || !exists || pending.AppHash() != wantBytes.AppHash() {
		t.Fatalf("byte fallback AppHash: %v", err)
	}
}

func TestB0QueryAndNonceBoundary(t *testing.T) {
	a, _, _, receiver := appWithBalance(t, "pluto-b0-v1", 100)
	defer a.Close()
	ctx := context.Background()
	for _, req := range []*abci.RequestQuery{{}, {Path: "b0/account/v1", Data: []byte{1}}, {Path: "b0/account/v1", Data: make([]byte, 20), Height: 1}, {Path: "b0/account/v1", Data: make([]byte, 20), Prove: true}} {
		q, err := a.Query(ctx, req)
		if err != nil || q.Code != 1 || q.Log != "unsupported_query" || q.Codespace != "pluto.query.v1" || len(q.Value) != 0 {
			t.Fatalf("unsupported Query: %+v %v", q, err)
		}
	}
	q, err := a.Query(ctx, &abci.RequestQuery{Path: "b0/account/v1", Data: receiver[:]})
	if err != nil || q.Code != 0 || !bytes.Equal(q.Value, []byte{0}) {
		t.Fatalf("missing account: %+v %v", q, err)
	}
	// Nonce overflow is exercised by the accepted state-engine fixture.
}
