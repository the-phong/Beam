//go:build integration

package node

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	b0state "example.com/pluto/internal/classical/state"
	b0store "example.com/pluto/internal/classical/store"
)

func d026ReviewFixture(t *testing.T) (string, []byte) {
	t.Helper()
	comet, err := filepath.Abs("../../build/cometbft")
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := hex.DecodeString(d026GenesisHex)
	if err != nil {
		t.Fatal(err)
	}
	return comet, genesis
}

func TestD026ConcurrentInitClaimsHomeOnce(t *testing.T) {
	comet, genesis := d026ReviewFixture(t)
	home := filepath.Join(t.TempDir(), "nested", "home")
	arrived := make(chan struct{}, 2)
	release := make(chan struct{})
	var claims atomic.Int32
	hooks := &b0InitHooks{
		BeforeHome: func() error { arrived <- struct{}{}; <-release; return nil },
		AfterHome:  func() error { claims.Add(1); return nil },
	}
	results := make(chan error, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer workers.Done()
			results <- initB0WithHooks(home, comet, "pluto-b0-v1", genesis, 26657, 26656, 26658, hooks, io.Discard)
		}()
	}
	<-arrived
	<-arrived
	close(release)
	workers.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case strings.Contains(err.Error(), "node home already exists"):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent initialization error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 || claims.Load() != 1 {
		t.Fatalf("exclusive claim failed: successes=%d conflicts=%d after-home claims=%d", successes, conflicts, claims.Load())
	}
	paths := []string{
		"config/config.toml", "config/genesis.json", "config/pluto.json",
		"config/priv_validator_key.json", "config/node_key.json", "data/priv_validator_state.json",
	}
	before := make(map[string][sha256.Size]byte, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(home, path))
		if err != nil {
			t.Fatal(err)
		}
		before[path] = sha256.Sum256(data)
	}
	state, err := b0store.ProbeExisting(filepath.Join(home, "data", "pluto-b0-v2"), "pluto-b0-v1")
	if err != nil || state.Height() != 0 || state.Accounts()[0].Balance != 100 || state.Accounts()[0].Nonce != 0 {
		t.Fatalf("winner's H0 incomplete: state=%+v err=%v", state, err)
	}
	if err := InitB0(home, comet, "pluto-b0-v1", genesis, 26657, 26656, 26658, io.Discard); err == nil || !strings.Contains(err.Error(), "node home already exists") {
		t.Fatalf("repeat initialization did not refuse existing identity: %v", err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(home, path))
		if err != nil || sha256.Sum256(data) != before[path] {
			t.Fatalf("concurrent or repeated initialization changed %s: %v", path, err)
		}
	}
	hash := state.AppHash()
	t.Logf("both callers reached BeforeHome; one exclusive AfterHome claim; successes=1 conflicts=1 B=0 C=0 A=0 H=%x sender=100/0; six configuration/identity/state artifact SHA-256 values unchanged after repeat refusal", hash)
}

func TestD026InitRejectsCorruptCometArtifacts(t *testing.T) {
	comet, genesis := d026ReviewFixture(t)
	for _, tc := range []struct {
		name, file, want string
		mutate           func([]byte) ([]byte, error)
	}{
		{"config-syntax", "config/config.toml", "read CometBFT configuration", func([]byte) ([]byte, error) { return []byte("[rpc\n"), nil }},
		{"config-abci", "config/config.toml", "conflicts with initialized B0 addresses", func(raw []byte) ([]byte, error) {
			from := []byte(`proxy_app = "tcp://127.0.0.1:26658"`)
			if !bytes.Contains(raw, from) {
				return nil, errors.New("ABCI address fixture not found")
			}
			return bytes.Replace(raw, from, []byte(`proxy_app = "tcp://127.0.0.1:29999"`), 1), nil
		}},
		{"config-rpc", "config/config.toml", "conflicts with initialized B0 addresses", func(raw []byte) ([]byte, error) {
			from := []byte(`laddr = "tcp://127.0.0.1:26657"`)
			if !bytes.Contains(raw, from) {
				return nil, errors.New("RPC address fixture not found")
			}
			return bytes.Replace(raw, from, []byte(`laddr = "tcp://127.0.0.1:29998"`), 1), nil
		}},
		{"config-p2p", "config/config.toml", "conflicts with initialized B0 addresses", func(raw []byte) ([]byte, error) {
			from := []byte(`laddr = "tcp://127.0.0.1:26656"`)
			if !bytes.Contains(raw, from) {
				return nil, errors.New("P2P address fixture not found")
			}
			return bytes.Replace(raw, from, []byte(`laddr = "tcp://127.0.0.1:29997"`), 1), nil
		}},
		{"validator-key-json", "config/priv_validator_key.json", "decode CometBFT validator key", func([]byte) ([]byte, error) { return []byte("{bad"), nil }},
		{"validator-key-binding", "config/priv_validator_key.json", "validator key conflicts with genesis", func(raw []byte) ([]byte, error) {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(raw, &fields); err != nil {
				return nil, err
			}
			fields["address"] = json.RawMessage(`"0000000000000000000000000000000000000000"`)
			return json.Marshal(fields)
		}},
		{"validator-state-incomplete", "data/priv_validator_state.json", "missing height", func([]byte) ([]byte, error) { return []byte("{}"), nil }},
		{"validator-state-stale", "data/priv_validator_state.json", "not fresh", func([]byte) ([]byte, error) { return []byte(`{"height":"2","round":0,"step":0}`), nil }},
		{"node-key-json", "config/node_key.json", "load CometBFT node key", func([]byte) ([]byte, error) { return []byte("{bad"), nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "home")
			reached := false
			hooks := &b0InitHooks{AfterMetadata: func() error {
				reached = true
				path := filepath.Join(home, tc.file)
				raw, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				corrupt, err := tc.mutate(raw)
				if err != nil {
					return err
				}
				return os.WriteFile(path, corrupt, 0600)
			}}
			err := initB0WithHooks(home, comet, "pluto-b0-v1", genesis, 26657, 26656, 26658, hooks, io.Discard)
			if !reached || err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("corruption not rejected at verification: reached=%t err=%v want=%q", reached, err, tc.want)
			}
			state, probeErr := b0store.ProbeExisting(filepath.Join(home, "data", "pluto-b0-v2"), "pluto-b0-v1")
			expected, parseErr := b0state.ParseGenesis(genesis, "pluto-b0-v1")
			if probeErr != nil || parseErr != nil || state.AppHash() != expected.AppHash() {
				t.Fatalf("failed initialization altered H0: probe=%v parse=%v", probeErr, parseErr)
			}
			if retryErr := InitB0(home, comet, "pluto-b0-v1", genesis, 26657, 26656, 26658, io.Discard); retryErr == nil || !strings.Contains(retryErr.Error(), "node home already exists") {
				t.Fatalf("incomplete home was reinitialized: %v", retryErr)
			}
			hash := state.AppHash()
			t.Logf("AfterMetadata corruption=%s reached=true init_error=%q B=0 C=0 A=0 H=%x sender=100/0; existing home retained; repeat init refused", tc.name, err, hash)
		})
	}
}
