package b1tx_test

import (
	"testing"

	"example.com/pluto/internal/hybrid/tx"
)

func TestHybridEnvelopeBounds(t *testing.T) {
	if b1tx.MinWireSize != 3898 || b1tx.MaxWireSize != 3947 {
		t.Fatalf("wire bounds changed min=%d max=%d", b1tx.MinWireSize, b1tx.MaxWireSize)
	}
	for _, raw := range [][]byte{nil, make([]byte, b1tx.MinWireSize-1), make([]byte, b1tx.MaxWireSize+1)} {
		if _, err := b1tx.Parse(raw); err == nil {
			t.Fatal("out-of-bounds envelope accepted")
		}
	}
}
