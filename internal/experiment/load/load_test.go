package load

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"example.com/pluto/internal/experiment/profiles"
)

func pilotConfig(t *testing.T) (Config, string) {
	t.Helper()
	c, hash, err := ReadConfig(filepath.Join("..", "..", "..", "research", "config", "s06_pilot.json"))
	if err != nil {
		t.Fatal(err)
	}
	return c, hash
}

func TestS06PilotPlansHaveDirectOracleAndMatchedRegimes(t *testing.T) {
	c, hash := pilotConfig(t)
	f, err := NewPilotFixture("pluto-s06-pilot", 90)
	if err != nil {
		t.Fatal(err)
	}
	var bytePlans [4]Plan
	for _, p := range []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH} {
		for _, regime := range []string{EqualTxRate, EqualByteRate, FixedLegitimatePlusAdversary} {
			plan, err := BuildPilotPlan(f, p, regime, "constant", fmt.Sprintf("pilot-p%d-%s", p, regime), 611, c, hash, 2)
			if err != nil {
				t.Fatalf("profile %d regime %s: %v", p, regime, err)
			}
			if plan.Regime == EqualByteRate {
				bytePlans[int(p)-1] = plan
			}
		}
	}
	for i := 0; i < len(bytePlans); i++ {
		if err := ComparablePlans(bytePlans[0], bytePlans[i]); err != nil {
			t.Fatalf("equal-byte profile %d not paired: %v", i+1, err)
		}
	}
	cpSize, _ := researchv2.ProfileCP.WireSize(f.ChainID)
	hSize, _ := researchv2.ProfileH.WireSize(f.ChainID)
	if cpSize != hSize {
		t.Fatal("CP and H lost exact wire-size equality")
	}
	burst, err := BuildPilotPlan(f, researchv2.ProfileH, EqualTxRate, "burst", "pilot-burst", 611, c, hash, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(burst.Events) < 2 || burst.Waveform != "burst" {
		t.Fatal("missing bounded burst schedule")
	}
	ramp, err := BuildPilotPlan(f, researchv2.ProfileH, EqualTxRate, "ramp", "pilot-ramp", 611, c, hash, 10)
	if err != nil || ramp.Waveform != "ramp" || len(ramp.Events) != 41 {
		t.Fatalf("missing bounded ramp schedule: events=%d err=%v", len(ramp.Events), err)
	}
	var windows [4]int
	for _, event := range ramp.Events[1:] {
		windows[(event.OffsetNS-int64(time.Second))/int64(time.Second)]++
	}
	if windows != [4]int{4, 8, 12, 16} {
		t.Fatalf("R16 ramp windows %v", windows)
	}
}

func TestS06PlanRefusesDuplicateAndUnsafeSource(t *testing.T) {
	c, hash := pilotConfig(t)
	f, err := NewPilotFixture("pluto-s06-plan", 12)
	if err != nil {
		t.Fatal(err)
	}
	p, err := BuildPilotPlan(f, researchv2.ProfileC, EqualTxRate, "constant", "pilot-safe", 17, c, hash, 2)
	if err != nil {
		t.Fatal(err)
	}
	p.Events[1].BytesHex, p.Events[1].SHA256 = p.Events[0].BytesHex, p.Events[0].SHA256
	if err := p.Validate(c, hash); !errors.Is(err, ErrPlan) {
		t.Fatalf("duplicate wire bytes accepted: %v", err)
	}
	p, err = BuildPilotPlan(f, researchv2.ProfileC, EqualTxRate, "constant", "pilot-safe", 17, c, hash, 2)
	if err != nil {
		t.Fatal(err)
	}
	p.Events[0].Source = "p2p"
	if err := p.Validate(c, hash); !errors.Is(err, ErrPlan) {
		t.Fatalf("unimplemented direct P2P submission accepted: %v", err)
	}
	c.Safety.MaxDurationSeconds = 601
	if err := c.Validate(); !errors.Is(err, ErrConfig) {
		t.Fatal("hard duration ceiling ignored")
	}
}

type blockedSender struct{}

func (blockedSender) Send(ctx context.Context, _ []byte) SendResult {
	<-ctx.Done()
	return SendResult{Completed: time.Now(), Err: ctx.Err()}
}

func TestS06OpenLoopQueueMissIsFailure(t *testing.T) {
	c, hash := pilotConfig(t)
	f, err := NewPilotFixture("pluto-s06-queue", 12)
	if err != nil {
		t.Fatal(err)
	}
	p, err := BuildPilotPlan(f, researchv2.ProfileC, EqualTxRate, "constant", "pilot-queue-miss", 8, c, hash, 2)
	if err != nil {
		t.Fatal(err)
	}
	p.WarmupNS, p.SteadyNS, p.DrainNS, p.TargetTxPS = 0, int64(time.Second), 0, 3
	p.Events = p.Events[:3]
	for i := range p.Events {
		p.Events[i].OffsetNS = int64(i) * int64(time.Millisecond)
	}
	c.Safety.MaxOutstanding = 1
	if err := p.Validate(c, hash); err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), p, c, hash, strings.Repeat("a", 64), blockedSender{}, GuardFunc(func(context.Context) error { return nil }))
	if err != nil || result.Status != StatusRateMiss || result.ActualSendCount != 0 {
		t.Fatalf("full queue silently throttled: result=%+v err=%v", result, err)
	}
}

func TestS06RuntimeGuardAbortsOutstandingLoad(t *testing.T) {
	c, hash := pilotConfig(t)
	f, err := NewPilotFixture("pluto-s06-abort", 12)
	if err != nil {
		t.Fatal(err)
	}
	p, err := BuildPilotPlan(f, researchv2.ProfileC, EqualTxRate, "constant", "pilot-safety-abort", 9, c, hash, 2)
	if err != nil {
		t.Fatal(err)
	}
	c.Network.CollectorIntervalMS = 100
	var checks atomic.Int32
	guard := GuardFunc(func(context.Context) error {
		if checks.Add(1) > 1 {
			return fmt.Errorf("%w: injected observer failure", ErrSafety)
		}
		return nil
	})
	result, err := Run(context.Background(), p, c, hash, strings.Repeat("b", 64), blockedSender{}, guard)
	if err != nil || result.Status != StatusSafetyAbort || !strings.Contains(result.Reason, "observer failure") || checks.Load() < 2 {
		t.Fatalf("runtime safety failure did not abort: status=%s reason=%s checks=%d err=%v", result.Status, result.Reason, checks.Load(), err)
	}
}

func TestS06KillSwitchAndPublicTargetRefused(t *testing.T) {
	c, _ := pilotConfig(t)
	if _, err := NewHTTPSender("http://example.com:26657", time.Second); err == nil {
		t.Fatal("public RPC endpoint accepted")
	}
	if _, err := NewHTTPSender("http://localhost:26657", time.Second); err == nil {
		t.Fatal("DNS hostname accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com/", http.StatusFound)
	}))
	defer server.Close()
	sender, err := NewHTTPSender(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result := sender.Send(context.Background(), []byte{1}); result.Err == nil {
		t.Fatal("HTTP redirect left the private endpoint")
	}
	root := t.TempDir()
	kill := filepath.Join(root, "STOP")
	if err := os.WriteFile(kill, []byte("stop\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var nodes [4]WatchedNode
	pidBase := os.Getpid() + 1000000
	for i := range nodes {
		nodes[i] = WatchedNode{RPC: fmt.Sprintf("http://127.0.0.1:%d", 22000+i), ObserverFile: fmt.Sprintf("%s/observer-%d.jsonl", root, i), ObserverPID: pidBase + i, CometPID: pidBase + 100 + i, AppPID: pidBase + 200}
	}
	guard, err := NewHostGuard(c, nodes, root, kill, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := guard.Check(context.Background()); !errors.Is(err, ErrSafety) || !strings.Contains(err.Error(), "kill switch") {
		t.Fatalf("manual kill switch ignored: %v", err)
	}
}

func TestFile02MempoolLimitReasonClassification(t *testing.T) {
	for _, tc := range []struct {
		message string
		next    int
		want    string
	}{
		{"mempool is full: number of txs 1 (max: 1), total txs bytes 2568 (max: 8388608)", 2568, "mempool_count_full"},
		{"mempool is full: number of txs 1 (max: 128), total txs bytes 2568 (max: 2700)", 2568, "mempool_bytes_full"},
		{"mempool is full: number of txs 1 (max: 1), total txs bytes 2568 (max: 2700)", 2568, "mempool_count_and_bytes_full"},
		{"some other RPC error", 1, "rpc_reject"},
		{"Tx too large. Max size is 8192, but got 8193", 8193, "mempool_tx_too_large"},
	} {
		if got := classifyRPCAdmission(tc.message, tc.next); got != tc.want {
			t.Fatalf("%q classified %q, want %q", tc.message, got, tc.want)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error","data":"mempool is full: number of txs 1 (max: 1), total txs bytes 2568 (max: 8388608)"}}`))
	}))
	defer server.Close()
	sender, err := NewHTTPSender(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got := sender.Send(context.Background(), []byte{1})
	if got.Err == nil || got.Reason != "mempool_count_full" {
		t.Fatalf("unexpected bounded RPC classification: %+v", got)
	}
}

func TestS06PublicCorpusOracleRejectsMislabel(t *testing.T) {
	c, err := ReadScenarioCorpus(filepath.Join("..", "..", "..", "research", "corpus", "s06_scenarios.json"))
	if err != nil {
		t.Fatal(err)
	}
	var sawLateML bool
	for i, item := range c.Cases {
		if item.Scenario == "invalid_ml_after_ecdsa" && item.Profile == researchv2.ProfileH && item.ExpectedStage == researchv2.StageML {
			sawLateML = true
		}
		if item.Scenario == "late_state_invalid" {
			c.Cases[i].Capability = "A_arbitrary_sender"
			if err := VerifyScenarioCorpus(c); !errors.Is(err, ErrCorpus) {
				t.Fatalf("wrong capability accepted: %v", err)
			}
			c.Cases[i].Capability = item.Capability
			break
		}
	}
	if !sawLateML {
		// The corpus order puts H after C; inspect all cases explicitly.
		for _, item := range c.Cases {
			if item.Scenario == "invalid_ml_after_ecdsa" && item.Profile == researchv2.ProfileH && item.ExpectedStage == researchv2.StageML {
				sawLateML = true
			}
		}
	}
	if !sawLateML {
		t.Fatal("H late bad-ML vector was not proven after secp")
	}
}
