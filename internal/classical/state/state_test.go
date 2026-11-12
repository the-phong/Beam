package b0state

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"testing"

	"example.com/pluto/internal/classical/tx"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

const chain = "pluto-b0-v1"

type fixture struct {
	ID       string `json:"id"`
	Wire     string `json:"wire_hex"`
	Sender   string `json:"sender_address_hex"`
	Receiver string `json:"receiver_hex"`
}

func baseFixture(t testing.TB) (Address, Address, []byte) {
	t.Helper()
	data, err := os.ReadFile("../../../research/evidence/phase_03b/golden_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []fixture
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) < 1 || vectors[0].ID != "V01" {
		t.Fatal("missing V01")
	}
	var a, b Address
	ab, _ := hex.DecodeString(vectors[0].Sender)
	bb, _ := hex.DecodeString(vectors[0].Receiver)
	copy(a[:], ab)
	copy(b[:], bb)
	wire, err := hex.DecodeString(vectors[0].Wire)
	if err != nil {
		t.Fatal(err)
	}
	return a, b, wire
}

func genesis(t *testing.T) (State, Address, Address, []byte) {
	t.Helper()
	a, b, wire := baseFixture(t)
	encoded, err := EncodeGenesis(chain, []Allocation{{Address: a, Balance: 100}})
	if err != nil {
		t.Fatal(err)
	}
	s, err := ParseGenesis(encoded, chain)
	if err != nil {
		t.Fatal(err)
	}
	return s, a, b, wire
}

func signed(t testing.TB, keyByte byte, receiver Address, amount, nonce uint64) []byte {
	t.Helper()
	var secret [32]byte
	secret[31] = keyByte // explicitly public TEST ONLY scalar 1 or 2.
	key := secp256k1.PrivKeyFromBytes(secret[:])
	var record tx.Transaction
	record.ChainID = chain
	copy(record.Sender[:], key.PubKey().SerializeCompressed())
	copy(record.Receiver[:], receiver[:])
	record.Amount, record.Nonce = amount, nonce
	digest, err := tx.SigningDigest(record)
	if err != nil {
		t.Fatal(err)
	}
	sig := ecdsa.Sign(key, digest[:])
	r, s := sig.R(), sig.S()
	r.PutBytes(&record.R)
	s.PutBytes(&record.S)
	wire, err := tx.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func equalState(t *testing.T, got, want State) {
	t.Helper()
	a, e1 := got.LogicalBytes()
	b, e2 := want.LogicalBytes()
	if e1 != nil || e2 != nil || !bytes.Equal(a, b) || got.AppHash() != want.AppHash() {
		t.Fatalf("states differ: %v %v", e1, e2)
	}
}

func TestGenesisCanonicalAndBoundaries(t *testing.T) {
	s, a, _, _ := genesis(t)
	if s.Height() != 0 || s.TotalSupply() != 100 || len(s.Accounts()) != 1 || s.Accounts()[0].Nonce != 0 {
		t.Fatal("genesis state")
	}
	for _, id := range []string{"c", string(bytes.Repeat([]byte{'c'}, 50))} {
		wire, err := EncodeGenesis(id, []Allocation{{Address: a, Balance: 1}})
		if err != nil || len(wire) != 18+len(id)+28 {
			t.Fatal("genesis length", err)
		}
		parsed, err := ParseGenesis(wire, id)
		if err != nil || parsed.ChainID() != id || parsed.Height() != 0 {
			t.Fatal("genesis parse", err)
		}
	}
	copyAccounts := s.Accounts()
	copyAccounts[0].Balance = 0
	if s.Accounts()[0].Balance != 100 {
		t.Fatal("account slice escaped")
	}
	logical, err := s.LogicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeLogical(logical, s.AppHash(), chain)
	if err != nil {
		t.Fatal(err)
	}
	equalState(t, decoded, s)
}

func TestIndependentGoldenVectors(t *testing.T) {
	data, err := os.ReadFile("../../../research/evidence/phase_03c/state_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		GenesisWire  string `json:"genesis_wire_hex"`
		TransferWire string `json:"transfer_wire_hex"`
		SelfWire     string `json:"self_wire_hex"`
		Genesis      struct {
			Logical string `json:"logical_hex"`
			Hash    string `json:"apphash_hex"`
		} `json:"genesis"`
		PostTransfer struct {
			Logical string `json:"logical_hex"`
			Hash    string `json:"apphash_hex"`
		} `json:"post_transfer"`
		PostSelf struct {
			Logical string `json:"logical_hex"`
			Hash    string `json:"apphash_hex"`
		} `json:"post_self_transfer"`
		PostEmpty struct {
			Logical string `json:"logical_hex"`
			Hash    string `json:"apphash_hex"`
		} `json:"post_empty_block"`
	}
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	s, a, _, _ := genesis(t)
	genesisWire, err := EncodeGenesis(chain, []Allocation{{Address: a, Balance: 100}})
	if err != nil || hex.EncodeToString(genesisWire) != vectors.GenesisWire {
		t.Fatal("genesis wire", err)
	}
	check := func(label string, state State, logicalHex, hashHex string) {
		t.Helper()
		logical, err := state.LogicalBytes()
		if err != nil || hex.EncodeToString(logical) != logicalHex || hex.EncodeToString(state.hash[:]) != hashHex {
			t.Fatal(label, err)
		}
	}
	check("genesis", s, vectors.Genesis.Logical, vectors.Genesis.Hash)
	transfer, err := hex.DecodeString(vectors.TransferWire)
	if err != nil {
		t.Fatal(err)
	}
	s, code, err := Evaluate(s, transfer)
	if err != nil || code != 0 {
		t.Fatal(code, err)
	}
	s, err = s.AdvanceBlock()
	if err != nil {
		t.Fatal(err)
	}
	check("transfer", s, vectors.PostTransfer.Logical, vectors.PostTransfer.Hash)
	self, err := hex.DecodeString(vectors.SelfWire)
	if err != nil {
		t.Fatal(err)
	}
	s, code, err = Evaluate(s, self)
	if err != nil || code != 0 {
		t.Fatal(code, err)
	}
	s, err = s.AdvanceBlock()
	if err != nil {
		t.Fatal(err)
	}
	check("self", s, vectors.PostSelf.Logical, vectors.PostSelf.Hash)
	s, err = s.AdvanceBlock()
	if err != nil {
		t.Fatal(err)
	}
	check("empty", s, vectors.PostEmpty.Logical, vectors.PostEmpty.Hash)
}

func TestGenesisRejections(t *testing.T) {
	_, a, b, _ := genesis(t)
	good, _ := EncodeGenesis(chain, []Allocation{{Address: a, Balance: 100}})
	invalid := map[string]func([]byte) []byte{
		"magic":             func(w []byte) []byte { w[0] = 0; return w },
		"version":           func(w []byte) []byte { w[11] = 3; return w },
		"zero chain length": func(w []byte) []byte { w[13] = 0; return w },
		"long chain length": func(w []byte) []byte { w[13] = 51; return w },
		"bad chain byte":    func(w []byte) []byte { w[14] = 0xff; return w },
		"zero count":        func(w []byte) []byte { binary.BigEndian.PutUint32(w[14+len(chain):], 0); return w },
		"over cap":          func(w []byte) []byte { binary.BigEndian.PutUint32(w[14+len(chain):], 65537); return w },
		"zero balance":      func(w []byte) []byte { clear(w[len(w)-8:]); return w },
		"truncated":         func(w []byte) []byte { return w[:len(w)-1] },
		"trailing":          func(w []byte) []byte { return append(w, 0) },
	}
	for name, change := range invalid {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseGenesis(change(bytes.Clone(good)), chain); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	if _, err := ParseGenesis(good, "other"); err == nil {
		t.Fatal("wrong chain")
	}
	if _, err := ParseGenesis(make([]byte, 2), chain); err == nil {
		t.Fatal("short")
	}
	if _, err := EncodeGenesis("", []Allocation{{Address: a, Balance: 1}}); err == nil {
		t.Fatal("empty chain")
	}
	if _, err := EncodeGenesis(chain, nil); err == nil {
		t.Fatal("empty allocations")
	}
	if _, err := EncodeGenesis(chain, []Allocation{{Address: a, Balance: 0}}); err == nil {
		t.Fatal("zero allocation")
	}
	if _, err := EncodeGenesis(chain, []Allocation{{Address: b, Balance: 1}, {Address: a, Balance: 1}}); err == nil {
		t.Fatal("unsorted")
	}
	if _, err := EncodeGenesis(chain, []Allocation{{Address: a, Balance: 1}, {Address: a, Balance: 1}}); err == nil {
		t.Fatal("duplicate")
	}
	if _, err := EncodeGenesis(chain, []Allocation{{Address: a, Balance: math.MaxUint64}, {Address: b, Balance: 1}}); err == nil {
		t.Fatal("overflow")
	}
	unsorted := append(bytes.Clone(good), make([]byte, 28)...)
	binary.BigEndian.PutUint32(unsorted[14+len(chain):], 2)
	copy(unsorted[len(good):len(good)+20], b[:])
	binary.BigEndian.PutUint64(unsorted[len(good)+20:], 1)
	copy(unsorted[18+len(chain):18+len(chain)+20], b[:]) // duplicate and unsorted
	if _, err := ParseGenesis(unsorted, chain); err == nil {
		t.Fatal("noncanonical allocation order")
	}
}

func TestGenesisMaximumAccountCount(t *testing.T) {
	allocations := make([]Allocation, MaxAccounts)
	for i := range allocations {
		binary.BigEndian.PutUint32(allocations[i].Address[16:], uint32(i))
		allocations[i].Balance = 1
	}
	wire, err := EncodeGenesis("c", allocations)
	if err != nil || len(wire) != 18+1+28*MaxAccounts {
		t.Fatal("maximum genesis encode", err)
	}
	state, err := ParseGenesis(wire, "c")
	if err != nil || len(state.Accounts()) != MaxAccounts || state.TotalSupply() != MaxAccounts {
		t.Fatal("maximum genesis parse", err)
	}
	tooMany := append(allocations, Allocation{Balance: 1})
	if _, err := EncodeGenesis("c", tooMany); err != ErrGenesis {
		t.Fatal("over-cap genesis accepted", err)
	}
}

func TestTransfersSequentialAndReplay(t *testing.T) {
	s0, a, b, v01 := genesis(t)
	s1, code, err := Evaluate(s0, v01)
	if err != nil || code != 0 {
		t.Fatalf("V01 %d %v", code, err)
	}
	aa, _ := s1.Account(a)
	bb, _ := s1.Account(b)
	if aa.Balance != 75 || aa.Nonce != 1 || bb.Balance != 25 || bb.Nonce != 0 || s1.TotalSupply() != 100 {
		t.Fatal("V01 state")
	}
	if original, _ := s0.Account(a); original.Balance != 100 || len(s0.Accounts()) != 1 {
		t.Fatal("committed snapshot changed")
	}
	if _, code, err := Evaluate(s1, v01); err != nil || code != CodeInvalidNonce {
		t.Fatal("replay", code, err)
	}
	second := signed(t, 1, b, 30, 1)
	ordered, codes, err := SimulateOrdered(s0, [][]byte{v01, second})
	if err != nil || len(codes) != 2 || codes[0] != 0 || codes[1] != 0 {
		t.Fatal("ordered", codes, err)
	}
	aa, _ = ordered.Account(a)
	bb, _ = ordered.Account(b)
	if aa.Balance != 45 || aa.Nonce != 2 || bb.Balance != 55 {
		t.Fatal("ordered state")
	}
	failed := signed(t, 1, b, 101, 0)
	end, codes, err := SimulateOrdered(s0, [][]byte{failed, v01})
	if err != nil || codes[0] != CodeInsufficientBalance || codes[1] != 0 {
		t.Fatal("failed then valid", codes, err)
	}
	equalState(t, end, s1)
	independent, _, _ := SimulateOrdered(s0, [][]byte{v01, second})
	equalState(t, independent, ordered)
	// A newly funded B spends with its own key and nonce zero.
	spend := signed(t, 2, a, 5, 0)
	end, codes, err = SimulateOrdered(s0, [][]byte{v01, spend})
	if err != nil || codes[0] != 0 || codes[1] != 0 {
		t.Fatal("newly funded sender", codes, err)
	}
	aa, _ = end.Account(a)
	bb, _ = end.Account(b)
	if aa.Balance != 80 || aa.Nonce != 1 || bb.Balance != 20 || bb.Nonce != 1 {
		t.Fatal("funded spend")
	}
}

func TestTransitionBoundaries(t *testing.T) {
	s0, a, b, _ := genesis(t)
	cases := []struct {
		name string
		wire []byte
		code tx.Code
	}{
		{"future nonce", signed(t, 1, b, 1, 1), CodeInvalidNonce},
		{"insufficient", signed(t, 1, b, 101, 0), CodeInsufficientBalance},
		{"zero amount", signed(t, 1, b, 0, 0), tx.CodeZeroAmount},
		{"absent sender", signed(t, 2, a, 1, 0), CodeInsufficientBalance},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			next, code, err := Evaluate(s0, tc.wire)
			if err != nil || code != tc.code {
				t.Fatal(code, err)
			}
			equalState(t, next, s0)
		})
	}
	exact := signed(t, 1, b, 100, 0)
	end, code, err := Evaluate(s0, exact)
	if err != nil || code != 0 {
		t.Fatal(code, err)
	}
	aa, _ := end.Account(a)
	if aa.Balance != 0 || aa.Nonce != 1 {
		t.Fatal("zero-balance sender removed")
	}
	self := signed(t, 1, a, 25, 0)
	end, code, err = Evaluate(s0, self)
	if err != nil || code != 0 || len(end.Accounts()) != 1 {
		t.Fatal("self transfer", code, err)
	}
	aa, _ = end.Account(a)
	if aa.Balance != 100 || aa.Nonce != 1 {
		t.Fatal("self state")
	}
	maxNonce, _ := NewState(chain, 0, 1, []Account{{Address: a, Balance: 0, Nonce: math.MaxUint64}, {Address: b, Balance: 1}})
	end, code, err = Evaluate(maxNonce, signed(t, 1, b, 1, math.MaxUint64))
	if err != nil || code != CodeInvalidNonce {
		t.Fatal("nonce precedence", code, err)
	}
	equalState(t, end, maxNonce)
	maxHeight, _ := NewState(chain, math.MaxInt64, 100, []Account{{Address: a, Balance: 100}})
	if _, err := maxHeight.AdvanceBlock(); err != ErrHeightOverflow {
		t.Fatal("height overflow", err)
	}
	advanced, err := s0.AdvanceBlock()
	if err != nil || advanced.Height() != 1 || advanced.AppHash() == s0.AppHash() {
		t.Fatal("empty block")
	}
	if _, err := NewState(chain, 0, 2, []Account{{Address: a, Balance: 1}}); err == nil {
		t.Fatal("supply mismatch")
	}
	if _, err := NewState(chain, 0, 0, []Account{{Address: b}, {Address: a}}); err == nil {
		t.Fatal("unordered")
	}
}

func TestAccountCapAndCorruptLogical(t *testing.T) {
	_, a, b, _ := genesis(t)
	accounts := make([]Account, MaxAccounts)
	for i := 0; i < MaxAccounts-1; i++ {
		binary.BigEndian.PutUint32(accounts[i].Address[16:], uint32(i))
	}
	accounts[MaxAccounts-1] = Account{Address: a, Balance: 1}
	s, err := NewState(chain, 0, 1, accounts)
	if err != nil {
		t.Fatal(err)
	}
	next, code, err := Evaluate(s, signed(t, 1, b, 1, 0))
	if err != nil || code != CodeAccountLimit {
		t.Fatal(code, err)
	}
	equalState(t, next, s)
	logical, _ := s.LogicalBytes()
	hash := s.AppHash()
	if _, err := DecodeLogical(logical, hash, "wrong"); err == nil {
		t.Fatal("wrong chain")
	}
	for _, mutate := range []func([]byte){
		func(b []byte) { b[3] = 3 },
		func(b []byte) { b[5] = 0 },
		func(b []byte) { b[len(b)-1] ^= 1 },
		func(b []byte) { binary.BigEndian.PutUint32(b[6+len(chain)+16:], 0) },
	} {
		bad := bytes.Clone(logical)
		mutate(bad)
		if _, err := DecodeLogical(bad, hash, chain); err == nil {
			t.Fatal("accepted corrupted logical state")
		}
	}
	if _, err := DecodeLogical(logical[:len(logical)-1], hash, chain); err == nil {
		t.Fatal("truncated")
	}
	badHash := hash
	badHash[0] ^= 1
	if _, err := DecodeLogical(logical, badHash, chain); err == nil {
		t.Fatal("digest corruption")
	}
}

func FuzzGenesis(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("PLUTOG2!"))
	var a Address
	a[19] = 1
	good, _ := EncodeGenesis("c", []Allocation{{Address: a, Balance: 1}})
	f.Add(good)
	f.Fuzz(func(t *testing.T, raw []byte) {
		s, err := ParseGenesis(raw, "c")
		if err == nil {
			if e := s.Validate(); e != nil {
				t.Fatal(e)
			}
			allocs := make([]Allocation, 0, len(s.Accounts()))
			for _, a := range s.Accounts() {
				allocs = append(allocs, Allocation{Address: a.Address, Balance: a.Balance})
			}
			encoded, e := EncodeGenesis(s.ChainID(), allocs)
			if e != nil || !bytes.Equal(encoded, raw) {
				t.Fatal("noncanonical genesis")
			}
		}
	})
}

func FuzzLogical(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 2})
	f.Fuzz(func(t *testing.T, raw []byte) {
		var h [32]byte
		_, _ = DecodeLogical(raw, h, "c")
	})
}
