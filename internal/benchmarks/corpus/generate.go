package corpus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"sort"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/b1state"
	"example.com/pluto/internal/b1test"
	"example.com/pluto/internal/b1tx"
	"example.com/pluto/internal/tx"
)

func changed(src []byte, offset int, value byte) []byte {
	out := bytes.Clone(src)
	out[offset] = value
	return out
}

func appendCommonStructural(p *Profile, valid []byte, hybrid bool) {
	malformed := uint32(1)
	addCase(p, "S01", "empty transaction", "structural", "A", "initial", "empty byte string", nil, malformed, "S1", "none")
	addCase(p, "S02", "truncated transaction", "structural", "A", "initial", "remove final byte", valid[:len(valid)-1], malformed, "S1", "none")
	addZeroRepeatCase(p, "S03", "oversized transaction", "structural", "A", (1<<20)+1, malformed, "S1")
	addCase(p, "S04", "incorrect magic", "structural", "A", "initial", "replace first magic byte", changed(valid, 0, 'X'), malformed, "S1", "none")
	addCase(p, "S05", "incorrect version", "structural", "A", "initial", "replace version byte", changed(valid, 9, 0xff), malformed, "S1", "none")
	chainOffset := 12
	if hybrid {
		addCase(p, "S06", "incorrect authentication profile", "structural", "A", "initial", "replace profile byte", changed(valid, 10, 0), malformed, "S1", "none")
		addCase(p, "S07", "incorrect chain length", "structural", "A", "initial", "replace u8 length", changed(valid, 11, 0), malformed, "S1", "none")
	} else {
		addCase(p, "S07", "incorrect chain length", "structural", "A", "initial", "replace u16 length", changed(valid, 11, 0xff), malformed, "S1", "none")
	}
	addCase(p, "S08", "invalid chain character", "structural", "A", "initial", "replace chain byte", changed(valid, chainOffset, '/'), malformed, "S1", "none")
	addCase(p, "S09", "trailing data", "structural", "A", "initial", "append one byte", append(bytes.Clone(valid), 0), malformed, "S1", "none")
	addCase(p, "C01", "wrong chain context", "chain", "B", "initial", "replace one chain byte", changed(valid, chainOffset, 'x'), 2, "S2", "none")
}

func signedB0(key *secp256k1.PrivateKey, receiver b0state.Address, chain string, amount, nonce uint64) ([]byte, error) {
	var t tx.Transaction
	t.ChainID = chain
	copy(t.Sender[:], key.PubKey().SerializeCompressed())
	t.Receiver = [20]byte(receiver)
	t.Amount, t.Nonce = amount, nonce
	d, err := tx.SigningDigest(t)
	if err != nil {
		return nil, err
	}
	s := ecdsa.Sign(key, d[:])
	r, ss := s.R(), s.S()
	r.PutBytes(&t.R)
	ss.PutBytes(&t.S)
	return tx.Encode(t)
}

func generateB0() (Profile, error) {
	return generateB0WithLayers(1)
}

func generateB0WithLayers(layers int) (Profile, error) {
	const chain = "phase05-b0"
	p := Profile{Name: "b0", ChainID: chain, AccountCount: SenderCount + 1}
	keys := make([]*secp256k1.PrivateKey, SenderCount)
	alloc := make([]b0state.Allocation, 0, SenderCount+1)
	for i := range keys {
		key, err := secp256k1.GeneratePrivateKey()
		if err != nil {
			return Profile{}, err
		}
		keys[i] = key
		address, err := tx.SenderAddress(*(*[33]byte)(key.PubKey().SerializeCompressed()))
		if err != nil {
			return Profile{}, err
		}
		alloc = append(alloc, b0state.Allocation{Address: b0state.Address(address), Balance: 100})
	}
	var receiver b0state.Address
	copy(receiver[:], []byte("phase05-receiver-b0"))
	alloc = append(alloc, b0state.Allocation{Address: receiver, Balance: 1})
	sort.Slice(alloc, func(i, j int) bool { return bytes.Compare(alloc[i].Address[:], alloc[j].Address[:]) < 0 })
	genesis, err := b0state.EncodeGenesis(chain, alloc)
	if err != nil {
		return Profile{}, err
	}
	p.GenesisHex, p.GenesisSHA256 = hex.EncodeToString(genesis), digest(genesis)
	for i, key := range keys {
		raw, err := signedB0(key, receiver, chain, 1, 0)
		if err != nil {
			return Profile{}, err
		}
		addr, _ := tx.SenderAddress(*(*[33]byte)(key.PubKey().SerializeCompressed()))
		p.Valid = append(p.Valid, vector(fmt.Sprintf("V%03d", i), raw, b0state.Address(addr), receiver))
	}
	for nonce := 1; nonce < layers; nonce++ {
		for i, key := range keys {
			raw, err := signedB0(key, receiver, chain, 1, uint64(nonce))
			if err != nil {
				return Profile{}, err
			}
			addr, _ := tx.SenderAddress(*(*[33]byte)(key.PubKey().SerializeCompressed()))
			p.Valid = append(p.Valid, vectorAtNonce(fmt.Sprintf("V%03d", nonce*SenderCount+i), raw, b0state.Address(addr), receiver, uint64(nonce)))
		}
	}
	valid, _ := p.Valid[0].Bytes()
	other, _ := p.Valid[1].Bytes()
	appendCommonStructural(&p, valid, false)
	n := len(chain)
	keyStart, amountStart := 12+n, 65+n
	addCase(&p, "I01", "substituted sender key", "identity", "B", "initial", "replace compressed sender key", func() []byte {
		r := bytes.Clone(valid)
		copy(r[keyStart:keyStart+33], other[keyStart:keyStart+33])
		return r
	}(), 4, "S4", "none")
	addCase(&p, "I02", "invalid sender key", "identity", "A", "initial", "zero compressed key prefix", changed(valid, keyStart, 0), 3, "S3", "none")
	foreignKey, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return Profile{}, err
	}
	addCase(&p, "I03", "unfunded new sender", "identity", "A", "initial", "sign with fresh unregistered B0 key", mustB0(foreignKey, receiver, chain, 1, 0), 7, "S6", "none")
	addCase(&p, "P01", "zeroed classical signature", "classical_proof", "B", "initial", "zero 64-byte proof", append(bytes.Clone(valid[:len(valid)-64]), make([]byte, 64)...), 4, "S4", "none")
	addCase(&p, "P02", "invalid classical signature", "classical_proof", "B", "initial", "flip first R byte", changed(valid, 81+n, valid[81+n]^1), 4, "S4", "none")
	addCase(&p, "P03", "classical proof from another intent", "classical_proof", "B", "initial", "copy proof from other sender", func() []byte { r := bytes.Clone(valid); copy(r[len(r)-64:], other[len(other)-64:]); return r }(), 4, "S4", "none")
	addCase(&p, "P04", "zero R scalar", "classical_proof", "A", "initial", "zero R", func() []byte { r := bytes.Clone(valid); clear(r[81+n : 113+n]); return r }(), 4, "S4", "none")
	addCase(&p, "P05", "out-of-range R scalar", "classical_proof", "A", "initial", "set R to 0xff", func() []byte {
		r := bytes.Clone(valid)
		for i := 81 + n; i < 113+n; i++ {
			r[i] = 0xff
		}
		return r
	}(), 4, "S4", "none")
	addCase(&p, "A01", "zero amount", "state", "B", "initial", "zero amount bytes", func() []byte { r := bytes.Clone(valid); clear(r[amountStart : amountStart+8]); return r }(), 5, "S2", "none")
	addCase(&p, "A02", "incorrect nonce", "state", "C", "initial", "sign nonce 1", mustB0(keys[0], receiver, chain, 1, 1), 6, "S6", "none")
	addCase(&p, "A03", "insufficient balance", "state", "C", "initial", "sign amount 101", mustB0(keys[0], receiver, chain, 101, 0), 7, "S6", "none")
	addCase(&p, "A04", "maximum amount", "state", "C", "initial", "sign max uint64 amount", mustB0(keys[0], receiver, chain, math.MaxUint64, 0), 7, "S6", "none")
	addCase(&p, "A05", "maximum nonce", "state", "C", "initial", "sign max uint64 nonce", mustB0(keys[0], receiver, chain, 1, math.MaxUint64), 6, "S6", "none")
	addCase(&p, "A06", "replayed valid transfer", "state", "B", "after_V000", "reuse public signed V000", valid, 6, "S6", "none")
	addCase(&p, "A07", "same nonce conflicting intent", "state", "C", "after_V000", "sign another amount at nonce 0", mustB0(keys[0], receiver, chain, 2, 0), 6, "S6", "none")
	addCase(&p, "A08", "valid successor nonce", "control", "C", "after_V000", "sign nonce 1", mustB0(keys[0], receiver, chain, 1, 1), 0, "S6", "transfer")
	addProposal(&p, "Q01", "invalid bypassing CheckTx", "direct invalid proposal", "P02")
	addProposal(&p, "Q02", "valid then invalid", "ordered mixed proposal", "V000", "P02")
	addProposal(&p, "Q03", "same nonce conflict", "ordered duplicate", "V000", "A07")
	addProposal(&p, "Q04", "oversized envelope", "oversized byte string", "S03")
	addProposal(&p, "Q05", "mixed invalid classes", "mixed ordered proposal", "S01", "V000", "P02")
	return p, nil
}

func mustB0(key *secp256k1.PrivateKey, receiver b0state.Address, chain string, amount, nonce uint64) []byte {
	raw, err := signedB0(key, receiver, chain, amount, nonce)
	if err != nil {
		panic(err)
	}
	return raw
}

func mustB1(f *b1test.Fixture, sender b1test.Identity, receiver b0state.Address, amount, nonce uint64) []byte {
	raw, err := f.SignedBy(sender, receiver, amount, nonce)
	if err != nil {
		panic(err)
	}
	return raw
}

func generateB1() (Profile, error) {
	return generateB1WithLayers(1)
}

func generateB1WithLayers(layers int) (Profile, error) {
	const chain = "phase05-b1"
	p := Profile{Name: "b1", ChainID: chain, AccountCount: SenderCount + 1}
	ids := make([]b1test.Identity, SenderCount+1)
	alloc := make([]b1state.Allocation, 0, len(ids))
	for i := range ids {
		id, err := b1test.NewIdentity()
		if err != nil {
			return Profile{}, err
		}
		ids[i] = id
		balance := uint64(100)
		if i == SenderCount {
			balance = 0
		}
		alloc = append(alloc, b1state.Allocation{Binding: id.Binding, Balance: balance})
	}
	sort.Slice(alloc, func(i, j int) bool {
		return bytes.Compare(alloc[i].Binding.Address[:], alloc[j].Binding.Address[:]) < 0
	})
	genesis, err := b1state.EncodeGenesis(chain, alloc)
	if err != nil {
		return Profile{}, err
	}
	p.GenesisHex, p.GenesisSHA256 = hex.EncodeToString(genesis), digest(genesis)
	f := &b1test.Fixture{ChainID: chain, Genesis: genesis}
	receiver := ids[SenderCount].Binding.Address
	for i := 0; i < SenderCount; i++ {
		raw, err := f.SignedBy(ids[i], receiver, 1, 0)
		if err != nil {
			return Profile{}, err
		}
		p.Valid = append(p.Valid, vector(fmt.Sprintf("V%03d", i), raw, ids[i].Binding.Address, receiver))
	}
	for nonce := 1; nonce < layers; nonce++ {
		for i := 0; i < SenderCount; i++ {
			raw, err := f.SignedBy(ids[i], receiver, 1, uint64(nonce))
			if err != nil {
				return Profile{}, err
			}
			p.Valid = append(p.Valid, vectorAtNonce(fmt.Sprintf("V%03d", nonce*SenderCount+i), raw, ids[i].Binding.Address, receiver, uint64(nonce)))
		}
	}
	valid, _ := p.Valid[0].Bytes()
	other, _ := p.Valid[1].Bytes()
	appendCommonStructural(&p, valid, true)
	n := len(chain)
	senderStart := 12 + n
	secpStart := senderStart + 20
	mlStart := secpStart + 33
	receiverStart := mlStart + len(ids[0].Binding.MLKey)
	secpSigStart := len(valid) - 64 - mldsa44.SignatureSize
	mlSigStart := len(valid) - mldsa44.SignatureSize
	addCase(&p, "I01", "unregistered sender", "identity", "A", "initial", "replace sender address", changed(valid, senderStart, valid[senderStart]^1), 11, "S3", "none")
	addCase(&p, "I02", "substituted secp public key", "identity", "B", "initial", "copy other registered secp key", func() []byte {
		r := bytes.Clone(valid)
		copy(r[secpStart:secpStart+33], other[secpStart:secpStart+33])
		return r
	}(), 11, "S3", "none")
	addCase(&p, "I03", "substituted ML public key", "identity", "B", "initial", "copy other registered ML key", func() []byte {
		r := bytes.Clone(valid)
		copy(r[mlStart:mlStart+len(ids[0].Binding.MLKey)], other[mlStart:mlStart+len(ids[0].Binding.MLKey)])
		return r
	}(), 11, "S3", "none")
	addCase(&p, "I04", "invalid secp public key", "identity", "A", "initial", "zero secp prefix", changed(valid, secpStart, 0), 3, "S3", "none")
	addCase(&p, "I05", "unregistered zero ML public key", "identity", "A", "initial", "zero ML key bytes; fixed-size unpacking accepts the encoding, registry binding rejects it", func() []byte {
		r := bytes.Clone(valid)
		clear(r[mlStart : mlStart+len(ids[0].Binding.MLKey)])
		return r
	}(), 11, "S3", "none")
	addCase(&p, "I06", "unregistered recipient", "identity", "C", "initial", "replace recipient address", changed(valid, receiverStart, valid[receiverStart]^1), 11, "S3", "none")
	addCase(&p, "P01", "zeroed classical signature", "classical_proof", "B", "initial", "zero secp proof", func() []byte { r := bytes.Clone(valid); clear(r[secpSigStart:mlSigStart]); return r }(), 4, "S4", "none")
	addCase(&p, "P02", "zeroed ML signature", "ml_proof", "B", "initial", "zero ML proof", func() []byte { r := bytes.Clone(valid); clear(r[mlSigStart:]); return r }(), 13, "S5", "none")
	addCase(&p, "P03", "invalid classical signature", "classical_proof", "B", "initial", "flip R byte", changed(valid, secpSigStart, valid[secpSigStart]^1), 4, "S4", "none")
	addCase(&p, "P04", "invalid ML signature", "ml_proof", "B", "initial", "flip final ML proof byte", changed(valid, len(valid)-1, valid[len(valid)-1]^1), 13, "S5", "none")
	addCase(&p, "P05", "both proofs invalid", "classical_proof", "B", "initial", "zero both signatures", func() []byte { r := bytes.Clone(valid); clear(r[secpSigStart:]); return r }(), 4, "S4", "none")
	addCase(&p, "P06", "classical proof from other intent", "classical_proof", "B", "initial", "copy other secp proof", func() []byte {
		r := bytes.Clone(valid)
		copy(r[secpSigStart:mlSigStart], other[secpSigStart:mlSigStart])
		return r
	}(), 4, "S4", "none")
	addCase(&p, "P07", "ML proof from other intent", "ml_proof", "B", "initial", "copy other ML proof", func() []byte { r := bytes.Clone(valid); copy(r[mlSigStart:], other[mlSigStart:]); return r }(), 13, "S5", "none")
	addCase(&p, "P08", "out-of-range secp R", "classical_proof", "A", "initial", "fill secp R with 0xff", func() []byte {
		r := bytes.Clone(valid)
		for i := secpSigStart; i < secpSigStart+32; i++ {
			r[i] = 0xff
		}
		return r
	}(), 4, "S4", "none")
	addCase(&p, "P09", "noncanonical ML signature bytes", "ml_proof", "B", "initial", "fill ML proof with 0xff", func() []byte {
		r := bytes.Clone(valid)
		for i := mlSigStart; i < len(r); i++ {
			r[i] = 0xff
		}
		return r
	}(), 13, "S5", "none")
	addCase(&p, "A01", "zero amount", "state", "B", "initial", "zero amount bytes", func() []byte { r := bytes.Clone(valid); clear(r[receiverStart+20 : receiverStart+28]); return r }(), 5, "S2", "none")
	addCase(&p, "A02", "incorrect nonce", "state", "C", "initial", "sign nonce 1", mustB1(f, ids[0], receiver, 1, 1), 6, "S6", "none")
	addCase(&p, "A03", "insufficient balance", "state", "C", "initial", "sign amount 101", mustB1(f, ids[0], receiver, 101, 0), 7, "S6", "none")
	addCase(&p, "A04", "maximum amount", "state", "C", "initial", "sign max amount", mustB1(f, ids[0], receiver, math.MaxUint64, 0), 7, "S6", "none")
	addCase(&p, "A05", "maximum nonce", "state", "C", "initial", "sign max nonce", mustB1(f, ids[0], receiver, 1, math.MaxUint64), 6, "S6", "none")
	addCase(&p, "A06", "replayed signed transaction", "state", "B", "after_V000", "reuse public signed V000", valid, 6, "S6", "none")
	addCase(&p, "A07", "same nonce conflicting intent", "state", "C", "after_V000", "sign different amount at nonce 0", mustB1(f, ids[0], receiver, 2, 0), 6, "S6", "none")
	addCase(&p, "A08", "valid successor nonce", "control", "C", "after_V000", "sign nonce 1", mustB1(f, ids[0], receiver, 1, 1), 0, "S6", "transfer")
	duplicate, err := b1tx.Parse(valid)
	if err != nil {
		return Profile{}, err
	}
	d, err := b1tx.SigningDigest(duplicate)
	if err != nil {
		return Profile{}, err
	}
	if err := mldsa44.SignTo(ids[0].ML, d[:], []byte(b1tx.MLContext), true, duplicate.MLSignature[:]); err != nil {
		return Profile{}, err
	}
	if bytes.Equal(duplicate.MLSignature[:], valid[len(valid)-mldsa44.SignatureSize:]) {
		return Profile{}, fmt.Errorf("randomized ML-DSA signature did not differ")
	}
	randomized, err := b1tx.Encode(duplicate)
	if err != nil {
		return Profile{}, err
	}
	addCase(&p, "A09", "same signed intent with distinct valid ML proof", "state", "B", "after_V000", "randomized ML-DSA signature over identical digest; public bytes frozen in corpus", randomized, 6, "S6", "none")
	addProposal(&p, "Q01", "invalid bypassing CheckTx", "direct invalid proposal", "P04")
	addProposal(&p, "Q02", "valid then invalid", "ordered mixed proposal", "V000", "P04")
	addProposal(&p, "Q03", "same nonce conflict", "ordered duplicate", "V000", "A07")
	addProposal(&p, "Q04", "oversized envelope", "oversized byte string", "S03")
	addProposal(&p, "Q05", "mixed invalid classes", "mixed ordered proposal", "S01", "V000", "P04")
	badGenesis := bytes.Clone(genesis)
	binary.BigEndian.PutUint32(badGenesis[8:12], 0)
	p.GenesisCases = append(p.GenesisCases, GenesisCase{ID: "G01", Name: "incorrect genesis version", GenesisHex: hex.EncodeToString(badGenesis), SHA256: digest(badGenesis), Expected: "REJECT", Construction: "replace genesis version"})
	badBinding := bytes.Clone(genesis)
	firstKeyOffset := 18 + len(chain) + 20 + 8
	badBinding[firstKeyOffset] ^= 1
	p.GenesisCases = append(p.GenesisCases, GenesisCase{ID: "G02", Name: "incorrect genesis key binding", GenesisHex: hex.EncodeToString(badBinding), SHA256: digest(badBinding), Expected: "REJECT", Construction: "flip first genesis secp key byte without changing registered address"})
	return p, nil
}
