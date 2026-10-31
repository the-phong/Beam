//go:build integration && fourval

package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	cmttypes "github.com/cometbft/cometbft/types"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/b1app"
	"example.com/pluto/internal/b1genesis"
	"example.com/pluto/internal/b1store"
	"example.com/pluto/internal/b1test"
	"example.com/pluto/internal/b1tx"
	"example.com/pluto/internal/node"
)

func newHybridNetwork(t *testing.T, fixture *b1test.Fixture) *fourNetwork {
	t.Helper()
	n := &fourNetwork{t: t, root: filepath.Join(t.TempDir(), "network"), genesis: fixture.Genesis}
	n.plutod, _ = filepath.Abs("../../build/plutod")
	comet, _ := filepath.Abs("../../build/cometbft")
	used := map[int]bool{}
	var ports [4]node.B0Ports
	for i := range n.nodes {
		for _, ptr := range []*int{&ports[i].RPC, &ports[i].P2P, &ports[i].ABCI} {
			for {
				port := freePort(t)
				if !used[port] {
					*ptr, used[port] = port, true
					break
				}
			}
		}
		n.nodes[i].ports = ports[i]
		n.nodes[i].home = filepath.Join(n.root, fmt.Sprintf("validator-%d", i))
	}
	if err := node.InitB1FourValidators(n.root, comet, fixture.ChainID, fixture.Genesis, ports, io.Discard); err != nil {
		t.Fatalf("B1 network init: %v", err)
	}
	t.Cleanup(func() {
		if dir := os.Getenv("PLUTO_4V_ARTIFACTS"); dir != "" {
			_ = os.MkdirAll(dir, 0700)
			if data, err := os.ReadFile(filepath.Join(n.nodes[0].home, "config", "genesis.json")); err == nil {
				_ = os.WriteFile(filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "_")+"-public-genesis.json"), data, 0600)
			}
		}
	})
	t.Cleanup(n.saveLogs)
	t.Cleanup(n.stopAll)
	var shared []byte
	seen := map[string]bool{}
	seenNodes := map[string]bool{}
	for i := range n.nodes {
		home := n.nodes[i].home
		raw, err := os.ReadFile(filepath.Join(home, "config", "genesis.json"))
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			shared = raw
		} else if !bytes.Equal(shared, raw) {
			t.Fatal("B1 shared genesis differs")
		}
		doc, err := cmttypes.GenesisDocFromFile(filepath.Join(home, "config", "genesis.json"))
		if err != nil || doc.ChainID != fixture.ChainID || len(doc.Validators) != 4 {
			t.Fatalf("B1 validator %d genesis: %v", i, err)
		}
		keyRaw, err := os.ReadFile(filepath.Join(home, "config", "priv_validator_key.json"))
		if err != nil {
			t.Fatal(err)
		}
		var key privval.FilePVKey
		if err := cmtjson.Unmarshal(keyRaw, &key); err != nil {
			t.Fatal(err)
		}
		if seen[string(key.Address)] || !bytes.Equal(key.Address, doc.Validators[i].Address) || doc.Validators[i].Power != 10 {
			t.Fatalf("B1 validator %d identity duplicate or mismatch", i)
		}
		seen[string(key.Address)] = true
		nodeKey, err := p2p.LoadNodeKey(filepath.Join(home, "config", "node_key.json"))
		if err != nil || nodeKey == nil {
			t.Fatalf("B1 validator %d P2P identity: %v", i, err)
		}
		if seenNodes[string(nodeKey.ID())] {
			t.Fatalf("B1 validator %d duplicated P2P identity", i)
		}
		seenNodes[string(nodeKey.ID())] = true
		state, err := b1store.ProbeExisting(filepath.Join(home, "data", "pluto-b1-v1"), fixture.ChainID, fixture.Genesis)
		if err != nil || state.Height() != 0 || state.TotalSupply() != 100 {
			t.Fatalf("B1 H0 validator %d: %v", i, err)
		}
	}
	if err := node.InitB1FourValidators(n.root, comet, fixture.ChainID, fixture.Genesis, ports, io.Discard); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("repeated B1 init accepted: %v", err)
	}
	t.Log("H22: four unique validator and P2P identities, shared canonical B1 genesis, four H0 snapshots, repeat init refused")
	return n
}

func TestB1HybridInitializationRefusals(t *testing.T) {
	for _, mode := range []string{"missing-h0", "corrupt-h0", "genesis-mismatch", "validator-key-mismatch", "missing-validator-binding", "foreign-b0-store"} {
		t.Run(mode, func(t *testing.T) {
			f, err := b1test.New("pluto-b1-refusal")
			if err != nil {
				t.Fatal(err)
			}
			n := newHybridNetwork(t, f)
			home := n.nodes[0].home
			switch mode {
			case "missing-h0":
				path := filepath.Join(home, "data", "pluto-b1-v1")
				if err := os.Rename(path, path+".held"); err != nil {
					t.Fatal(err)
				}
			case "corrupt-h0":
				db, err := pebble.Open(filepath.Join(home, "data", "pluto-b1-v1"), &pebble.Options{})
				if err != nil {
					t.Fatal(err)
				}
				if err := db.Set([]byte("pluto/committed-state/b1-v1"), []byte("corrupt"), pebble.Sync); err != nil {
					t.Fatal(err)
				}
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
			case "genesis-mismatch":
				other, err := b1test.New(f.ChainID)
				if err != nil {
					t.Fatal(err)
				}
				transport, err := b1genesis.Encode(other.Genesis, f.ChainID)
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(home, "config", "genesis.json")
				doc, err := cmttypes.GenesisDocFromFile(path)
				if err != nil {
					t.Fatal(err)
				}
				doc.AppState = transport
				if err := doc.SaveAs(path); err != nil {
					t.Fatal(err)
				}
			case "validator-key-mismatch":
				path := filepath.Join(home, "config", "priv_validator_key.json")
				other, err := os.ReadFile(filepath.Join(n.nodes[1].home, "config", "priv_validator_key.json"))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, other, 0600); err != nil {
					t.Fatal(err)
				}
			case "missing-validator-binding":
				path := filepath.Join(home, "config", "pluto.json")
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(raw, &fields); err != nil {
					t.Fatal(err)
				}
				delete(fields, "validator_address")
				raw, err = json.Marshal(fields)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
			case "foreign-b0-store":
				if err := os.Mkdir(filepath.Join(home, "data", "pluto-b0-v2"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command(n.plutod, "start", "--home", home)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() { result <- cmd.Wait() }()
			var startErr error
			select {
			case startErr = <-result:
			case <-time.After(5 * time.Second):
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				<-result
				t.Fatalf("B1 %s did not refuse startup within five seconds", mode)
			}
			if startErr == nil {
				t.Fatal("conflicting B1 home started")
			}
			if output.Len() == 0 {
				t.Fatalf("B1 refusal %s had no diagnostic", mode)
			}
			t.Logf("B1 %s refused: %s", mode, output.String())
		})
	}
}

func TestB1HybridCLIInitialization(t *testing.T) {
	raw, err := os.ReadFile("../../internal/b1tx/testdata/hybrid_public_vector.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector struct {
		ChainID string `json:"chain_id"`
		Genesis string `json:"genesis_hex"`
	}
	if err := json.Unmarshal(raw, &vector); err != nil {
		t.Fatal(err)
	}
	genesis, err := hex.DecodeString(vector.Genesis)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "fresh-b1-root")
	genesisFile := filepath.Join(t.TempDir(), "genesis-v3.bin")
	if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
		t.Fatal(err)
	}
	ports := [3]int{}
	selected := false
	for attempt := 0; attempt < 30 && !selected; attempt++ {
		for i := range ports {
			ports[i] = freePort(t)
		}
		all := map[int]bool{}
		selected = true
		for _, base := range ports {
			for i := 0; i < 4; i++ {
				p := base + 10*i
				if p > 65535 || all[p] {
					selected = false
				}
				all[p] = true
			}
		}
	}
	if !selected {
		t.Fatal("could not allocate distinct B1 CLI port ranges")
	}
	plutod, _ := filepath.Abs("../../build/plutod")
	args := []string{"init-b1-4v", "--root", root, "--chain-id", vector.ChainID, "--genesis-file", genesisFile, "--rpc-base-port", strconv.Itoa(ports[0]), "--p2p-base-port", strconv.Itoa(ports[1]), "--abci-base-port", strconv.Itoa(ports[2])}
	output, err := exec.Command(plutod, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("B1 CLI init: %v %s", err, output)
	}
	for i := 0; i < 4; i++ {
		home := filepath.Join(root, fmt.Sprintf("validator-%d", i))
		config, err := node.ReadConfig(home)
		if err != nil || config.Profile != node.ProfileB1 || config.ChainID != vector.ChainID {
			t.Fatalf("B1 CLI home %d metadata: %+v err=%v", i, config, err)
		}
		state, err := b1store.ProbeExisting(filepath.Join(home, "data", "pluto-b1-v1"), vector.ChainID, genesis)
		if err != nil || state.Height() != 0 {
			t.Fatalf("B1 CLI home %d H0: %v", i, err)
		}
	}
	if output, err := exec.Command(plutod, args...).CombinedOutput(); err == nil || !bytes.Contains(output, []byte("already exists")) {
		t.Fatalf("B1 CLI duplicate init accepted: %v %s", err, output)
	}
	t.Log("B1 CLI init-b1-4v: four H0 homes and shared Genesis V3; repeated initialization refused")
}

func hybridAccount(n *fourNetwork, i int, address b0state.Address) (uint64, uint64, error) {
	var response struct {
		Result struct {
			Response struct {
				Code  uint32 `json:"code"`
				Value string `json:"value"`
			} `json:"response"`
		} `json:"result"`
	}
	path := "/abci_query?path=%22b1%2Faccount%2Fv1%22&data=0x" + hex.EncodeToString(address[:])
	if err := rpcGetLong(n.t, n.nodes[i].ports.RPC, path, &response); err != nil {
		return 0, 0, err
	}
	value, err := base64.StdEncoding.DecodeString(response.Result.Response.Value)
	if err != nil || response.Result.Response.Code != 0 || len(value) != 17 || value[0] != 1 {
		return 0, 0, fmt.Errorf("B1 account response code=%d len=%d err=%v", response.Result.Response.Code, len(value), err)
	}
	return binary.BigEndian.Uint64(value[1:9]), binary.BigEndian.Uint64(value[9:17]), nil
}

func assertHybridState(t *testing.T, n *fourNetwork, f *b1test.Fixture, height int64) {
	t.Helper()
	n.waitAll(height+1, 90*time.Second)
	var wantBlock, wantApp string
	for i := range n.nodes {
		sb, sn, err := hybridAccount(n, i, f.Sender.Binding.Address)
		if err != nil || sb != 75 || sn != 1 {
			t.Fatalf("B1 validator %d sender=%d/%d err=%v", i, sb, sn, err)
		}
		rb, rn, err := hybridAccount(n, i, f.Receiver.Binding.Address)
		if err != nil || rb != 25 || rn != 0 {
			t.Fatalf("B1 validator %d receiver=%d/%d err=%v", i, rb, rn, err)
		}
		block, app, _, err := n.block(i, height+1)
		if err != nil || block == "" || app == "" {
			t.Fatalf("B1 validator %d block H%d: %v", i, height+1, err)
		}
		if i == 0 {
			wantBlock, wantApp = block, app
		} else if block != wantBlock || app != wantApp {
			t.Fatalf("B1 validator %d same-height divergence H%d block=%s app=%s want=%s/%s", i, height+1, block, app, wantBlock, wantApp)
		}
	}
	t.Logf("H01/H22 matching block H%d hash=%s AppHash(A=%d)=%s sender=75/1 receiver=25/0 supply=100", height+1, wantBlock, height, wantApp)
}

func TestB1HybridFourValidator(t *testing.T) {
	seenAcrossRuns := map[string]bool{}
	for run := 0; run < 3; run++ {
		t.Run(fmt.Sprintf("fresh-network-%d", run), func(t *testing.T) {
			f, err := b1test.New(fmt.Sprintf("pluto-b1-%d", run))
			if err != nil {
				t.Fatal(err)
			}
			n := newHybridNetwork(t, f)
			for i := range n.nodes {
				raw, err := os.ReadFile(filepath.Join(n.nodes[i].home, "config", "priv_validator_key.json"))
				if err != nil {
					t.Fatal(err)
				}
				var key privval.FilePVKey
				if err := cmtjson.Unmarshal(raw, &key); err != nil {
					t.Fatal(err)
				}
				if seenAcrossRuns[string(key.Address)] {
					t.Fatal("validator signing identity repeated across fresh B1 networks")
				}
				seenAcrossRuns[string(key.Address)] = true
			}
			n.startAll()
			n.waitAll(2, 90*time.Second)
			initial, _, err := n.height(0)
			if err != nil {
				t.Fatal(err)
			}
			// A valid hybrid transfer must finalize with exactly three validators alive.
			n.stop(3)
			n.waitSet([]int{0, 1, 2}, initial+3, 60*time.Second)
			wire, err := f.Signed(25, 0)
			if err != nil {
				t.Fatal(err)
			}
			bad := bytes.Clone(wire)
			bad[len(bad)-1] ^= 1
			rejected, err := n.broadcast(bad)
			if err != nil || rejected.Result.CheckTx.Code != uint32(b1tx.CodeBadMLSignature) {
				t.Fatalf("H05 B1 admission code=%d err=%v", rejected.Result.CheckTx.Code, err)
			}
			accepted, err := n.broadcast(wire)
			if err != nil || accepted.Result.CheckTx.Code != 0 || accepted.Result.TxResult.Code != 0 {
				t.Fatalf("H01 B1 broadcast: %+v err=%v", accepted, err)
			}
			height, err := strconv.ParseInt(accepted.Result.Height, 10, 64)
			if err != nil || height <= initial {
				t.Fatalf("B1 transaction height=%q err=%v", accepted.Result.Height, err)
			}
			n.waitSet([]int{0, 1, 2}, height+1, 90*time.Second)
			n.start(3)
			assertHybridState(t, n, f, height)
			// Restart a validator that already committed the transfer. Comet's
			// handshake must use its durable B1 state without applying it twice.
			n.stop(0)
			n.start(0)
			assertHybridState(t, n, f, height)
			for i := range n.nodes {
				_, _, txs, err := n.block(i, height)
				if err != nil || len(txs) != 1 || txs[0] != base64.StdEncoding.EncodeToString(wire) {
					t.Fatalf("B1 validator %d committed ordered tx H%d: %v", i, height, err)
				}
			}
			t.Logf("H01/H21/H22 run=%d valid hybrid transfer H=%d with 3/4 validators; stopped validator caught up, exact bytes=%x", run, height, wire)
			n.stopAll()
			candidate, err := f.Signed(10, 1)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("H19/H20 direct proposal raw_bad=%x raw_valid_next=%x", bad, candidate)
			for i := range n.nodes {
				app, err := b1app.Open(filepath.Join(n.nodes[i].home, "data", "pluto-b1-v1"), f.ChainID, f.Genesis)
				if err != nil {
					t.Fatal(err)
				}
				state, err := app.Committed()
				if err != nil {
					t.Fatal(err)
				}
				if state.TotalSupply() != 100 {
					t.Fatalf("B1 supply on validator %d=%d", i, state.TotalSupply())
				}
				for _, proposal := range [][][]byte{{bad}, {wire}, {candidate, bad}, {candidate, candidate}} {
					resp, err := app.ProcessProposal(context.Background(), &abci.RequestProcessProposal{Height: state.Height() + 1, Txs: proposal})
					if err != nil || resp.Status != abci.ResponseProcessProposal_REJECT {
						t.Fatalf("H19/H20 validator %d proposal accepted err=%v resp=%+v", i, err, resp)
					}
				}
				after, err := app.Committed()
				if err != nil || after.AppHash() != state.AppHash() {
					t.Fatalf("H19/H20 validator %d invalid proposal changed committed state: %v", i, err)
				}
				if err := app.Close(); err != nil {
					t.Fatal(err)
				}
			}
			t.Log("H19/H20: all four independent persisted applications reject invalid direct ProcessProposal candidates")
		})
	}
}
