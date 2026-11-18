//go:build ignore

// Run from the repository root: go run research/evidence/phase_03b/generate_vectors.go
// The scalars 1 and 2 are public TEST ONLY keys, never wallet or node keys.
package main

import (
	"encoding/hex"
	"encoding/json"
	"os"

	"example.com/pluto/internal/tx"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

type vector struct {
	ID           string  `json:"id"`
	KeyID        string  `json:"test_only_key_id"`
	ChainID      string  `json:"chain_id"`
	Amount       uint64  `json:"amount"`
	Nonce        uint64  `json:"nonce"`
	PubKey       string  `json:"compressed_pubkey_hex"`
	Sender       string  `json:"sender_address_hex"`
	Receiver     string  `json:"receiver_hex"`
	R            string  `json:"r_hex"`
	S            string  `json:"s_hex"`
	Unsigned     string  `json:"unsigned_payload_hex"`
	Preimage     string  `json:"signing_preimage_hex"`
	Digest       string  `json:"signing_digest_hex"`
	Wire         string  `json:"wire_hex"`
	TxIDEligible bool    `json:"txid_eligible"`
	TxID         string  `json:"pluto_txid_hex"`
	CometHash    string  `json:"comet_hash_hex"`
	Code         tx.Code `json:"expected_stateless_code"`
}

func makeVector(id, chain string, amount, nonce uint64) vector {
	var aSecret, bSecret [32]byte
	aSecret[31], bSecret[31] = 1, 2
	aKey := secp256k1.PrivKeyFromBytes(aSecret[:])
	bKey := secp256k1.PrivKeyFromBytes(bSecret[:])
	var sender, receiver [33]byte
	copy(sender[:], aKey.PubKey().SerializeCompressed())
	copy(receiver[:], bKey.PubKey().SerializeCompressed())
	aAddress, err := tx.SenderAddress(sender)
	if err != nil {
		panic(err)
	}
	bAddress, err := tx.SenderAddress(receiver)
	if err != nil {
		panic(err)
	}
	record := tx.Transaction{ChainID: chain, Sender: sender, Receiver: bAddress, Amount: amount, Nonce: nonce}
	preimage, err := tx.SigningPreimage(record)
	if err != nil {
		panic(err)
	}
	digest, err := tx.SigningDigest(record)
	if err != nil {
		panic(err)
	}
	sig := ecdsa.Sign(aKey, digest[:])
	r, s := sig.R(), sig.S()
	r.PutBytes(&record.R)
	s.PutBytes(&record.S)
	wire, err := tx.Encode(record)
	if err != nil {
		panic(err)
	}
	idHash, err := tx.TxID(wire)
	if err != nil {
		panic(err)
	}
	cometHash := tx.CometHash(wire)
	return vector{ID: id, KeyID: "TEST_ONLY_A=1; TEST_ONLY_B=2", ChainID: chain,
		Amount: amount, Nonce: nonce, PubKey: hex.EncodeToString(sender[:]),
		Sender: hex.EncodeToString(aAddress[:]), Receiver: hex.EncodeToString(bAddress[:]),
		R: hex.EncodeToString(record.R[:]), S: hex.EncodeToString(record.S[:]),
		Unsigned: hex.EncodeToString(wire[:len(wire)-64]), Preimage: hex.EncodeToString(preimage),
		Digest: hex.EncodeToString(digest[:]), Wire: hex.EncodeToString(wire),
		TxIDEligible: true, TxID: hex.EncodeToString(idHash[:]), CometHash: hex.EncodeToString(cometHash[:]),
		Code: tx.Validate(wire, chain)}
}

func main() {
	vectors := []vector{
		makeVector("V01", "pluto-b0-v1", 25, 0),
		makeVector("V11", "pluto-b0-v1", 0, 0),
		makeVector("boundary-min-chain", "c", 1, 0),
		makeVector("boundary-max-chain", "cccccccccccccccccccccccccccccccccccccccccccccccccc", 1, 0),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(vectors); err != nil {
		panic(err)
	}
}
