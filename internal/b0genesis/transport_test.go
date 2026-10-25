package b0genesis

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"example.com/pluto/internal/b0state"
)

const goldenGenesis = "504c55544f47322100000002000b706c75746f2d62302d763100000001b9631c0f8d9d2d9d56bd6151559f3339e14eb36a0000000000000064"

func golden(t *testing.T) []byte {
	t.Helper()
	b, err := hex.DecodeString(goldenGenesis)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGoldenTransportAndState(t *testing.T) {
	binaryGenesis := golden(t)
	encoded, err := Encode(binaryGenesis, "pluto-b0-v1")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(base64.StdEncoding.EncodeToString(binaryGenesis))
	if !bytes.Equal(encoded, want) {
		t.Fatalf("transport %q != %q", encoded, want)
	}
	recovered, state, err := Decode(encoded, "pluto-b0-v1")
	if err != nil || !bytes.Equal(recovered, binaryGenesis) {
		t.Fatalf("roundtrip: %v", err)
	}
	if state.Height() != 0 || state.TotalSupply() != 100 || len(state.Accounts()) != 1 || state.Accounts()[0].Balance != 100 || state.Accounts()[0].Nonce != 0 {
		t.Fatalf("wrong genesis state: %+v", state.Accounts())
	}
	hash := state.AppHash()
	if got := hex.EncodeToString(hash[:]); got != "e480c8b300c5ac1d662628f90334aae185ad38ed791c59ab14e3aa42abf083da" {
		t.Fatalf("AppHash %s", got)
	}
	withWhitespace := append([]byte(" \n"), encoded...)
	withWhitespace = append(withWhitespace, '\t')
	if _, _, err := Decode(withWhitespace, "pluto-b0-v1"); err != nil {
		t.Fatalf("JSON surrounding whitespace: %v", err)
	}
}

func TestTransportRejectsAlternates(t *testing.T) {
	g := golden(t)
	valid, err := Encode(g, "pluto-b0-v1")
	if err != nil {
		t.Fatal(err)
	}
	otherChain, _ := b0state.EncodeGenesis("pluto-other-1", []b0state.Allocation{{Address: b0state.Address{1}, Balance: 1}})
	wrongChainJSON, _ := json.Marshal(base64.StdEncoding.EncodeToString(otherChain))
	broken := bytes.Clone(g)
	broken[0] ^= 1
	brokenJSON, _ := json.Marshal(base64.StdEncoding.EncodeToString(broken))
	badPadding, _ := json.Marshal("Zg=")
	badPadBits, _ := json.Marshal("Zh==")
	for name, raw := range map[string][]byte{
		"null": []byte("null"), "object": []byte("{}"), "array": []byte("[]"),
		"boolean": []byte("true"), "number": []byte("1"),
		"trailing value": append(bytes.Clone(valid), []byte(" 0")...),
		"truncated":      valid[:len(valid)-1],
		"bad alphabet":   []byte(`"***="`), "bad padding": badPadding,
		"nonzero pad bits":    badPadBits,
		"escaped JSON":        []byte(`"\u0055` + string(valid[2:])),
		"non-JSON whitespace": append(bytes.Clone(valid), '\v'),
		"Unicode whitespace":  append(bytes.Clone(valid), []byte{0xc2, 0xa0}...),
		"oversized JSON":      bytes.Repeat([]byte(" "), MaxJSONSize+1),
		"wrong binary magic":  brokenJSON,
		"wrong chain":         wrongChainJSON,
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := Decode(raw, "pluto-b0-v1"); !errors.Is(err, ErrTransport) {
				t.Fatalf("error %v; want ErrTransport", err)
			}
		})
	}
	if _, err := Encode(broken, "pluto-b0-v1"); !errors.Is(err, ErrTransport) {
		t.Fatalf("Encode accepted bad binary: %v", err)
	}
}

func TestMalformedBinaryThroughTransport(t *testing.T) {
	g := golden(t)
	clone := func() []byte { return bytes.Clone(g) }
	badVersion := clone()
	badVersion[11] = 3
	badLength := clone()
	badLength[28] = 2 // K=2, but only one account record follows.
	zeroBalance := clone()
	clear(zeroBalance[len(zeroBalance)-8:])
	two, err := b0state.EncodeGenesis("pluto-b0-v1", []b0state.Allocation{
		{Address: b0state.Address{1}, Balance: 1},
		{Address: b0state.Address{2}, Balance: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	const first = 8 + 4 + 2 + 11 + 4
	unsorted := bytes.Clone(two)
	copy(unsorted[first:first+28], two[first+28:first+56])
	copy(unsorted[first+28:first+56], two[first:first+28])
	duplicate := bytes.Clone(two)
	copy(duplicate[first+28:first+48], duplicate[first:first+20])
	overflow := bytes.Clone(two)
	binary.BigEndian.PutUint64(overflow[first+20:first+28], ^uint64(0))
	for name, malformed := range map[string][]byte{
		"version": badVersion, "count length": badLength,
		"zero allocation": zeroBalance, "trailing byte": append(clone(), 0),
		"unsorted": unsorted, "duplicate": duplicate, "supply overflow": overflow,
	} {
		t.Run(name, func(t *testing.T) {
			wrapped, _ := json.Marshal(base64.StdEncoding.EncodeToString(malformed))
			if _, _, err := Decode(wrapped, "pluto-b0-v1"); !errors.Is(err, ErrTransport) {
				t.Fatalf("malformed binary accepted: %v", err)
			}
		})
	}
}

func TestMaximumLengthCalculation(t *testing.T) {
	const wantBinary = 1_835_076
	const wantBase64 = 2_446_768
	if MaxBinarySize != wantBinary || MaxBase64Size != wantBase64 || base64.StdEncoding.EncodedLen(wantBinary) != wantBase64 {
		t.Fatalf("limits: binary=%d base64=%d", MaxBinarySize, MaxBase64Size)
	}
	accounts := make([]b0state.Allocation, b0state.MaxAccounts)
	for i := range accounts {
		binary.BigEndian.PutUint32(accounts[i].Address[16:], uint32(i))
		accounts[i].Balance = 1
	}
	chain := string(bytes.Repeat([]byte{'c'}, 50))
	g, err := b0state.EncodeGenesis(chain, accounts)
	if err != nil || len(g) != wantBinary {
		t.Fatalf("max genesis: len=%d err=%v", len(g), err)
	}
	encoded, err := Encode(g, chain)
	if err != nil || len(encoded) != wantBase64+2 {
		t.Fatalf("max transport: len=%d err=%v", len(encoded), err)
	}
	recovered, state, err := Decode(encoded, chain)
	if err != nil || !bytes.Equal(recovered, g) || len(state.Accounts()) != b0state.MaxAccounts {
		t.Fatalf("max transport roundtrip: %v", err)
	}
	testWhitespaceBoundary(t, encoded, g, chain)
}

func TestSmallTransportWhitespaceBoundary(t *testing.T) {
	g := golden(t)
	encoded, err := Encode(g, "pluto-b0-v1")
	if err != nil {
		t.Fatal(err)
	}
	testWhitespaceBoundary(t, encoded, g, "pluto-b0-v1")
}

func testWhitespaceBoundary(t *testing.T, encoded, binaryGenesis []byte, chain string) {
	t.Helper()
	// The 64-byte allowance is total surrounding JSON whitespace, regardless
	// of how it is split across the two sides of the canonical string.
	allowed := append(bytes.Repeat([]byte(" \t\n\r"), 8), encoded...)
	allowed = append(allowed, bytes.Repeat([]byte("\r\n\t "), 8)...)
	if got, _, err := Decode(allowed, chain); err != nil || !bytes.Equal(got, binaryGenesis) {
		t.Fatalf("64 surrounding whitespace bytes rejected: %v", err)
	}
	tooMuch := append(bytes.Clone(allowed), ' ')
	if _, _, err := Decode(tooMuch, chain); !errors.Is(err, ErrTransport) {
		t.Fatalf("65 surrounding whitespace bytes accepted: %v", err)
	}
	if len(encoded) == MaxBase64Size+2 {
		if len(allowed) != MaxJSONSize || len(tooMuch) != MaxJSONSize+1 {
			t.Fatal("maximum transport boundary calculation changed")
		}
	}
}
