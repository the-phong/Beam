// Package corpus defines public, replayable Phase 05 transaction inputs.
// Generate uses ephemeral signing keys; only public genesis and signed bytes
// leave the generator. Loading and oracle evaluation require no private key.
package corpus

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"example.com/pluto/internal/b0state"
)

const Version = "pluto-phase05-corpus-v2"
const ContentionVersion = "pluto-phase05-corpus-contention-v1"
const SenderCount = 64
const ContentionNonceLayers = 6
const MaxFileBytes = 16 << 20

type Vector struct {
	ID          string `json:"id"`
	WireHex     string `json:"wire_hex"`
	SHA256      string `json:"sha256"`
	SenderHex   string `json:"sender_hex"`
	ReceiverHex string `json:"receiver_hex"`
	Size        int    `json:"size"`
	Nonce       uint64 `json:"nonce,omitempty"`
}

type Case struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Class          string `json:"class"`
	Capability     string `json:"capability"`
	Context        string `json:"context"`
	Construction   string `json:"construction"`
	WireHex        string `json:"wire_hex,omitempty"`
	Reconstruction string `json:"reconstruction,omitempty"`
	SHA256         string `json:"sha256"`
	Size           int    `json:"size"`
	ExpectedCode   uint32 `json:"expected_checktx_code"`
	ReachedStage   string `json:"reached_stage"`
	StateEffect    string `json:"expected_state_effect"`
}

type Proposal struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Capability   string   `json:"capability"`
	TxIDs        []string `json:"tx_ids"`
	Expected     string   `json:"expected_process_proposal"`
	Construction string   `json:"construction"`
}

type GenesisCase struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	GenesisHex   string `json:"genesis_hex"`
	SHA256       string `json:"sha256"`
	Expected     string `json:"expected_result"`
	Construction string `json:"construction"`
}

type Profile struct {
	Name          string        `json:"name"`
	ChainID       string        `json:"chain_id"`
	GenesisHex    string        `json:"genesis_hex"`
	GenesisSHA256 string        `json:"genesis_sha256"`
	AccountCount  int           `json:"account_count"`
	Valid         []Vector      `json:"valid_pool"`
	Cases         []Case        `json:"cases"`
	Proposals     []Proposal    `json:"proposals"`
	GenesisCases  []GenesisCase `json:"genesis_cases,omitempty"`
}

type Corpus struct {
	Version  string    `json:"version"`
	Profiles []Profile `json:"profiles"`
}

func digest(raw []byte) string {
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

func vector(id string, wire []byte, sender, receiver b0state.Address) Vector {
	return Vector{ID: id, WireHex: hex.EncodeToString(wire), SHA256: digest(wire),
		SenderHex: hex.EncodeToString(sender[:]), ReceiverHex: hex.EncodeToString(receiver[:]), Size: len(wire)}
}

func vectorAtNonce(id string, wire []byte, sender, receiver b0state.Address, nonce uint64) Vector {
	v := vector(id, wire, sender, receiver)
	v.Nonce = nonce
	return v
}

func addCase(p *Profile, id, name, class, capability, context, construction string, wire []byte, code uint32, stage, effect string) {
	p.Cases = append(p.Cases, Case{ID: id, Name: name, Class: class, Capability: capability,
		Context: context, Construction: construction, WireHex: hex.EncodeToString(wire),
		SHA256: digest(wire), Size: len(wire), ExpectedCode: code,
		ReachedStage: stage, StateEffect: effect})
}

func addZeroRepeatCase(p *Profile, id, name, class, capability string, n int, code uint32, stage string) {
	wire := make([]byte, n)
	p.Cases = append(p.Cases, Case{ID: id, Name: name, Class: class, Capability: capability,
		Context: "initial", Construction: fmt.Sprintf("repeat 0x00 exactly %d times", n),
		Reconstruction: fmt.Sprintf("zero:%d", n), SHA256: digest(wire), Size: n,
		ExpectedCode: code, ReachedStage: stage, StateEffect: "none"})
}

func addProposal(p *Profile, id, name, construction string, ids ...string) {
	p.Proposals = append(p.Proposals, Proposal{ID: id, Name: name,
		Capability: "C: registered proposer or direct laboratory injection", TxIDs: ids,
		Expected: "REJECT", Construction: construction})
}

func (c Case) Bytes() ([]byte, error) {
	var raw []byte
	var err error
	if c.Reconstruction != "" {
		var n int
		if _, err = fmt.Sscanf(c.Reconstruction, "zero:%d", &n); err != nil || n < 0 || n > (1<<20)+1 {
			return nil, errors.New("invalid corpus reconstruction")
		}
		raw = make([]byte, n)
	} else {
		raw, err = hex.DecodeString(c.WireHex)
		if err != nil {
			return nil, err
		}
	}
	if len(raw) != c.Size || digest(raw) != c.SHA256 {
		return nil, errors.New("corpus case size or SHA-256 mismatch")
	}
	return raw, nil
}

func (v Vector) Bytes() ([]byte, error) {
	raw, err := hex.DecodeString(v.WireHex)
	if err != nil || len(raw) != v.Size || digest(raw) != v.SHA256 {
		return nil, errors.New("corpus vector size or SHA-256 mismatch")
	}
	return raw, nil
}

func (p Profile) Genesis() ([]byte, error) {
	raw, err := hex.DecodeString(p.GenesisHex)
	if err != nil || digest(raw) != p.GenesisSHA256 {
		return nil, errors.New("corpus genesis SHA-256 mismatch")
	}
	return raw, nil
}

func (p Profile) Case(id string) (Case, bool) {
	for _, c := range p.Cases {
		if c.ID == id {
			return c, true
		}
	}
	return Case{}, false
}

func (p Profile) Tx(id string) ([]byte, error) {
	for _, v := range p.Valid {
		if v.ID == id {
			return v.Bytes()
		}
	}
	if c, ok := p.Case(id); ok {
		return c.Bytes()
	}
	return nil, fmt.Errorf("unknown corpus transaction %q", id)
}

func (c Corpus) Profile(name string) (Profile, bool) {
	for _, p := range c.Profiles {
		if p.Name == name {
			return p, true
		}
	}
	return Profile{}, false
}

func Generate() (Corpus, error) {
	b0, err := generateB0()
	if err != nil {
		return Corpus{}, err
	}
	b1, err := generateB1()
	if err != nil {
		return Corpus{}, err
	}
	return Corpus{Version: Version, Profiles: []Profile{b0, b1}}, nil
}

// GenerateContention creates public, pre-signed sequential transactions.
// Signing keys remain ephemeral and are never present in the returned corpus.
func GenerateContention() (Corpus, error) {
	b0, err := generateB0WithLayers(ContentionNonceLayers)
	if err != nil {
		return Corpus{}, err
	}
	b1, err := generateB1WithLayers(ContentionNonceLayers)
	if err != nil {
		return Corpus{}, err
	}
	return Corpus{Version: ContentionVersion, Profiles: []Profile{b0, b1}}, nil
}

// WriteFresh never overwrites an existing public corpus. The private keys
// used by Generate remain in memory and are not serialized by these types.
func WriteFresh(path string, c Corpus) error {
	if err := ValidateStructure(c); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > MaxFileBytes {
		return errors.New("corpus exceeds file size cap")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func Load(path string) (Corpus, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return Corpus{}, "", err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil || len(raw) > MaxFileBytes {
		return Corpus{}, "", errors.New("corpus file unreadable or oversized")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var c Corpus
	if err := decoder.Decode(&c); err != nil {
		return Corpus{}, "", err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Corpus{}, "", errors.New("trailing corpus JSON")
	}
	if err := ValidateStructure(c); err != nil {
		return Corpus{}, "", err
	}
	return c, digest(raw), nil
}

func ValidateStructure(c Corpus) error {
	if (c.Version != Version && c.Version != ContentionVersion) || len(c.Profiles) != 2 || c.Profiles[0].Name != "b0" || c.Profiles[1].Name != "b1" {
		return errors.New("invalid corpus version or profile set")
	}
	expectedVectors := SenderCount
	if c.Version == ContentionVersion {
		expectedVectors *= ContentionNonceLayers
	}
	for _, p := range c.Profiles {
		if p.AccountCount != SenderCount+1 || len(p.Valid) != expectedVectors || len(p.Cases) < 20 || len(p.Proposals) < 4 {
			return fmt.Errorf("incomplete %s corpus", p.Name)
		}
		if _, err := p.Genesis(); err != nil {
			return err
		}
		seen := make(map[string]bool)
		for i, v := range p.Valid {
			if v.ID == "" || seen[v.ID] {
				return errors.New("duplicate or missing vector ID")
			}
			if c.Version == ContentionVersion && (v.Nonce != uint64(i/SenderCount) || v.SenderHex != p.Valid[i%SenderCount].SenderHex) {
				return errors.New("invalid contention nonce or sender sequence")
			}
			seen[v.ID] = true
			if _, err := v.Bytes(); err != nil {
				return err
			}
		}
		for _, cs := range p.Cases {
			if cs.ID == "" || seen[cs.ID] || cs.Name == "" || cs.Class == "" || cs.Capability == "" || cs.Context == "" || cs.ReachedStage == "" || cs.StateEffect == "" {
				return errors.New("incomplete or duplicate corpus case")
			}
			seen[cs.ID] = true
			if _, err := cs.Bytes(); err != nil {
				return err
			}
		}
		for _, proposal := range p.Proposals {
			if proposal.ID == "" || len(proposal.TxIDs) == 0 || proposal.Expected != "REJECT" {
				return errors.New("incomplete proposal case")
			}
			for _, id := range proposal.TxIDs {
				if _, err := p.Tx(id); err != nil {
					return err
				}
			}
		}
		for _, gc := range p.GenesisCases {
			raw, err := hex.DecodeString(gc.GenesisHex)
			if err != nil || digest(raw) != gc.SHA256 || !strings.HasPrefix(gc.ID, "G") {
				return errors.New("invalid genesis case")
			}
		}
	}
	return nil
}
