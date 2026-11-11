package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIngressProxyClassifiesWithoutRawBytes(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/broadcast_tx_sync" || r.URL.Query().Get("tx") != "0x0102" {
			t.Fatal("proxy changed transaction request")
		}
		fmt.Fprint(w, `{"result":{"code":4}}`)
	}))
	defer backend.Close()
	p, err := NewIngressProxy(backend.URL, 1)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	p.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/broadcast_tx_sync?tx=0x0102", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("proxy status %d", w.Code)
	}
	events, drops := p.Drain()
	if len(events) != 1 || drops != 0 || events[0].IngressUTC == "" || events[0].ObservedUTC == "" || events[0].WireBytes != 2 || events[0].Reason != "app_reject" || events[0].Code == nil || *events[0].Code != 4 || events[0].RoundTripNS < 0 {
		t.Fatalf("wrong proxy event: %+v drops=%d", events, drops)
	}
	raw, err := json.Marshal(events[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" || containsAny(string(raw), "0102", "tx", "sender", "signature") {
		t.Fatalf("event leaks private fields: %s", raw)
	}
	p.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/broadcast_tx_sync?tx=0x0102", nil))
	p.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/broadcast_tx_sync?tx=0x0102", nil))
	_, drops = p.Drain()
	if drops != 1 {
		t.Fatalf("bounded proxy did not count drop: %d", drops)
	}
}

func TestClientRPCIsSeparateFromProxyRoundTrip(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"result":{"code":0}}`)
	}))
	defer backend.Close()
	proxy, err := NewIngressProxy(backend.URL, 2)
	if err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(proxy)
	defer front.Close()
	client, err := NewSubmitter(front.URL)
	if err != nil {
		t.Fatal(err)
	}
	submission, err := client.BroadcastSync(context.Background(), []byte{1, 2})
	if err != nil || submission.SentUTC == "" || submission.ObservedUTC == "" || submission.Code == nil || *submission.Code != 0 || submission.ClientRPCNS <= 0 {
		t.Fatalf("client timing: %+v %v", submission, err)
	}
	events, dropped := proxy.Drain()
	if len(events) != 1 || dropped != 0 || events[0].Reason != "accepted_by_checktx" || events[0].RoundTripNS <= 0 {
		t.Fatalf("proxy timing: %+v drops=%d", events, dropped)
	}
}

func containsAny(s string, values ...string) bool {
	for _, v := range values {
		if len(v) != 0 && strings.Contains(s, v) {
			return true
		}
	}
	return false
}
