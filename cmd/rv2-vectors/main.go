// rv2-vectors emits public synthetic S03 verification vectors. It generates
// ephemeral private keys in memory and never prints keys or signing seeds.
package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"example.com/pluto/internal/researchv2"
	"example.com/pluto/internal/researchv2test"
)

type vector struct {
	Profile       string `json:"profile"`
	WireBase64    string `json:"wire_base64"`
	WireSize      int    `json:"wire_size"`
	WireSHA256    string `json:"wire_sha256"`
	TxID          string `json:"tx_id"`
	SigningDigest string `json:"signing_digest"`
	PostAppHash   string `json:"post_app_hash"`
}
type document struct {
	Format        string   `json:"format"`
	ChainID       string   `json:"chain_id"`
	GenesisBase64 string   `json:"genesis_base64"`
	GenesisSHA256 string   `json:"genesis_sha256"`
	Sender        string   `json:"sender"`
	Receiver      string   `json:"receiver"`
	Amount        uint64   `json:"amount"`
	Nonce         uint64   `json:"nonce"`
	Vectors       []vector `json:"vectors"`
}

func hex32(h [32]byte) string { return hex.EncodeToString(h[:]) }
func run() error {
	f, err := researchv2test.New("rv2-golden-v1")
	if err != nil {
		return err
	}
	genesisHash := sha256.Sum256(f.Genesis)
	doc := document{Format: "pluto-research-v2-s03-public-v1", ChainID: f.ChainID, GenesisBase64: base64.StdEncoding.EncodeToString(f.Genesis), GenesisSHA256: hex32(genesisHash), Sender: hex.EncodeToString(f.Sender.Binding.Account[:]), Receiver: hex.EncodeToString(f.Receiver.Binding.Account[:]), Amount: 7, Nonce: 0}
	for _, entry := range []struct {
		name    string
		profile researchv2.Profile
	}{{"C", researchv2.ProfileC}, {"CP", researchv2.ProfileCP}, {"P", researchv2.ProfileP}, {"H", researchv2.ProfileH}} {
		raw, err := f.Signed(entry.profile, doc.Amount, doc.Nonce)
		if err != nil {
			return err
		}
		tx, err := researchv2.Parse(raw)
		if err != nil {
			return err
		}
		digest, err := researchv2.SigningDigest(tx)
		if err != nil {
			return err
		}
		id, err := researchv2.TxID(raw)
		if err != nil {
			return err
		}
		state, err := researchv2.ParseGenesis(f.Genesis, f.ChainID)
		if err != nil {
			return err
		}
		next, result, err := researchv2.EvaluateTrusted(state, entry.profile, raw)
		if err != nil || result.Code != researchv2.CodeOK {
			return fmt.Errorf("profile %s failed vector evaluation: %+v: %w", entry.name, result, err)
		}
		wireHash := sha256.Sum256(raw)
		doc.Vectors = append(doc.Vectors, vector{Profile: entry.name, WireBase64: base64.StdEncoding.EncodeToString(raw), WireSize: len(raw), WireSHA256: hex32(wireHash), TxID: hex32(id), SigningDigest: hex32(digest), PostAppHash: hex32(next.AppHash())})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
