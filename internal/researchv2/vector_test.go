package researchv2_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"example.com/pluto/internal/researchv2"
)

func TestPublicGoldenVectors(t *testing.T) {
	rawDoc, err := os.ReadFile("testdata/public_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Format        string `json:"format"`
		ChainID       string `json:"chain_id"`
		GenesisBase64 string `json:"genesis_base64"`
		GenesisSHA256 string `json:"genesis_sha256"`
		Sender        string `json:"sender"`
		Receiver      string `json:"receiver"`
		Amount        uint64 `json:"amount"`
		Nonce         uint64 `json:"nonce"`
		Vectors       []struct {
			Profile       string `json:"profile"`
			WireBase64    string `json:"wire_base64"`
			WireSize      int    `json:"wire_size"`
			WireSHA256    string `json:"wire_sha256"`
			TxID          string `json:"tx_id"`
			SigningDigest string `json:"signing_digest"`
			PostAppHash   string `json:"post_app_hash"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(rawDoc, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Format != "pluto-research-v2-s03-public-v1" || len(doc.Vectors) != 4 {
		t.Fatal("unexpected vector document")
	}
	genesis, err := base64.StdEncoding.DecodeString(doc.GenesisBase64)
	if err != nil {
		t.Fatal(err)
	}
	genesisDigest := sha256.Sum256(genesis)
	if hex.EncodeToString(genesisDigest[:]) != doc.GenesisSHA256 {
		t.Fatal("genesis hash changed")
	}
	base, err := researchv2.ParseGenesis(genesis, doc.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	mapProfile := map[string]researchv2.Profile{"C": researchv2.ProfileC, "CP": researchv2.ProfileCP, "P": researchv2.ProfileP, "H": researchv2.ProfileH}
	seen := map[string]bool{}
	sizes := map[string]int{}
	var reference [32]byte
	for _, entry := range doc.Vectors {
		p, found := mapProfile[entry.Profile]
		if !found || seen[entry.Profile] {
			t.Fatalf("unknown/duplicate profile %q", entry.Profile)
		}
		seen[entry.Profile] = true
		raw, err := base64.StdEncoding.DecodeString(entry.WireBase64)
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) != entry.WireSize {
			t.Fatalf("%s wire length changed", entry.Profile)
		}
		wantSize, err := p.WireSize(doc.ChainID)
		if err != nil || len(raw) != wantSize {
			t.Fatalf("%s canonical wire size", entry.Profile)
		}
		sizes[entry.Profile] = len(raw)
		wireHash := sha256.Sum256(raw)
		if hex.EncodeToString(wireHash[:]) != entry.WireSHA256 {
			t.Fatalf("%s wire hash changed", entry.Profile)
		}
		tx, err := researchv2.Parse(raw)
		if err != nil || tx.Profile != p || tx.ChainID != doc.ChainID || tx.Amount != doc.Amount || tx.Nonce != doc.Nonce || hex.EncodeToString(tx.Sender[:]) != doc.Sender || hex.EncodeToString(tx.Receiver[:]) != doc.Receiver {
			t.Fatalf("%s intent changed: %v", entry.Profile, err)
		}
		encoded, err := researchv2.Encode(tx)
		if err != nil || !bytes.Equal(encoded, raw) {
			t.Fatalf("%s noncanonical wire", entry.Profile)
		}
		digest, err := researchv2.SigningDigest(tx)
		if err != nil || hex.EncodeToString(digest[:]) != entry.SigningDigest {
			t.Fatalf("%s signing digest changed", entry.Profile)
		}
		id, err := researchv2.TxID(raw)
		if err != nil || hex.EncodeToString(id[:]) != entry.TxID {
			t.Fatalf("%s TxID changed", entry.Profile)
		}
		next, result, err := researchv2.Evaluate(base, p, raw)
		postHash := next.AppHash()
		if err != nil || result.Code != researchv2.CodeOK || hex.EncodeToString(postHash[:]) != entry.PostAppHash {
			t.Fatalf("%s vector rejected or state changed: %+v %v", entry.Profile, result, err)
		}
		if reference == ([32]byte{}) {
			reference = next.AppHash()
		} else if next.AppHash() != reference {
			t.Fatalf("%s same intent different AppHash", entry.Profile)
		}
	}
	if len(seen) != 4 || sizes["CP"] != sizes["H"] {
		t.Fatal("incomplete or unequal CP/H public vectors")
	}
}
