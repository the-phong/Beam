package b0app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"math/big"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/cockroachdb/pebble"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/b0store"
	"example.com/pluto/internal/tx"
)

func TestB0AcceptedSingleTransactionVectors(t *testing.T) {
	const chain = "pluto-b0-v1"
	_, receiver := testAddresses(t)
	valid := signedB0(t, chain, receiver, 25, 0)
	clone := func() []byte { return bytes.Clone(valid) }
	badR := clone()
	clear(badR[len(badR)-64 : len(badR)-32])
	highS := clone()
	low := new(big.Int).SetBytes(highS[len(highS)-32:])
	new(big.Int).Sub(secp256k1.Params().N, low).FillBytes(highS[len(highS)-32:])
	modifiedAmount := clone()
	binary.BigEndian.PutUint64(modifiedAmount[len(modifiedAmount)-80:len(modifiedAmount)-72], 26)
	badKey := clone()
	badKey[12+len(chain)] = 4
	badMagic := clone()
	badMagic[0] = 0
	badVersion := clone()
	badVersion[9] = 2
	short := clone()[:len(valid)-1]
	long := append(clone(), 0)
	badLength0 := clone()
	binary.BigEndian.PutUint16(badLength0[10:12], 0)
	badLength51 := clone()
	binary.BigEndian.PutUint16(badLength51[10:12], 51)
	badChainByte := clone()
	badChainByte[12] = 0xff
	uncompressedKey := clone()
	keyEnd := 12 + len(chain) + 33
	uncompressedKey = append(append([]byte(nil), uncompressedKey[:keyEnd]...), append(bytes.Repeat([]byte{0}, 32), uncompressedKey[keyEnd:]...)...)
	zeroAndBadSignature := clone()
	binary.BigEndian.PutUint64(zeroAndBadSignature[len(zeroAndBadSignature)-80:len(zeroAndBadSignature)-72], 0)
	zeroAndBadSignature[len(zeroAndBadSignature)-33] ^= 1
	badBinding := clone()
	var bSecret [32]byte
	bSecret[31] = 2
	copy(badBinding[12+len(chain):12+len(chain)+33], secp256k1.PrivKeyFromBytes(bSecret[:]).PubKey().SerializeCompressed())
	cases := []struct {
		id       string
		raw      []byte
		code     tx.Code
		eligible bool
	}{
		{"V01", valid, 0, true}, {"V02", badR, 4, false}, {"V03", highS, 4, false},
		{"V04", modifiedAmount, 4, true}, {"V05", signedB0(t, "pluto-other-1", receiver, 25, 0), 2, true},
		{"V06", badKey, 3, true},
		{"V07-bad-magic", badMagic, 1, false}, {"V07-version", badVersion, 1, false},
		{"V07-short", short, 1, false}, {"V07-trailing", long, 1, false},
		{"V07-chain-length-zero", badLength0, 1, false}, {"V07-chain-length-51", badLength51, 1, false},
		{"V07-chain-byte", badChainByte, 1, false},
		{"V07-65-byte-key", uncompressedKey, 1, false},
		{"V08", signedB0(t, chain, receiver, 25, 1), 6, true},
		{"V10", signedB0(t, chain, receiver, 101, 0), 7, true},
		{"V11", signedB0(t, chain, receiver, 0, 0), 5, true},
		{"V12", signedB0(t, chain, receiver, 100, 0), 0, true},
		{"V16-binding", badBinding, 4, true},
		{"V16-unfunded-B", signedB0Key(t, 2, chain, receiver, 1, 0), 7, true},
		{"V24", zeroAndBadSignature, 5, true},
	}
	for _, vector := range cases {
		t.Run(vector.id, func(t *testing.T) {
			a, _, _, _ := appWithBalance(t, chain, 100)
			defer a.Close()
			ctx := context.Background()
			id, idErr := tx.TxID(vector.raw)
			if (idErr == nil) != vector.eligible {
				t.Fatalf("TxID eligibility: %v", idErr)
			}
			check, err := a.CheckTx(ctx, &abci.RequestCheckTx{Tx: vector.raw})
			if err != nil || check.Code != uint32(vector.code) || check.Log != resultLog(vector.code) {
				t.Fatalf("CheckTx: %+v %v", check, err)
			}
			proposal, err := a.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: [][]byte{vector.raw}})
			if err != nil {
				t.Fatal(err)
			}
			wanted := abci.ResponseProcessProposal_REJECT
			if vector.code == 0 {
				wanted = abci.ResponseProcessProposal_ACCEPT
			}
			if proposal.Status != wanted {
				t.Fatalf("ProcessProposal: %+v", proposal)
			}
			before, _, err := a.Committed()
			if err != nil {
				t.Fatal(err)
			}
			final, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{vector.raw}})
			if err != nil || len(final.TxResults) != 1 || final.TxResults[0].Code != uint32(vector.code) || final.TxResults[0].Log != resultLog(vector.code) {
				t.Fatalf("FinalizeBlock: %+v %v", final, err)
			}
			pending, exists, err := a.store.Pending()
			if err != nil || !exists {
				t.Fatalf("pending: %v", err)
			}
			expected, code, err := b0state.Evaluate(before, vector.raw)
			if err != nil || code != vector.code {
				t.Fatalf("reference Evaluate code=%d err=%v", code, err)
			}
			expected, err = expected.AdvanceBlock()
			if err != nil {
				t.Fatal(err)
			}
			pendingHash := pending.AppHash()
			if pendingHash != expected.AppHash() || !bytes.Equal(final.AppHash, pendingHash[:]) {
				t.Fatalf("pending AppHash mismatch")
			}
			if _, err := a.Commit(ctx, &abci.RequestCommit{}); err != nil {
				t.Fatal(err)
			}
			committed, _, err := a.Committed()
			if err != nil || committed.AppHash() != expected.AppHash() {
				t.Fatalf("committed state: %v", err)
			}
			hash := committed.AppHash()
			t.Logf("%s wire_hex=%s expected_code=%d txid_eligible=%t pluto_txid=%x comet_hash=%x committed_apphash_h1=%x", vector.id, hex.EncodeToString(vector.raw), vector.code, vector.eligible, id, tx.CometHash(vector.raw), hash)
		})
	}
}

// Inject only a complete, integrity-valid synthetic state into a disposable
// D-024 store. This is an ABCI contract fixture, not a real consensus node.
func syntheticApp(t testing.TB, genesisBalance uint64, state b0state.State) *App {
	t.Helper()
	sender, _ := testAddresses(t)
	genesis, err := b0state.EncodeGenesis("pluto-b0-v1", []b0state.Allocation{{Address: sender, Balance: genesisBalance}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "b0")
	store, err := b0store.Create(path, genesis, "pluto-b0-v1")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	logical, err := state.LogicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	hash := state.AppHash()
	record := append(append([]byte("PLUTOS2!"), logical...), hash[:]...)
	db, err := pebble.Open(path, &pebble.Options{ErrorIfNotExists: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Set([]byte("pluto/committed-state/v2"), record, pebble.Sync); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	a, err := Open(path, "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestB0AcceptedSyntheticBoundaryVectors(t *testing.T) {
	sender, receiver := testAddresses(t)
	maxNonceState, err := b0state.NewState("pluto-b0-v1", 1, 100, []b0state.Account{{Address: sender, Balance: 100, Nonce: math.MaxUint64}})
	if err != nil {
		t.Fatal(err)
	}
	zeroBalanceState, err := b0state.NewState("pluto-b0-v1", 1, 1, []b0state.Account{{Address: sender, Nonce: math.MaxUint64}, {Address: receiver, Balance: 1}})
	if err != nil {
		t.Fatal(err)
	}
	accounts := make([]b0state.Account, 65536)
	accounts[0] = b0state.Account{Address: sender, Balance: 1}
	for i := 1; i < len(accounts); i++ {
		binary.BigEndian.PutUint32(accounts[i].Address[16:], uint32(i-1))
	}
	sort.Slice(accounts, func(i, j int) bool { return bytes.Compare(accounts[i].Address[:], accounts[j].Address[:]) < 0 })
	capState, err := b0state.NewState("pluto-b0-v1", 1, 1, accounts)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		id      string
		state   b0state.State
		balance uint64
		nonce   uint64
		amount  uint64
		want    uint32
	}{
		{"V17", maxNonceState, 100, math.MaxUint64, 1, 6},
		{"V19", capState, 1, 0, 1, 9},
		{"V25", zeroBalanceState, 1, math.MaxUint64, 1, 6},
	} {
		t.Run(item.id, func(t *testing.T) {
			a := syntheticApp(t, item.balance, item.state)
			defer a.Close()
			raw := signedB0(t, "pluto-b0-v1", receiver, item.amount, item.nonce)
			check, err := a.CheckTx(context.Background(), &abci.RequestCheckTx{Tx: raw})
			if err != nil || check.Code != item.want {
				t.Fatalf("CheckTx: %+v %v", check, err)
			}
			proposal, err := a.ProcessProposal(context.Background(), &abci.RequestProcessProposal{Height: 2, Txs: [][]byte{raw}})
			if err != nil || proposal.Status != abci.ResponseProcessProposal_REJECT {
				t.Fatalf("ProcessProposal: %+v %v", proposal, err)
			}
			final, err := a.FinalizeBlock(context.Background(), &abci.RequestFinalizeBlock{Height: 2, Txs: [][]byte{raw}})
			if err != nil || final.TxResults[0].Code != item.want {
				t.Fatalf("FinalizeBlock: %+v %v", final, err)
			}
			pending, _, err := a.store.Pending()
			beforeAccounts, afterAccounts := item.state.Accounts(), pending.Accounts()
			if err != nil || !slices.Equal(beforeAccounts, afterAccounts) || pending.TotalSupply() != item.state.TotalSupply() || pending.Height() != 2 {
				t.Fatalf("state changed on failed tx: %v", err)
			}
			t.Logf("%s synthetic_integrity_valid_h1=%x wire_hex=%s code=%d pending_apphash_h2=%x", item.id, item.state.AppHash(), hex.EncodeToString(raw), item.want, pending.AppHash())
		})
	}
}

func TestB0AcceptedOrderedVectors(t *testing.T) {
	const chain = "pluto-b0-v1"
	sender, receiver := testAddresses(t)
	first := signedB0(t, chain, receiver, 25, 0)
	second := signedB0(t, chain, receiver, 30, 1)
	insufficient := signedB0(t, chain, receiver, 101, 0)
	self := signedB0(t, chain, sender, 25, 0)
	for _, item := range []struct {
		id                         string
		raws                       [][]byte
		codes                      []uint32
		balanceA, nonceA, balanceB uint64
		accept                     bool
	}{
		{"V13", [][]byte{first, first}, []uint32{0, 6}, 75, 1, 25, false},
		{"V14", [][]byte{first, second}, []uint32{0, 0}, 45, 2, 55, true},
		{"V15", [][]byte{{1}, first}, []uint32{1, 0}, 75, 1, 25, false},
		{"V21", [][]byte{insufficient, first}, []uint32{7, 0}, 75, 1, 25, false},
		{"V22", [][]byte{self}, []uint32{0}, 100, 1, 0, true},
	} {
		t.Run(item.id, func(t *testing.T) {
			a, _, _, _ := appWithBalance(t, chain, 100)
			defer a.Close()
			ctx := context.Background()
			proposal, err := a.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1, Txs: item.raws})
			if err != nil {
				t.Fatal(err)
			}
			want := abci.ResponseProcessProposal_REJECT
			if item.accept {
				want = abci.ResponseProcessProposal_ACCEPT
			}
			if proposal.Status != want {
				t.Fatalf("proposal status: %+v", proposal)
			}
			final, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1, Txs: item.raws})
			if err != nil || len(final.TxResults) != len(item.codes) {
				t.Fatalf("final: %+v %v", final, err)
			}
			for i, code := range item.codes {
				if final.TxResults[i].Code != code {
					t.Fatalf("tx %d code %d want %d", i, final.TxResults[i].Code, code)
				}
			}
			if _, err := a.Commit(ctx, &abci.RequestCommit{}); err != nil {
				t.Fatal(err)
			}
			state, _, err := a.Committed()
			if err != nil {
				t.Fatal(err)
			}
			aAccount, _ := state.Account(sender)
			bAccount, bFound := state.Account(receiver)
			if aAccount.Balance != item.balanceA || aAccount.Nonce != item.nonceA || bAccount.Balance != item.balanceB || state.TotalSupply() != 100 {
				t.Fatalf("accounts: A=%+v B=%+v", aAccount, bAccount)
			}
			if item.id == "V22" && bFound {
				t.Fatal("self transfer created receiver")
			}
		})
	}
}

func TestB0V18GenesisSupplyOverflowAndV23CorruptRecord(t *testing.T) {
	sender, receiver := testAddresses(t)
	const chain = "pluto-b0-v1"
	genesis, err := b0state.EncodeGenesis(chain, []b0state.Allocation{{Address: sender, Balance: 100}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "b0")
	a, err := Open(path, chain, genesis)
	if err != nil {
		t.Fatal(err)
	}
	// Construct a canonical-looking two-allocation binary whose checked
	// total-supply sum overflows uint64. The accepted encoder must reject it.
	bad := make([]byte, 8+4+2+len(chain)+4+2*(20+8))
	copy(bad[:8], "PLUTOG2!")
	binary.BigEndian.PutUint32(bad[8:12], 2)
	binary.BigEndian.PutUint16(bad[12:14], uint16(len(chain)))
	copy(bad[14:], chain)
	p := 14 + len(chain)
	binary.BigEndian.PutUint32(bad[p:p+4], 2)
	p += 4
	copy(bad[p:p+20], sender[:])
	p += 20
	binary.BigEndian.PutUint64(bad[p:p+8], math.MaxUint64)
	p += 8
	copy(bad[p:p+20], receiver[:])
	p += 20
	binary.BigEndian.PutUint64(bad[p:p+8], 1)
	encoded, err := json.Marshal(base64.StdEncoding.EncodeToString(bad))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.InitChain(context.Background(), &abci.RequestInitChain{InitialHeight: 1, ChainId: chain, AppStateBytes: encoded}); err == nil {
		t.Fatal("V18 overflowing genesis accepted")
	}
	if _, exists, err := a.Committed(); err != nil || exists {
		t.Fatalf("V18 created state: exists=%t err=%v", exists, err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := b0store.Create(path, genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := pebble.Open(path, &pebble.Options{ErrorIfNotExists: true})
	if err != nil {
		t.Fatal(err)
	}
	value, closer, err := db.Get([]byte("pluto/committed-state/v2"))
	if err != nil {
		t.Fatal(err)
	}
	corrupt := bytes.Clone(value)
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}
	corrupt[len(corrupt)-33] ^= 1 // logical nonce byte; retain old digest
	if err := db.Set([]byte("pluto/committed-state/v2"), corrupt, pebble.Sync); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(path, chain, genesis); err == nil {
		_ = reopened.Close()
		t.Fatal("V23 corrupt complete record accepted")
	}
	t.Log("V18 InitChain rejected overflowing supply without creating a store; V23 application Open rejected modified logical state with unchanged digest")
}
