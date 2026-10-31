package genesis_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	b1genesis "example.com/pluto/internal/hybrid/genesis"
	b1test "example.com/pluto/internal/hybrid/testutil"
)

func TestB1CanonicalGenesisTransport(t *testing.T) {
	f, err := b1test.New("pluto-b1-transport")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := b1genesis.Encode(f.Genesis, f.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	decoded, state, err := b1genesis.Decode(encoded, f.ChainID)
	if err != nil || !bytes.Equal(decoded, f.Genesis) || state.Height() != 0 {
		t.Fatalf("canonical transport: %v", err)
	}
	if _, _, err := b1genesis.Decode(encoded, "other-chain"); err == nil {
		t.Fatal("wrong chain accepted")
	}
	if _, _, err := b1genesis.Decode(append(bytes.Clone(encoded), 'x'), f.ChainID); err == nil {
		t.Fatal("trailing garbage accepted")
	}
	var base64Text string
	if err := json.Unmarshal(encoded, &base64Text); err != nil {
		t.Fatal(err)
	}
	for _, raw := range [][]byte{
		[]byte(base64Text),
		[]byte(`"` + base64.RawStdEncoding.EncodeToString(f.Genesis) + `"`),
		[]byte(`{"genesis":"` + base64Text + `"}`),
		append(bytes.Repeat([]byte{' '}, 65), encoded...),
	} {
		if _, _, err := b1genesis.Decode(raw, f.ChainID); err == nil {
			t.Fatalf("noncanonical B1 app_state accepted length=%d", len(raw))
		}
	}
}
