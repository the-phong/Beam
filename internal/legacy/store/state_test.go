package store

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"testing"
)

func TestCanonicalCommitmentAndEncoding(t *testing.T) {
	s, err := Genesis("pluto-test-1")
	if err != nil {
		t.Fatal(err)
	}
	input := append([]byte("PLUTO/APP_STATE/V1\x00"), 0, 0, 0, 1, 0, 12)
	input = append(input, []byte("pluto-test-1")...)
	input = binary.BigEndian.AppendUint64(input, 0)
	want := sha256.Sum256(input)
	if s.AppHash != want {
		t.Fatalf("genesis hash %x want %x", s.AppHash, want)
	}
	record, err := encode(s)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := decode(record)
	if err != nil || reloaded != s {
		t.Fatalf("round trip: %+v, %v", reloaded, err)
	}
	next, err := s.Next()
	if err != nil {
		t.Fatal(err)
	}
	if next.Height != 1 || next.AppHash == s.AppHash {
		t.Fatal("empty block must advance commitment")
	}
	if got := hex.EncodeToString(s.AppHash[:]); len(got) != 64 {
		t.Fatal(got)
	}
}

func TestInvalidState(t *testing.T) {
	for _, chain := range []string{"", "space id", "é", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"} {
		if _, err := Genesis(chain); err == nil {
			t.Errorf("accepted invalid chain ID %q", chain)
		}
	}
	s, _ := Genesis("pluto-test")
	record, _ := encode(s)
	bad := append([]byte(nil), record...)
	bad[len(bad)-1] ^= 1
	if _, err := decode(bad); err == nil {
		t.Fatal("accepted corrupted AppHash")
	}
	bad = append([]byte(nil), record...)
	bad[8] = 9
	if _, err := decode(bad); err == nil {
		t.Fatal("accepted unknown version")
	}
	if _, err := decode(record[:len(record)-1]); err == nil {
		t.Fatal("accepted truncated record")
	}
	if _, err := decode(append(record, 0)); err == nil {
		t.Fatal("accepted trailing byte")
	}
	s.Height = -1
	if err := s.Validate(); err == nil {
		t.Fatal("accepted negative height")
	}
	s.Height = math.MaxInt64
	s.AppHash = hashState(s)
	if _, err := s.Next(); err == nil {
		t.Fatal("accepted height overflow")
	}
}
