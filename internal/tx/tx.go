// Package tx implements the stateless B0 V1 transaction protocol. It is not
// connected to the Phase 02 ABCI application or its persisted state.
package tx

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

const (
	Magic       = "PLUTOTX1"
	Version     = uint16(1)
	MinWireSize = 146
	MaxWireSize = 195
	MaxChainID  = 50
	signDomain  = "PLUTO/B0/SIGN/V1\x00"
	addrDomain  = "PLUTO/B0/ADDR/V1\x00"
	idDomain    = "PLUTO/B0/TXID/V1\x00"
)

// Transaction holds exactly the fields represented by the V1 wire format.
// The fixed-size byte arrays preserve the original scalar encodings.
type Transaction struct {
	ChainID  string
	Sender   [33]byte
	Receiver [20]byte
	Amount   uint64
	Nonce    uint64
	R        [32]byte
	S        [32]byte
}

// Code is the accepted ABCI-compatible first-failure category for stateless
// validation. Codes 6 and above belong to later, stateful phases.
type Code uint32

const (
	CodeOK         Code = 0
	CodeMalformed  Code = 1
	CodeWrongChain Code = 2
	CodeBadKey     Code = 3
	CodeBadSig     Code = 4
	CodeZeroAmount Code = 5
)

var ErrMalformed = errors.New("malformed B0 V1 transaction")
var ErrInvalidKey = errors.New("invalid compressed secp256k1 public key")
var ErrIneligibleTxID = errors.New("wire is not eligible for a Pluto TxID")

func validChainID(s string) bool {
	if len(s) < 1 || len(s) > MaxChainID {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '_' || b == '-' || b == '.') {
			return false
		}
	}
	return true
}

// Parse checks only the exact structural V1 format. Key, scalar, amount,
// chain-context, and ECDSA checks have separate APIs and precedence.
func Parse(wire []byte) (Transaction, error) {
	var tx Transaction
	if len(wire) < MinWireSize || len(wire) > MaxWireSize ||
		string(wire[:8]) != Magic || binary.BigEndian.Uint16(wire[8:10]) != Version {
		return Transaction{}, ErrMalformed
	}
	n := int(binary.BigEndian.Uint16(wire[10:12]))
	if n < 1 || n > MaxChainID || len(wire) != 145+n || !validChainID(string(wire[12:12+n])) {
		return Transaction{}, ErrMalformed
	}
	tx.ChainID = string(wire[12 : 12+n])
	copy(tx.Sender[:], wire[12+n:45+n])
	copy(tx.Receiver[:], wire[45+n:65+n])
	tx.Amount = binary.BigEndian.Uint64(wire[65+n : 73+n])
	tx.Nonce = binary.BigEndian.Uint64(wire[73+n : 81+n])
	copy(tx.R[:], wire[81+n:113+n])
	copy(tx.S[:], wire[113+n:145+n])
	return tx, nil
}

// Encode emits one canonical V1 representation of the supplied fields.
// It checks structural fields only; Validate performs authentication.
func Encode(tx Transaction) ([]byte, error) {
	if !validChainID(tx.ChainID) {
		return nil, ErrMalformed
	}
	n := len(tx.ChainID)
	wire := make([]byte, 145+n)
	copy(wire[:8], Magic)
	binary.BigEndian.PutUint16(wire[8:10], Version)
	binary.BigEndian.PutUint16(wire[10:12], uint16(n))
	copy(wire[12:12+n], tx.ChainID)
	copy(wire[12+n:45+n], tx.Sender[:])
	copy(wire[45+n:65+n], tx.Receiver[:])
	binary.BigEndian.PutUint64(wire[65+n:73+n], tx.Amount)
	binary.BigEndian.PutUint64(wire[73+n:81+n], tx.Nonce)
	copy(wire[81+n:113+n], tx.R[:])
	copy(wire[113+n:145+n], tx.S[:])
	return wire, nil
}

// SigningPreimage returns the exact domain, u32be length, and unsigned
// payload. This never includes signature bytes.
func SigningPreimage(tx Transaction) ([]byte, error) {
	wire, err := Encode(tx)
	if err != nil {
		return nil, err
	}
	n := len(wire) - 64
	out := make([]byte, len(signDomain)+4+n)
	copy(out, signDomain)
	binary.BigEndian.PutUint32(out[len(signDomain):], uint32(n))
	copy(out[len(signDomain)+4:], wire[:n])
	return out, nil
}

func SigningDigest(tx Transaction) ([32]byte, error) {
	preimage, err := SigningPreimage(tx)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(preimage), nil
}

// SenderAddress validates a compressed SEC 1 point before deriving its
// domain-separated 20-byte Pluto address.
func SenderAddress(compressed [33]byte) ([20]byte, error) {
	if _, err := parseKey(compressed); err != nil {
		return [20]byte{}, err
	}
	digest := sha256.Sum256(append([]byte(addrDomain), compressed[:]...))
	var address [20]byte
	copy(address[:], digest[:20])
	return address, nil
}

// TxID is defined for structurally valid V1 bytes with canonical low-S
// scalar fields, independently of authentication or account state.
func TxID(wire []byte) ([32]byte, error) {
	tx, err := Parse(wire)
	if err != nil || !canonicalScalars(tx) {
		return [32]byte{}, ErrIneligibleTxID
	}
	preimage := make([]byte, len(idDomain)+4+len(wire))
	copy(preimage, idDomain)
	binary.BigEndian.PutUint32(preimage[len(idDomain):], uint32(len(wire)))
	copy(preimage[len(idDomain)+4:], wire)
	return sha256.Sum256(preimage), nil
}

// CometHash is CometBFT v0.38.26's raw-byte SHA-256 transaction hash.
// Unlike TxID, it is defined for arbitrary bytes.
func CometHash(raw []byte) [32]byte { return sha256.Sum256(raw) }

func parseKey(key [33]byte) (*secp256k1.PublicKey, error) {
	if key[0] != 2 && key[0] != 3 {
		return nil, ErrInvalidKey
	}
	parsed, err := secp256k1.ParsePubKey(key[:])
	if err != nil {
		return nil, ErrInvalidKey
	}
	return parsed, nil
}

func canonicalScalars(tx Transaction) bool {
	var r, s secp256k1.ModNScalar
	return !r.SetByteSlice(tx.R[:]) && !r.IsZero() &&
		!s.SetByteSlice(tx.S[:]) && !s.IsZero() && !s.IsOverHalfOrder()
}

// Validate applies the accepted stateless first-failure order. A CodeOK
// result does not establish nonce, balance, or block-level validity.
func Validate(wire []byte, expectedChainID string) Code {
	tx, err := Parse(wire)
	if err != nil {
		return CodeMalformed
	}
	if tx.ChainID != expectedChainID {
		return CodeWrongChain
	}
	if tx.Amount == 0 {
		return CodeZeroAmount
	}
	if !canonicalScalars(tx) {
		return CodeBadSig
	}
	pub, err := parseKey(tx.Sender)
	if err != nil {
		return CodeBadKey
	}
	digest, _ := SigningDigest(tx) // Parsed chain ID makes Encode infallible.
	var r, s secp256k1.ModNScalar
	r.SetByteSlice(tx.R[:])
	s.SetByteSlice(tx.S[:])
	if !ecdsa.NewSignature(&r, &s).Verify(digest[:], pub) {
		return CodeBadSig
	}
	return CodeOK
}
