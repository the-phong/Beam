//go:build ignore

// Run from repository root. Scalars A=1 and B=2 are public TEST ONLY keys.
package main

import (
	"encoding/hex"
	"encoding/json"
	"os"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/tx"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

type phase03BVector struct {
	Wire     string `json:"wire_hex"`
	Sender   string `json:"sender_address_hex"`
	Receiver string `json:"receiver_hex"`
}

type account struct {
	Address string `json:"address_hex"`
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}
type snapshot struct {
	Chain    string    `json:"chain_id"`
	Height   int64     `json:"height"`
	Supply   uint64    `json:"total_supply"`
	Accounts []account `json:"accounts"`
	Logical  string    `json:"logical_hex"`
	AppHash  string    `json:"apphash_hex"`
}
type negative struct {
	ID            string  `json:"id"`
	Starting      string  `json:"starting_snapshot"`
	Wire          string  `json:"wire_hex"`
	Code          tx.Code `json:"expected_code"`
	UnchangedHash string  `json:"unchanged_apphash_hex"`
}
type vectors struct {
	FixtureKeys         string     `json:"fixture_keys"`
	GenesisWire         string     `json:"genesis_wire_hex"`
	Genesis             snapshot   `json:"genesis"`
	TransferWire        string     `json:"transfer_wire_hex"`
	PostTransfer        snapshot   `json:"post_transfer"`
	SelfWire            string     `json:"self_wire_hex"`
	PostSelf            snapshot   `json:"post_self_transfer"`
	PostEmpty           snapshot   `json:"post_empty_block"`
	MinChainGenesisWire string     `json:"min_chain_genesis_wire_hex"`
	MinChainGenesis     snapshot   `json:"min_chain_genesis"`
	MaxChainGenesisWire string     `json:"max_chain_genesis_wire_hex"`
	MaxChainGenesis     snapshot   `json:"max_chain_genesis"`
	Negatives           []negative `json:"negative_cases"`
}

func h(b []byte) string { return hex.EncodeToString(b) }
func snap(s b0state.State) snapshot {
	logical, err := s.LogicalBytes()
	if err != nil {
		panic(err)
	}
	accounts := make([]account, 0, len(s.Accounts()))
	for _, a := range s.Accounts() {
		accounts = append(accounts, account{h(a.Address[:]), a.Balance, a.Nonce})
	}
	hash := s.AppHash()
	return snapshot{s.ChainID(), s.Height(), s.TotalSupply(), accounts, h(logical), h(hash[:])}
}
func signA(chain string, receiver b0state.Address, amount, nonce uint64) []byte {
	var secret [32]byte
	secret[31] = 1
	key := secp256k1.PrivKeyFromBytes(secret[:])
	var record tx.Transaction
	record.ChainID = chain
	copy(record.Sender[:], key.PubKey().SerializeCompressed())
	copy(record.Receiver[:], receiver[:])
	record.Amount, record.Nonce = amount, nonce
	digest, err := tx.SigningDigest(record)
	if err != nil {
		panic(err)
	}
	sig := ecdsa.Sign(key, digest[:])
	r, s := sig.R(), sig.S()
	r.PutBytes(&record.R)
	s.PutBytes(&record.S)
	wire, err := tx.Encode(record)
	if err != nil {
		panic(err)
	}
	return wire
}
func main() {
	data, err := os.ReadFile("research/evidence/phase_03b/golden_vectors.json")
	if err != nil {
		panic(err)
	}
	var old []phase03BVector
	if err := json.Unmarshal(data, &old); err != nil {
		panic(err)
	}
	var a, b b0state.Address
	aBytes, _ := hex.DecodeString(old[0].Sender)
	bBytes, _ := hex.DecodeString(old[0].Receiver)
	copy(a[:], aBytes)
	copy(b[:], bBytes)
	transfer, err := hex.DecodeString(old[0].Wire)
	if err != nil {
		panic(err)
	}
	genesisWire, err := b0state.EncodeGenesis("pluto-b0-v1", []b0state.Allocation{{Address: a, Balance: 100}})
	if err != nil {
		panic(err)
	}
	genesis, err := b0state.ParseGenesis(genesisWire, "pluto-b0-v1")
	if err != nil {
		panic(err)
	}
	postTransfer, code, err := b0state.Evaluate(genesis, transfer)
	if err != nil || code != 0 {
		panic("transfer")
	}
	postTransfer, err = postTransfer.AdvanceBlock()
	if err != nil {
		panic(err)
	}
	self := signA("pluto-b0-v1", a, 25, 1)
	postSelf, code, err := b0state.Evaluate(postTransfer, self)
	if err != nil || code != 0 {
		panic("self")
	}
	postSelf, err = postSelf.AdvanceBlock()
	if err != nil {
		panic(err)
	}
	postEmpty, err := postSelf.AdvanceBlock()
	if err != nil {
		panic(err)
	}
	minWire, _ := b0state.EncodeGenesis("c", []b0state.Allocation{{Address: a, Balance: 100}})
	minState, _ := b0state.ParseGenesis(minWire, "c")
	maxChain := "cccccccccccccccccccccccccccccccccccccccccccccccccc"
	maxWire, _ := b0state.EncodeGenesis(maxChain, []b0state.Allocation{{Address: a, Balance: 100}})
	maxState, _ := b0state.ParseGenesis(maxWire, maxChain)
	replay, code, err := b0state.Evaluate(postTransfer, transfer)
	if err != nil || code != b0state.CodeInvalidNonce {
		panic("replay")
	}
	if replay.AppHash() != postTransfer.AppHash() {
		panic("replay changed state")
	}
	insufficient := signA("pluto-b0-v1", b, 101, 0)
	_, code, err = b0state.Evaluate(genesis, insufficient)
	if err != nil || code != b0state.CodeInsufficientBalance {
		panic("insufficient")
	}
	out := vectors{FixtureKeys: "PUBLIC TEST ONLY: A=1, B=2", GenesisWire: h(genesisWire), Genesis: snap(genesis),
		TransferWire: h(transfer), PostTransfer: snap(postTransfer), SelfWire: h(self), PostSelf: snap(postSelf), PostEmpty: snap(postEmpty),
		MinChainGenesisWire: h(minWire), MinChainGenesis: snap(minState), MaxChainGenesisWire: h(maxWire), MaxChainGenesis: snap(maxState),
		Negatives: []negative{{"replay", "post_transfer", h(transfer), b0state.CodeInvalidNonce, snap(postTransfer).AppHash},
			{"insufficient_balance", "genesis", h(insufficient), b0state.CodeInsufficientBalance, snap(genesis).AppHash}}}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
}
