// Package researchv2 defines the isolated four-profile research transaction
// and state machine. It does not alter accepted B0 or B1 encodings.
package researchv2

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
	TxMagic              = "PLUTOTX4"
	TxVersion     uint16 = 4
	MaxChainID           = 50
	coreFixed            = 12 + 20 + 20 + 8 + 8
	SecpProofSize        = 64
	MLProofSize          = mldsa44.SignatureSize
	intentDomain         = "PLUTO/RV2/INTENT/V1\x00"
	idDomain             = "PLUTO/RV2/TXID/V1\x00"
	MLContext            = "PLUTO/RV2/MLDSA/V1"
)

type Profile byte

const (
	ProfileC  Profile = 1
	ProfileCP Profile = 2
	ProfileP  Profile = 3
	ProfileH  Profile = 4
)

func (p Profile) Valid() bool    { return p >= ProfileC && p <= ProfileH }
func (p Profile) UsesSecp() bool { return p == ProfileC || p == ProfileCP || p == ProfileH }
func (p Profile) UsesML() bool   { return p == ProfileP || p == ProfileH }
func (p Profile) WireSize(chainID string) (int, error) {
	if !p.Valid() || !ValidChainID(chainID) {
		return 0, ErrMalformed
	}
	return coreFixed + len(chainID) + p.proofSize(), nil
}
func (p Profile) proofSize() int {
	switch p {
	case ProfileC:
		return SecpProofSize
	case ProfileCP, ProfileH:
		return SecpProofSize + MLProofSize
	case ProfileP:
		return MLProofSize
	default:
		return 0
	}
}

type Transaction struct {
	Profile       Profile
	ChainID       string
	Sender        [20]byte
	Receiver      [20]byte
	Amount        uint64
	Nonce         uint64
	SecpSignature [SecpProofSize]byte
	MLSignature   [MLProofSize]byte
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
	CodeWrongProfile        Code = 14
)

type Stage string

const (
	StageStructural Stage = "structural"
	StageProfile    Stage = "profile"
	StageChain      Stage = "chain"
	StageAmount     Stage = "amount"
	StageBinding    Stage = "binding"
	StageDigest     Stage = "digest"
	StageSecp       Stage = "secp"
	StageML         Stage = "ml"
	StageState      Stage = "state"
	StageStateCopy  Stage = "state_copy"
	StageAccepted   Stage = "accepted"
)

type Result struct {
	Code      Code
	LastStage Stage
}

var ErrMalformed = errors.New("malformed research-v2 transaction")

func ValidChainID(s string) bool {
	if len(s) < 1 || len(s) > MaxChainID {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-' || b == '.') {
			return false
		}
	}
	return true
}

func Parse(raw []byte) (Transaction, error) {
	if len(raw) < coreFixed+1+SecpProofSize || len(raw) > coreFixed+MaxChainID+SecpProofSize+MLProofSize || string(raw[:8]) != TxMagic || binary.BigEndian.Uint16(raw[8:10]) != TxVersion {
		return Transaction{}, ErrMalformed
	}
	p := Profile(raw[10])
	n := int(raw[11])
	if !p.Valid() || n < 1 || n > MaxChainID || len(raw) != coreFixed+n+p.proofSize() || !ValidChainID(string(raw[12:12+n])) {
		return Transaction{}, ErrMalformed
	}
	t := Transaction{Profile: p, ChainID: string(raw[12 : 12+n])}
	o := 12 + n
	copy(t.Sender[:], raw[o:o+20])
	o += 20
	copy(t.Receiver[:], raw[o:o+20])
	o += 20
	t.Amount = binary.BigEndian.Uint64(raw[o : o+8])
	o += 8
	t.Nonce = binary.BigEndian.Uint64(raw[o : o+8])
	o += 8
	if p.UsesSecp() {
		copy(t.SecpSignature[:], raw[o:o+SecpProofSize])
		o += SecpProofSize
	}
	if p == ProfileCP {
		for _, b := range raw[o:] {
			if b != 0 {
				return Transaction{}, ErrMalformed
			}
		}
	}
	if p.UsesML() {
		copy(t.MLSignature[:], raw[o:o+MLProofSize])
	}
	return t, nil
}

func Encode(t Transaction) ([]byte, error) {
	n, err := t.Profile.WireSize(t.ChainID)
	if err != nil {
		return nil, err
	}
	if !t.Profile.UsesSecp() && !allZero(t.SecpSignature[:]) {
		return nil, ErrMalformed
	}
	if !t.Profile.UsesML() && !allZero(t.MLSignature[:]) {
		return nil, ErrMalformed
	}
	raw := make([]byte, n)
	copy(raw[:8], TxMagic)
	binary.BigEndian.PutUint16(raw[8:10], TxVersion)
	raw[10], raw[11] = byte(t.Profile), byte(len(t.ChainID))
	copy(raw[12:], t.ChainID)
	o := 12 + len(t.ChainID)
	copy(raw[o:o+20], t.Sender[:])
	o += 20
	copy(raw[o:o+20], t.Receiver[:])
	o += 20
	binary.BigEndian.PutUint64(raw[o:o+8], t.Amount)
	o += 8
	binary.BigEndian.PutUint64(raw[o:o+8], t.Nonce)
	o += 8
	if t.Profile.UsesSecp() {
		copy(raw[o:o+SecpProofSize], t.SecpSignature[:])
		o += SecpProofSize
	}
	// ProfileCP's allocated tail is the only canonical zero-padding encoding.
	if t.Profile.UsesML() {
		copy(raw[o:o+MLProofSize], t.MLSignature[:])
	}
	return raw, nil
}

func allZero(raw []byte) bool {
	for _, b := range raw {
		if b != 0 {
			return false
		}
	}
	return true
}

// SigningDigest covers the complete logical transfer and profile domain, but
// neither CP padding nor any signature proof. Both H proofs sign this digest.
func SigningDigest(t Transaction) ([32]byte, error) {
	if !t.Profile.Valid() || !ValidChainID(t.ChainID) {
		return [32]byte{}, ErrMalformed
	}
	raw := make([]byte, len(intentDomain)+coreFixed+len(t.ChainID))
	copy(raw, intentDomain)
	o := len(intentDomain)
	copy(raw[o:o+8], TxMagic)
	o += 8
	binary.BigEndian.PutUint16(raw[o:o+2], TxVersion)
	o += 2
	raw[o], raw[o+1] = byte(t.Profile), byte(len(t.ChainID))
	o += 2
	copy(raw[o:o+len(t.ChainID)], t.ChainID)
	o += len(t.ChainID)
	copy(raw[o:o+20], t.Sender[:])
	o += 20
	copy(raw[o:o+20], t.Receiver[:])
	o += 20
	binary.BigEndian.PutUint64(raw[o:o+8], t.Amount)
	o += 8
	binary.BigEndian.PutUint64(raw[o:o+8], t.Nonce)
	return sha256.Sum256(raw), nil
}

func TxID(raw []byte) ([32]byte, error) {
	if _, err := Parse(raw); err != nil {
		return [32]byte{}, err
	}
	buf := make([]byte, len(idDomain)+4+len(raw))
	copy(buf, idDomain)
	binary.BigEndian.PutUint32(buf[len(idDomain):], uint32(len(raw)))
	copy(buf[len(idDomain)+4:], raw)
	return sha256.Sum256(buf), nil
}

func parseSecpKey(raw [33]byte) (*secp256k1.PublicKey, error) {
	if raw[0] != 2 && raw[0] != 3 {
		return nil, errors.New("invalid secp key")
	}
	return secp256k1.ParsePubKey(raw[:])
}
func parseMLKey(raw [mldsa44.PublicKeySize]byte) (*mldsa44.PublicKey, error) {
	var pub mldsa44.PublicKey
	if err := pub.UnmarshalBinary(raw[:]); err != nil || !bytes.Equal(pub.Bytes(), raw[:]) {
		return nil, errors.New("invalid ML key")
	}
	return &pub, nil
}
func verifySecp(key [33]byte, sig [SecpProofSize]byte, digest [32]byte) Code {
	var r, s secp256k1.ModNScalar
	if r.SetByteSlice(sig[:32]) || r.IsZero() || s.SetByteSlice(sig[32:]) || s.IsZero() || s.IsOverHalfOrder() {
		return CodeBadSecpSignature
	}
	pub, err := parseSecpKey(key)
	if err != nil {
		return CodeBadSecpKey
	}
	if !ecdsa.NewSignature(&r, &s).Verify(digest[:], pub) {
		return CodeBadSecpSignature
	}
	return CodeOK
}
func verifyML(key [mldsa44.PublicKeySize]byte, sig [MLProofSize]byte, digest [32]byte) Code {
	pub, err := parseMLKey(key)
	if err != nil {
		return CodeBadMLKey
	}
	if !mldsa44.Verify(pub, digest[:], []byte(MLContext), sig[:]) {
		return CodeBadMLSignature
	}
	return CodeOK
}
