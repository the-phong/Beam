package corpus

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const publicCorpus = "../../../research/corpus/phase_05/public_corpus.json"

func TestPublicCorpusOracle(t *testing.T) {
	c, _, err := Load(publicCorpus)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckOracle(c); err != nil {
		t.Fatal(err)
	}
}

func TestCorpusIntegrityAndNoOverwrite(t *testing.T) {
	c, _, err := Load(publicCorpus)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "copy.json")
	if err := WriteFresh(path, c); err != nil {
		t.Fatal(err)
	}
	if err := WriteFresh(path, c); err == nil {
		t.Fatal("overwrote existing corpus")
	}
	cs := c.Profiles[1].Cases[0]
	cs.SHA256 = "0000"
	if _, err := cs.Bytes(); err == nil {
		t.Fatal("accepted altered digest")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, []byte("{}")...)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(path); err == nil {
		t.Fatal("accepted trailing JSON")
	}
}

func FuzzCorpusDecoder(f *testing.F) {
	raw, err := os.ReadFile(publicCorpus)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(raw[:min(len(raw), 1024)])
	f.Add([]byte(`{"version":"pluto-phase05-corpus-v1"}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > MaxFileBytes {
			t.Skip()
		}
		var c Corpus
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if dec.Decode(&c) == nil {
			_ = ValidateStructure(c)
		}
	})
}
