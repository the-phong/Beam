// Package b1tx defines the isolated B1 hybrid transfer envelope. It never
// interprets or changes an accepted B0 transaction.
package tx

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

const (
	Magic                = "PLUTOTX2"
	Version       uint16 = 2
	ProfileHybrid byte   = 1
	MaxChainID           = 50
	unsignedBase         = 12 + 20 + 33 + mldsa44.PublicKeySize + 20 + 8 + 8
	MinWireSize          = unsignedBase + 1 + 64 + mldsa44.SignatureSize
	MaxWireSize          = unsignedBase + MaxChainID + 64 + mldsa44.SignatureSize
	signDomain           = "PLUTO/B1/HYBRID/SIGN/V1\x00"
	addressDomain        = "PLUTO/B1/ADDR/V1\x00"
	idDomain             = "PLUTO/B1/TXID/V1\x00"
	// Pure ML-DSA signs the same 32-byte canonical digest as ECDSA. This
	// context is additionally included by FIPS 204's pure-ML-DSA framing.
	MLContext = "PLUTO/B1/HYBRID/V1"
)

type Transaction struct {
	ChainID       string
	Sender        [20]byte
	SecpKey       [33]byte
	MLKey         [mldsa44.PublicKeySize]byte
	Receiver      [20]byte
	Amount        uint64
	Nonce         uint64
	SecpSignature [64]byte
	MLSignature   [mldsa44.SignatureSize]byte
}

type Code uint32

const (
	CodeOK                  Code = 0
	CodeMalformed           Code = 1
	CodeWrongChain          Code = 2
	CodeBadSecpKey          Code = 3
	CodeBadSecpSignature    Code = 4
	CodeZeroAmount          Code = 5
	CodeInvalidNonce        Code = 6
	CodeInsufficientBalance Code = 7
	CodeBalanceOverflow     Code = 8
	CodeAccountLimit        Code = 9
	CodeBlockLimit          Code = 10
	CodeBadBinding          Code = 11
	CodeBadMLKey            Code = 12
	CodeBadMLSignature      Code = 13
)

var ErrMalformed = errors.New("malformed B1 hybrid transaction")
var ErrKey = errors.New("invalid B1 public key")

func ValidChainID(s string) bool {
	if len(s) < 1 || len(s) > MaxChainID {
		return false
	}
	for i := range s {
		b := s[i]
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '_' || b == '-' || b == '.') {
			return false
		}
	}
	return true
}

// Parse bounds total input before inspecting fields or allocating. Fixed
// widths and one exact length leave no optional, trailing, or nested fields.
func Parse(raw []byte) (Transaction, error) {
	if len(raw) < MinWireSize || len(raw) > MaxWireSize ||
		string(raw[:8]) != Magic || binary.BigEndian.Uint16(raw[8:10]) != Version ||
		raw[10] != ProfileHybrid {
		return Transaction{}, ErrMalformed
	}
	n := int(raw[11])
	if n < 1 || n > MaxChainID || len(raw) != unsignedBase+n+64+mldsa44.SignatureSize ||
		!ValidChainID(string(raw[12:12+n])) {
		return Transaction{}, ErrMalformed
	}
	t := Transaction{ChainID: string(raw[12 : 12+n])}
	p := 12 + n
	copy(t.Sender[:], raw[p:p+20])
	p += 20
	copy(t.SecpKey[:], raw[p:p+33])
	p += 33
	copy(t.MLKey[:], raw[p:p+mldsa44.PublicKeySize])
	p += mldsa44.PublicKeySize
	copy(t.Receiver[:], raw[p:p+20])
	p += 20
	t.Amount = binary.BigEndian.Uint64(raw[p : p+8])
	p += 8
	t.Nonce = binary.BigEndian.Uint64(raw[p : p+8])
	p += 8
	copy(t.SecpSignature[:], raw[p:p+64])
	p += 64
	copy(t.MLSignature[:], raw[p:p+mldsa44.SignatureSize])
	return t, nil
}

func Encode(t Transaction) ([]byte, error) {
	if !ValidChainID(t.ChainID) {
		return nil, ErrMalformed
	}
	n := len(t.ChainID)
	raw := make([]byte, unsignedBase+n+64+mldsa44.SignatureSize)
	copy(raw[:8], Magic)
	binary.BigEndian.PutUint16(raw[8:10], Version)
	raw[10], raw[11] = ProfileHybrid, byte(n)
	copy(raw[12:12+n], t.ChainID)
	p := 12 + n
	copy(raw[p:p+20], t.Sender[:])
	p += 20
	copy(raw[p:p+33], t.SecpKey[:])
	p += 33
	copy(raw[p:p+mldsa44.PublicKeySize], t.MLKey[:])
	p += mldsa44.PublicKeySize
	copy(raw[p:p+20], t.Receiver[:])
	p += 20
	binary.BigEndian.PutUint64(raw[p:p+8], t.Amount)
	p += 8
	binary.BigEndian.PutUint64(raw[p:p+8], t.Nonce)
	p += 8
	copy(raw[p:p+64], t.SecpSignature[:])
	p += 64
	copy(raw[p:p+mldsa44.SignatureSize], t.MLSignature[:])
	return raw, nil
}

func SigningDigest(t Transaction) ([32]byte, error) {
	raw, err := Encode(t)
	if err != nil {
		return [32]byte{}, err
	}
	u := raw[:len(raw)-64-mldsa44.SignatureSize]
	preimage := make([]byte, len(signDomain)+4+len(u))
	copy(preimage, signDomain)
	binary.BigEndian.PutUint32(preimage[len(signDomain):], uint32(len(u)))
	copy(preimage[len(signDomain)+4:], u)
	return sha256.Sum256(preimage), nil
}

func SenderAddress(secp [33]byte, ml [mldsa44.PublicKeySize]byte) ([20]byte, error) {
	if _, err := ParseSecpKey(secp); err != nil {
		return [20]byte{}, ErrKey
	}
	if _, err := ParseMLKey(ml); err != nil {
		return [20]byte{}, ErrKey
	}
	preimage := make([]byte, len(addressDomain)+len(secp)+len(ml))
	copy(preimage, addressDomain)
	copy(preimage[len(addressDomain):], secp[:])
	copy(preimage[len(addressDomain)+len(secp):], ml[:])
	hash := sha256.Sum256(preimage)
	var address [20]byte
	copy(address[:], hash[:20])
	return address, nil
}

func ParseSecpKey(key [33]byte) (*secp256k1.PublicKey, error) {
	if key[0] != 2 && key[0] != 3 {
		return nil, ErrKey
	}
	pub, err := secp256k1.ParsePubKey(key[:])
	if err != nil {
		return nil, ErrKey
	}
	return pub, nil
}

func ParseMLKey(key [mldsa44.PublicKeySize]byte) (*mldsa44.PublicKey, error) {
	var pub mldsa44.PublicKey
	if err := pub.UnmarshalBinary(key[:]); err != nil || !bytes.Equal(pub.Bytes(), key[:]) {
		return nil, ErrKey
	}
	return &pub, nil
}

func VerifySecp(t Transaction, digest [32]byte) Code {
	var r, s secp256k1.ModNScalar
	if r.SetByteSlice(t.SecpSignature[:32]) || r.IsZero() ||
		s.SetByteSlice(t.SecpSignature[32:]) || s.IsZero() || s.IsOverHalfOrder() {
		return CodeBadSecpSignature
	}
	pub, err := ParseSecpKey(t.SecpKey)
	if err != nil {
		return CodeBadSecpKey
	}
	if !ecdsa.NewSignature(&r, &s).Verify(digest[:], pub) {
		return CodeBadSecpSignature
	}
	return CodeOK
}

func VerifyML(t Transaction, digest [32]byte) Code {
	pub, err := ParseMLKey(t.MLKey)
	if err != nil {
		return CodeBadMLKey
	}
	if !mldsa44.Verify(pub, digest[:], []byte(MLContext), t.MLSignature[:]) {
		return CodeBadMLSignature
	}
	return CodeOK
}

func TxID(raw []byte) ([32]byte, error) {
	if _, err := Parse(raw); err != nil {
		return [32]byte{}, err
	}
	preimage := make([]byte, len(idDomain)+4+len(raw))
	copy(preimage, idDomain)
	binary.BigEndian.PutUint32(preimage[len(idDomain):], uint32(len(raw)))
	copy(preimage[len(idDomain)+4:], raw)
	return sha256.Sum256(preimage), nil
}
