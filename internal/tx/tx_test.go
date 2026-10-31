package tx

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"math/big"
	"os"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

const fixtureChain = "pluto-b0-v1"

func signedFixture(t *testing.T, amount, nonce uint64) []byte {
	t.Helper()
	var secret [32]byte
	secret[31] = 1 // TEST_ONLY_A, public and unsuitable for production.
	key := secp256k1.PrivKeyFromBytes(secret[:])
	var tx Transaction
	tx.ChainID = fixtureChain
	copy(tx.Sender[:], key.PubKey().SerializeCompressed())
	tx.Receiver[19] = 2
	tx.Amount, tx.Nonce = amount, nonce
	digest, err := SigningDigest(tx)
	if err != nil {
		t.Fatal(err)
	}
	sig := ecdsa.Sign(key, digest[:])
	r, s := sig.R(), sig.S()
	r.PutBytes(&tx.R)
	s.PutBytes(&tx.S)
	wire, err := Encode(tx)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func TestRoundTripAndIdentity(t *testing.T) {
	wire := signedFixture(t, 25, 0)
	tx, err := Parse(wire)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := Encode(tx)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("round trip: %v", err)
	}
	if got := Validate(wire, fixtureChain); got != CodeOK {
		t.Fatalf("valid fixture: %d", got)
	}
	if _, err := SenderAddress(tx.Sender); err != nil {
		t.Fatal(err)
	}
	id, err := TxID(wire)
	if err != nil || id == CometHash(wire) {
		t.Fatalf("distinct valid IDs: %v", err)
	}
	if CometHash(wire) != sha256.Sum256(wire) {
		t.Fatal("CometHash mismatch")
	}
	if _, err := SigningPreimage(tx); err != nil {
		t.Fatal(err)
	}
}

func TestGoldenVectors(t *testing.T) {
	data, err := os.ReadFile("../../research/evidence/phase_03b/golden_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct {
		ID        string `json:"id"`
		ChainID   string `json:"chain_id"`
		Wire      string `json:"wire_hex"`
		Unsigned  string `json:"unsigned_payload_hex"`
		Preimage  string `json:"signing_preimage_hex"`
		Digest    string `json:"signing_digest_hex"`
		Sender    string `json:"sender_address_hex"`
		TxID      string `json:"pluto_txid_hex"`
		CometHash string `json:"comet_hash_hex"`
		Code      Code   `json:"expected_stateless_code"`
	}
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 4 {
		t.Fatalf("expected four frozen vectors, got %d", len(vectors))
	}
	for _, v := range vectors {
		t.Run(v.ID, func(t *testing.T) {
			wire, err := hex.DecodeString(v.Wire)
			if err != nil {
				t.Fatal(err)
			}
			tx, err := Parse(wire)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := Encode(tx)
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatal("round trip")
			}
			if hex.EncodeToString(wire[:len(wire)-64]) != v.Unsigned {
				t.Fatal("unsigned payload")
			}
			preimage, err := SigningPreimage(tx)
			if err != nil || hex.EncodeToString(preimage) != v.Preimage {
				t.Fatal("preimage")
			}
			digest, err := SigningDigest(tx)
			if err != nil || hex.EncodeToString(digest[:]) != v.Digest {
				t.Fatal("digest")
			}
			address, err := SenderAddress(tx.Sender)
			if err != nil || hex.EncodeToString(address[:]) != v.Sender {
				t.Fatal("sender address")
			}
			id, err := TxID(wire)
			if err != nil || hex.EncodeToString(id[:]) != v.TxID {
				t.Fatal("TxID")
			}
			comet := CometHash(wire)
			if hex.EncodeToString(comet[:]) != v.CometHash {
				t.Fatal("CometHash")
			}
			if got := Validate(wire, v.ChainID); got != v.Code {
				t.Fatalf("code %d, want %d", got, v.Code)
			}
		})
	}
}

func TestValidationPrecedenceAndTxID(t *testing.T) {
	valid := signedFixture(t, 25, 0)
	n := int(binary.BigEndian.Uint16(valid[10:12]))
	rAt, sAt := 81+n, 113+n
	cases := []struct {
		name   string
		mutate func([]byte) []byte
		chain  string
		code   Code
		id     bool
	}{
		{"bad magic", func(b []byte) []byte { b[0] = 0; return b }, fixtureChain, CodeMalformed, false},
		{"unknown version", func(b []byte) []byte { b[9] = 2; return b }, fixtureChain, CodeMalformed, false},
		{"short", func(b []byte) []byte { return b[:len(b)-1] }, fixtureChain, CodeMalformed, false},
		{"long trailing", func(b []byte) []byte { return append(b, 0) }, fixtureChain, CodeMalformed, false},
		{"chain length zero", func(b []byte) []byte { b[11] = 0; return b }, fixtureChain, CodeMalformed, false},
		{"chain length 51", func(b []byte) []byte { b[11] = 51; return b }, fixtureChain, CodeMalformed, false},
		{"chain character", func(b []byte) []byte { b[12] = 0xff; return b }, fixtureChain, CodeMalformed, false},
		{"wrong chain", func(b []byte) []byte { return b }, "other", CodeWrongChain, true},
		{"zero amount", func(b []byte) []byte { clear(b[65+n : 73+n]); return b }, fixtureChain, CodeZeroAmount, true},
		{"zero amount and bad signature", func(b []byte) []byte { clear(b[65+n : 73+n]); b[sAt+31] ^= 1; return b }, fixtureChain, CodeZeroAmount, true},
		{"zero r", func(b []byte) []byte { clear(b[rAt : rAt+32]); return b }, fixtureChain, CodeBadSig, false},
		{"zero s", func(b []byte) []byte { clear(b[sAt : sAt+32]); return b }, fixtureChain, CodeBadSig, false},
		{"r >= n", func(b []byte) []byte {
			for i := rAt; i < rAt+32; i++ {
				b[i] = 0xff
			}
			return b
		}, fixtureChain, CodeBadSig, false},
		{"s high", func(b []byte) []byte {
			for i := sAt; i < sAt+32; i++ {
				b[i] = 0xff
			}
			return b
		}, fixtureChain, CodeBadSig, false},
		{"bad point prefix", func(b []byte) []byte { b[12+n] = 4; return b }, fixtureChain, CodeBadKey, true},
		{"noncanonical x", func(b []byte) []byte {
			for i := 13 + n; i < 45+n; i++ {
				b[i] = 0xff
			}
			return b
		}, fixtureChain, CodeBadKey, true},
		{"modified amount", func(b []byte) []byte { b[72+n] ^= 1; return b }, fixtureChain, CodeBadSig, true},
		{"modified receiver", func(b []byte) []byte { b[64+n] ^= 1; return b }, fixtureChain, CodeBadSig, true},
		{"modified nonce", func(b []byte) []byte { b[80+n] ^= 1; return b }, fixtureChain, CodeBadSig, true},
		{"key substitution", func(b []byte) []byte {
			var secret [32]byte
			secret[31] = 2
			copy(b[12+n:45+n], secp256k1.PrivKeyFromBytes(secret[:]).PubKey().SerializeCompressed())
			return b
		}, fixtureChain, CodeBadSig, true},
		{"invalid but canonical r", func(b []byte) []byte { clear(b[rAt : rAt+32]); b[rAt+31] = 1; return b }, fixtureChain, CodeBadSig, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wire := c.mutate(bytes.Clone(valid))
			if got := Validate(wire, c.chain); got != c.code {
				t.Fatalf("code = %d, want %d", got, c.code)
			}
			_, err := TxID(wire)
			if (err == nil) != c.id {
				t.Fatalf("TxID exists = %v, want %v", err == nil, c.id)
			}
			if CometHash(wire) != sha256.Sum256(wire) {
				t.Fatal("CometHash mismatch")
			}
		})
	}
}

func TestBoundariesAndSignedFields(t *testing.T) {
	for _, chain := range []string{"c", "pluto-b0-v1", string(bytes.Repeat([]byte{'c'}, 50))} {
		wire := signedFixture(t, math.MaxUint64, math.MaxUint64)
		tx, _ := Parse(wire)
		tx.ChainID = chain
		// Re-sign the changed payload with the public test key.
		var secret [32]byte
		secret[31] = 1
		key := secp256k1.PrivKeyFromBytes(secret[:])
		digest, err := SigningDigest(tx)
		if err != nil {
			t.Fatal(err)
		}
		sig := ecdsa.Sign(key, digest[:])
		r, s := sig.R(), sig.S()
		r.PutBytes(&tx.R)
		s.PutBytes(&tx.S)
		encoded, err := Encode(tx)
		if err != nil {
			t.Fatal(err)
		}
		if len(encoded) != 145+len(chain) || Validate(encoded, chain) != CodeOK {
			t.Fatal("boundary signing failed")
		}
	}
	for _, chain := range []string{"", "bad chain", string(bytes.Repeat([]byte{'x'}, 51))} {
		if _, err := Encode(Transaction{ChainID: chain}); err == nil {
			t.Fatal("invalid chain accepted")
		}
	}
}

func TestHighSMalleabilityAndDoubleHash(t *testing.T) {
	wire := signedFixture(t, 25, 0)
	n := int(binary.BigEndian.Uint16(wire[10:12]))
	sAt := 113 + n
	order, ok := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	if !ok {
		t.Fatal("curve order")
	}
	s := new(big.Int).SetBytes(wire[sAt : sAt+32])
	high := new(big.Int).Sub(order, s).FillBytes(make([]byte, 32))
	copy(wire[sAt:sAt+32], high)
	if Validate(wire, fixtureChain) != CodeBadSig {
		t.Fatal("accepted high-S twin")
	}
	if _, err := TxID(wire); err == nil {
		t.Fatal("high-S twin acquired TxID")
	}

	wire = signedFixture(t, 25, 0)
	tx, err := Parse(wire)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := SigningDigest(tx)
	if err != nil {
		t.Fatal(err)
	}
	double := sha256.Sum256(digest[:])
	pub, err := parseKey(tx.Sender)
	if err != nil {
		t.Fatal(err)
	}
	var r, scalarS secp256k1.ModNScalar
	r.SetByteSlice(tx.R[:])
	scalarS.SetByteSlice(tx.S[:])
	sig := ecdsa.NewSignature(&r, &scalarS)
	if !sig.Verify(digest[:], pub) || sig.Verify(double[:], pub) {
		t.Fatal("digest verification unexpectedly double hashed")
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("PLUTOTX1"))
	f.Add(signedFixtureForFuzz())
	f.Fuzz(func(t *testing.T, raw []byte) {
		parsed, err := Parse(raw)
		if err == nil {
			encoded, err := Encode(parsed)
			if err != nil || !bytes.Equal(encoded, raw) {
				t.Fatal("noncanonical round trip")
			}
		}
		_, _ = TxID(raw)
		_ = CometHash(raw)
		_ = Validate(raw, fixtureChain)
	})
}

func FuzzCodec(f *testing.F) {
	f.Add("pluto-b0-v1", uint64(25), uint64(0))
	f.Add("c", uint64(math.MaxUint64), uint64(math.MaxUint64))
	f.Fuzz(func(t *testing.T, chain string, amount, nonce uint64) {
		var tx Transaction
		tx.ChainID, tx.Amount, tx.Nonce = chain, amount, nonce
		wire, err := Encode(tx)
		if err != nil {
			return
		}
		got, err := Parse(wire)
		if err != nil || got != tx {
			t.Fatal("codec mismatch")
		}
	})
}

func signedFixtureForFuzz() []byte {
	var secret [32]byte
	secret[31] = 1
	key := secp256k1.PrivKeyFromBytes(secret[:])
	var tx Transaction
	tx.ChainID = fixtureChain
	copy(tx.Sender[:], key.PubKey().SerializeCompressed())
	tx.Receiver[19] = 2
	tx.Amount = 25
	digest, _ := SigningDigest(tx)
	sig := ecdsa.Sign(key, digest[:])
	r, s := sig.R(), sig.S()
	r.PutBytes(&tx.R)
	s.PutBytes(&tx.S)
	wire, _ := Encode(tx)
	return wire
}
