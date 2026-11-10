package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCollectorKeepsAcceptedOccupancySeparateFromFailuresAndRelay(t *testing.T) {
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/num_unconfirmed_txs" {
			t.Fatal("unexpected RPC path")
		}
		fmt.Fprint(w, `{"result":{"total":"2","total_bytes":"2575"}}`)
	}))
	defer rpc.Close()
	metrics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "cometbft_mempool_failed_txs 7\ncometbft_mempool_rejected_txs 3\ncometbft_mempool_recheck_times 1\n")
		fmt.Fprint(w, "cometbft_p2p_peer_send_bytes_total{peer_id=\"a\",chID=\"0x30\"} 100\n")
		fmt.Fprint(w, "cometbft_p2p_peer_send_bytes_total{peer_id=\"b\",chID=\"0x20\"} 800\n")
		fmt.Fprint(w, "cometbft_p2p_peer_receive_bytes_total{peer_id=\"b\",chID=\"0x30\"} 200\n")
	}))
	defer metrics.Close()
	c, err := NewCollector(rpc.URL, metrics.URL, os.Getpid(), os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	s := c.Sample(context.Background())
	if s.SampleStartedUTC == "" || s.ObservedUTC == "" || s.SampleDurationNS < 0 || s.AcceptedMempoolCount == nil || *s.AcceptedMempoolCount != 2 || s.AcceptedMempoolBytes == nil || *s.AcceptedMempoolBytes != 2575 || s.FailedTxTotal == nil || *s.FailedTxTotal != 7 || s.ResourceRejectedTxTotal == nil || *s.ResourceRejectedTxTotal != 3 || s.RelaySendEnvelopeBytes == nil || *s.RelaySendEnvelopeBytes != 100 || s.RelayReceiveEnvelopeBytes == nil || *s.RelayReceiveEnvelopeBytes != 200 || s.Comet == nil || s.Application == nil || len(s.Unavailable) != 0 || s.DroppedSamples != 0 {
		t.Fatalf("incorrect sample: %+v", s)
	}
	rpc.Close()
	s = c.Sample(context.Background())
	if s.AcceptedMempoolCount != nil || s.AcceptedMempoolBytes != nil || s.DroppedSamples != 1 || len(s.Unavailable) == 0 {
		t.Fatalf("missing RPC was converted to zero: %+v", s)
	}
}

func TestCollectorRejectsNonLocalEndpoints(t *testing.T) {
	for _, endpoint := range []string{"https://127.0.0.1:26657", "http://example.com:26657", "http://127.0.0.1:26657/path", "http://user:password@127.0.0.1:26657"} {
		if _, err := NewCollector(endpoint, "http://127.0.0.1:26660", 1, 1); err == nil {
			t.Fatalf("accepted endpoint %q", endpoint)
		}
	}
}
