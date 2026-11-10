// Package researchv2test creates ephemeral signers for the four-profile
// tests. No private key, signing seed or runtime home is committed.
package researchv2test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"sort"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/profiles"
)

type Identity struct {
	Secp    *secp256k1.PrivateKey
	ML      *mldsa44.PrivateKey
	Binding researchv2.Binding
}
type Fixture struct {
	ChainID  string
	Genesis  []byte
	Sender   Identity
	Receiver Identity
}

func NewIdentity() (Identity, error) {
	secp, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return Identity{}, err
	}
	pub, ml, err := mldsa44.GenerateKey(rand.Reader)
	if err != nil {
		return Identity{}, err
	}
	var binding researchv2.Binding
	copy(binding.SecpKey[:], secp.PubKey().SerializeCompressed())
	copy(binding.MLKey[:], pub.Bytes())
	buf := make([]byte, len("PLUTO/RV2/ACCOUNT/V1\x00")+len(binding.SecpKey)+len(binding.MLKey))
	copy(buf, "PLUTO/RV2/ACCOUNT/V1\x00")
	copy(buf[len("PLUTO/RV2/ACCOUNT/V1\x00"):], binding.SecpKey[:])
	copy(buf[len("PLUTO/RV2/ACCOUNT/V1\x00")+len(binding.SecpKey):], binding.MLKey[:])
	h := sha256.Sum256(buf)
	copy(binding.Account[:], h[:20])
	return Identity{Secp: secp, ML: ml, Binding: binding}, nil
}
func New(chainID string) (*Fixture, error) {
	sender, err := NewIdentity()
	if err != nil {
		return nil, err
	}
	receiver, err := NewIdentity()
	if err != nil {
		return nil, err
	}
	allocations := []researchv2.Allocation{{Binding: sender.Binding, Balance: 100}, {Binding: receiver.Binding, Balance: 0}}
	sort.Slice(allocations, func(i, j int) bool {
		return bytes.Compare(allocations[i].Binding.Account[:], allocations[j].Binding.Account[:]) < 0
	})
	genesis, err := researchv2.EncodeGenesis(chainID, allocations)
	if err != nil {
		return nil, err
	}
	return &Fixture{ChainID: chainID, Genesis: genesis, Sender: sender, Receiver: receiver}, nil
}
func (f *Fixture) Signed(profile researchv2.Profile, amount, nonce uint64) ([]byte, error) {
	return f.SignedBy(profile, f.Sender, f.Sender.Binding.Account, f.Receiver.Binding.Account, amount, nonce)
}
func (f *Fixture) SignedBy(profile researchv2.Profile, signer Identity, sender, receiver b0state.Address, amount, nonce uint64) ([]byte, error) {
	if !profile.Valid() {
		return nil, researchv2.ErrMalformed
	}
	if profile.UsesSecp() && signer.Secp == nil {
		return nil, errors.New("missing secp signer")
	}
	if profile.UsesML() && signer.ML == nil {
		return nil, errors.New("missing ML signer")
	}
	t := researchv2.Transaction{Profile: profile, ChainID: f.ChainID, Sender: [20]byte(sender), Receiver: [20]byte(receiver), Amount: amount, Nonce: nonce}
	digest, err := researchv2.SigningDigest(t)
	if err != nil {
		return nil, err
	}
	if profile.UsesSecp() {
		sig := ecdsa.Sign(signer.Secp, digest[:])
		r, s := sig.R(), sig.S()
		var rb, sb [32]byte
		r.PutBytes(&rb)
		s.PutBytes(&sb)
		copy(t.SecpSignature[:32], rb[:])
		copy(t.SecpSignature[32:], sb[:])
	}
	if profile.UsesML() {
		if err := mldsa44.SignTo(signer.ML, digest[:], []byte(researchv2.MLContext), false, t.MLSignature[:]); err != nil {
			return nil, err
		}
	}
	return researchv2.Encode(t)
}
