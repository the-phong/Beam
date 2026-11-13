package bench

import (
	"os"
	"testing"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	researchv2 "example.com/pluto/internal/experiment/profiles"
	researchv2test "example.com/pluto/internal/experiment/profiles/testfixture"
)

var file02R01Result bool

// BenchmarkFile02R01 measures only the two verification primitives for a
// valid H intent. Key generation, wire parsing and digest formation are
// outside the timer; R02 measures the complete validation path separately.
func BenchmarkFile02R01(b *testing.B) {
	f, err := researchv2test.New("pluto-file02-r01")
	if err != nil {
		b.Fatal(err)
	}
	raw, err := f.Signed(researchv2.ProfileH, 1, 0)
	if err != nil {
		b.Fatal(err)
	}
	tx, err := researchv2.Parse(raw)
	if err != nil {
		b.Fatal(err)
	}
	digest, err := researchv2.SigningDigest(tx)
	if err != nil {
		b.Fatal(err)
	}
	secpPub, err := secp256k1.ParsePubKey(f.Sender.Binding.SecpKey[:])
	if err != nil {
		b.Fatal(err)
	}
	var r, s secp256k1.ModNScalar
	if r.SetByteSlice(tx.SecpSignature[:32]) || s.SetByteSlice(tx.SecpSignature[32:]) || r.IsZero() || s.IsZero() || s.IsOverHalfOrder() {
		b.Fatal("invalid fixture secp signature")
	}
	secpSig := ecdsa.NewSignature(&r, &s)
	var mlPub mldsa44.PublicKey
	if err := mlPub.UnmarshalBinary(f.Sender.Binding.MLKey[:]); err != nil {
		b.Fatal(err)
	}
	context := []byte(researchv2.MLContext)
	if !secpSig.Verify(digest[:], secpPub) || !mldsa44.Verify(&mlPub, digest[:], context, tx.MLSignature[:]) {
		b.Fatal("fixture verification failed")
	}
	secpCase := func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			file02R01Result = secpSig.Verify(digest[:], secpPub)
		}
		if !file02R01Result {
			b.Fatal("valid secp proof rejected")
		}
	}
	mlCase := func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			file02R01Result = mldsa44.Verify(&mlPub, digest[:], context, tx.MLSignature[:])
		}
		if !file02R01Result {
			b.Fatal("valid ML proof rejected")
		}
	}
	if os.Getenv("FILE02_R01_ORDER") == "ml_first" {
		b.Run("mldsa44", mlCase)
		b.Run("secp256k1", secpCase)
	} else {
		b.Run("secp256k1", secpCase)
		b.Run("mldsa44", mlCase)
	}
}
