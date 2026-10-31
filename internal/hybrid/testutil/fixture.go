// Package b1test creates ephemeral hybrid test identities. It contains no
// committed transaction private key, seed, validator identity, or runtime home.
package testutil

import (
	"bytes"
	"crypto/rand"
	"errors"
	"sort"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	b0state "example.com/pluto/internal/classical/state"
	b1state "example.com/pluto/internal/hybrid/state"
	b1tx "example.com/pluto/internal/hybrid/tx"
)

type Identity struct {
	Secp    *secp256k1.PrivateKey
	ML      *mldsa44.PrivateKey
	Binding b1state.Binding
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
	mlpub, ml, err := mldsa44.GenerateKey(rand.Reader)
	if err != nil {
		return Identity{}, err
	}
	var binding b1state.Binding
	copy(binding.SecpKey[:], secp.PubKey().SerializeCompressed())
	copy(binding.MLKey[:], mlpub.Bytes())
	address, err := b1tx.SenderAddress(binding.SecpKey, binding.MLKey)
	if err != nil {
		return Identity{}, err
	}
	binding.Address = b0state.Address(address)
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
	allocations := []b1state.Allocation{{Binding: sender.Binding, Balance: 100}, {Binding: receiver.Binding, Balance: 0}}
	sort.Slice(allocations, func(i, j int) bool {
		return bytes.Compare(allocations[i].Binding.Address[:], allocations[j].Binding.Address[:]) < 0
	})
	genesis, err := b1state.EncodeGenesis(chainID, allocations)
	if err != nil {
		return nil, err
	}
	return &Fixture{ChainID: chainID, Genesis: genesis, Sender: sender, Receiver: receiver}, nil
}

func (f *Fixture) Signed(amount, nonce uint64) ([]byte, error) {
	return f.SignedBy(f.Sender, f.Receiver.Binding.Address, amount, nonce)
}
func (f *Fixture) SignedBy(sender Identity, receiver b0state.Address, amount, nonce uint64) ([]byte, error) {
	if sender.Secp == nil || sender.ML == nil {
		return nil, errors.New("test signer is incomplete")
	}
	var t b1tx.Transaction
	t.ChainID = f.ChainID
	t.Sender = [20]byte(sender.Binding.Address)
	t.SecpKey = sender.Binding.SecpKey
	t.MLKey = sender.Binding.MLKey
	t.Receiver = [20]byte(receiver)
	t.Amount, t.Nonce = amount, nonce
	digest, err := b1tx.SigningDigest(t)
	if err != nil {
		return nil, err
	}
	sig := ecdsa.Sign(sender.Secp, digest[:])
	r, s := sig.R(), sig.S()
	var rb, sb [32]byte
	r.PutBytes(&rb)
	s.PutBytes(&sb)
	copy(t.SecpSignature[:32], rb[:])
	copy(t.SecpSignature[32:], sb[:])
	if err := mldsa44.SignTo(sender.ML, digest[:], []byte(b1tx.MLContext), false, t.MLSignature[:]); err != nil {
		return nil, err
	}
	return b1tx.Encode(t)
}
