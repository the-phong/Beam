//go:build integration && fourval

package integration

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	"example.com/pluto/internal/b1test"
	"example.com/pluto/internal/b1tx"
)

// A network with only two of four validators cannot commit. This leaves a
// stable window to observe mempool P2P relay independently of block delivery.
func TestResearchV2M01P2PRelayBoundary(t *testing.T) {
	fixture, err := b1test.New("pluto-rv2-m01-relay")
	if err != nil {
		t.Fatal(err)
	}
	network := newHybridNetwork(t, fixture)
	network.start(0)
	network.start(1)
	defer network.stopAll()

	ready := time.Now().Add(30 * time.Second)
	for time.Now().Before(ready) {
		_, _, err0 := network.height(0)
		_, _, err1 := network.height(1)
		if err0 == nil && err1 == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if _, _, err := network.height(0); err != nil {
		t.Fatalf("source RPC not ready: %v", err)
	}
	if _, _, err := network.height(1); err != nil {
		t.Fatalf("peer RPC not ready: %v", err)
	}
	wire, err := fixture.Signed(25, 0)
	if err != nil {
		t.Fatal(err)
	}
	var admission struct {
		Result struct {
			Code uint32 `json:"code"`
		} `json:"result"`
	}
	path := "/broadcast_tx_sync?tx=0x" + hex.EncodeToString(wire)
	if err := rpcGetLong(t, network.nodes[0].ports.RPC, path, &admission); err != nil || admission.Result.Code != 0 {
		t.Fatalf("valid RPC admission code=%d err=%v", admission.Result.Code, err)
	}

	// The peer did not receive this transaction over RPC. With no quorum, it
	// cannot have learned it from a committed proposal either.
	want := base64.StdEncoding.EncodeToString(wire)
	deadline := time.Now().Add(30 * time.Second)
	seen := false
	for time.Now().Before(deadline) {
		var pool struct {
			Result struct {
				Txs []string `json:"txs"`
			} `json:"result"`
		}
		if err := rpcGet(t, network.nodes[1].ports.RPC, "/unconfirmed_txs?limit=100", &pool); err == nil {
			for _, tx := range pool.Result.Txs {
				if tx == want {
					seen = true
					break
				}
			}
		}
		if seen {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !seen {
		t.Fatal("valid tx was not observed in peer accepted mempool before any commit")
	}
	for _, index := range []int{0, 1} {
		if height, _, err := network.height(index); err != nil || height != 0 {
			t.Fatalf("validator %d unexpectedly committed during relay probe: height=%d err=%v", index, height, err)
		}
	}

	bad := bytes.Clone(wire)
	bad[len(bad)-1] ^= 1
	badPath := "/broadcast_tx_sync?tx=0x" + hex.EncodeToString(bad)
	var rejection struct {
		Result struct {
			Code uint32 `json:"code"`
		} `json:"result"`
	}
	if err := rpcGetLong(t, network.nodes[0].ports.RPC, badPath, &rejection); err != nil || rejection.Result.Code != uint32(b1tx.CodeBadMLSignature) {
		t.Fatalf("bad ML proof admission code=%d err=%v", rejection.Result.Code, err)
	}
	var pool struct {
		Result struct {
			Txs []string `json:"txs"`
		} `json:"result"`
	}
	if err := rpcGet(t, network.nodes[1].ports.RPC, "/unconfirmed_txs?limit=100", &pool); err != nil {
		t.Fatal(err)
	}
	if len(pool.Result.Txs) != 1 || pool.Result.Txs[0] != want {
		t.Fatalf("peer accepted mempool changed after invalid submission: count=%d", len(pool.Result.Txs))
	}
	t.Logf("valid accepted tx relayed to peer mempool before commit; bad ML proof rejected; observed peer accepted count=%d", len(pool.Result.Txs))
}
