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
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/cometbft/cometbft/abci/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/privval"
	cmttypes "github.com/cometbft/cometbft/types"

	b0app "example.com/pluto/internal/classical/app"
	b0state "example.com/pluto/internal/classical/state"
	b0store "example.com/pluto/internal/classical/store"
	"example.com/pluto/internal/node"
)

type fourNode struct {
	home  string
	ports node.B0Ports
	cmd   *exec.Cmd
	done  chan error
	pid   int
	log   *os.File
	logs  []string
}

type fourNetwork struct {
	t       *testing.T
	root    string
	genesis []byte
	nodes   [4]fourNode
	plutod  string
}

func newFourNetwork(t *testing.T) *fourNetwork {
	t.Helper()
	var result fourNetwork
	result.t = t
	result.root = filepath.Join(t.TempDir(), "network")
	result.plutod, _ = filepath.Abs("../../build/plutod")
	comet, _ := filepath.Abs("../../build/cometbft")
	var err error
	result.genesis, err = hex.DecodeString(b0GoldenGenesis)
	if err != nil {
		t.Fatal(err)
	}
	used := map[int]bool{}
	for i := range result.nodes {
		ports := []*int{&result.nodes[i].ports.RPC, &result.nodes[i].ports.P2P, &result.nodes[i].ports.ABCI}
		for _, ptr := range ports {
			for {
				port := freePort(t)
				if !used[port] {
					*ptr = port
					used[port] = true
					break
				}
			}
		}
		result.nodes[i].home = filepath.Join(result.root, fmt.Sprintf("validator-%d", i))
	}
	var ports [4]node.B0Ports
	for i := range result.nodes {
		ports[i] = result.nodes[i].ports
	}
	if err := node.InitB0FourValidators(result.root, comet, "pluto-b0-v1", result.genesis, ports, io.Discard); err != nil {
		t.Fatalf("four-validator H0 initialization: %v", err)
	}
	t.Cleanup(result.saveLogs)
	t.Cleanup(result.stopAll)
	result.verifyGenesis()
	return &result
}

func (n *fourNetwork) verifyGenesis() {
	n.t.Helper()
	var first []byte
	seen := map[string]bool{}
	for i := range n.nodes {
		home := n.nodes[i].home
		raw, err := os.ReadFile(filepath.Join(home, "config", "genesis.json"))
		if err != nil {
			n.t.Fatal(err)
		}
		if i == 0 {
			first = raw
		} else if !bytes.Equal(first, raw) {
			n.t.Fatal("genesis bytes differ among validators")
		}
		doc, err := cmttypes.GenesisDocFromFile(filepath.Join(home, "config", "genesis.json"))
		if err != nil || doc.ChainID != "pluto-b0-v1" || len(doc.Validators) != 4 {
			n.t.Fatalf("validator %d genesis: %v", i, err)
		}
		keyBytes, err := os.ReadFile(filepath.Join(home, "config", "priv_validator_key.json"))
		if err != nil {
			n.t.Fatal(err)
		}
		var key privval.FilePVKey
		if err := cmtjson.Unmarshal(keyBytes, &key); err != nil {
			n.t.Fatal(err)
		}
		if seen[string(key.Address)] {
			n.t.Fatal("duplicate validator signing identity")
		}
		seen[string(key.Address)] = true
		if !bytes.Equal(key.Address, doc.Validators[i].Address) || doc.Validators[i].Power != 10 {
			n.t.Fatalf("validator %d key/genesis binding failed", i)
		}
		state, err := b0store.ProbeExisting(filepath.Join(home, "data", "pluto-b0-v2"), "pluto-b0-v1")
		if err != nil || state.Height() != 0 || state.Accounts()[0].Balance != 100 || state.Accounts()[0].Nonce != 0 {
			n.t.Fatalf("validator %d H0 mismatch: %v", i, err)
		}
	}
	if err := node.InitB0FourValidators(n.root, "unused", "pluto-b0-v1", n.genesis, [4]node.B0Ports{n.nodes[0].ports, n.nodes[1].ports, n.nodes[2].ports, n.nodes[3].ports}, io.Discard); err == nil || !strings.Contains(err.Error(), "already exists") {
		n.t.Fatalf("repeated network initialization did not refuse existing root: %v", err)
	}
	n.t.Log("T01/T14: four unique signing identities, one byte-identical shared genesis, four verified H0 snapshots; repeated init refused")
}

func (n *fourNetwork) start(i int) {
	n.t.Helper()
	item := &n.nodes[i]
	if item.cmd != nil {
		n.t.Fatalf("validator %d already running", i)
	}
	path := filepath.Join(filepath.Dir(n.root), fmt.Sprintf("validator-%d-run-%d.log", i, len(item.logs)))
	f, err := os.Create(path)
	if err != nil {
		n.t.Fatal(err)
	}
	cmd := exec.Command(n.plutod, "start", "--home", item.home)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout, cmd.Stderr = f, f
	if err := cmd.Start(); err != nil {
		_ = f.Close()
		n.t.Fatal(err)
	}
	item.cmd, item.pid, item.log, item.done = cmd, cmd.Process.Pid, f, make(chan error, 1)
	item.logs = append(item.logs, path)
	go func() { item.done <- cmd.Wait() }()
}

func (n *fourNetwork) stop(i int) {
	n.t.Helper()
	item := &n.nodes[i]
	if item.cmd == nil {
		return
	}
	_ = item.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-item.done:
	case <-time.After(15 * time.Second):
		_ = syscall.Kill(-item.pid, syscall.SIGKILL)
		<-item.done
		n.t.Errorf("validator %d graceful stop timed out", i)
	}
	_ = item.log.Close()
	item.cmd = nil
}

func (n *fourNetwork) stopAll() {
	for i := range n.nodes {
		n.stop(i)
	}
}

func (n *fourNetwork) saveLogs() {
	dir := os.Getenv("PLUTO_4V_ARTIFACTS")
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0700)
	for i := range n.nodes {
		for j, path := range n.nodes[i].logs {
			data, err := os.ReadFile(path)
			if err == nil {
				_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s-validator-%d-run-%d.log", strings.ReplaceAll(n.t.Name(), "/", "_"), i, j)), data, 0600)
			}
		}
	}
}

func (n *fourNetwork) startAll() {
	for i := range n.nodes {
		n.start(i)
	}
}

func (n *fourNetwork) height(i int) (int64, string, error) {
	var status statusResponse
	err := rpcGet(n.t, n.nodes[i].ports.RPC, "/status", &status)
	if err != nil {
		return 0, "", err
	}
	h, err := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64)
	return h, status.Result.SyncInfo.LatestAppHash, err
}

func (n *fourNetwork) waitAll(minimum int64, timeout time.Duration) {
	n.waitSet([]int{0, 1, 2, 3}, minimum, timeout)
}

func (n *fourNetwork) waitSet(indices []int, minimum int64, timeout time.Duration) {
	n.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ready := true
		for _, i := range indices {
			h, _, err := n.height(i)
			if err != nil || h < minimum {
				ready = false
				break
			}
		}
		if ready {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	for _, i := range indices {
		h, hash, err := n.height(i)
		n.t.Logf("validator %d height=%d apphash=%s err=%v log=%v", i, h, hash, err, n.nodes[i].logs)
	}
	n.t.Fatalf("validators %v did not all reach height %d", indices, minimum)
}

func (n *fourNetwork) account(i int, address b0state.Address) (uint64, uint64, error) {
	var response struct {
		Result struct {
			Response struct {
				Code  uint32 `json:"code"`
				Value string `json:"value"`
			} `json:"response"`
		} `json:"result"`
	}
	path := "/abci_query?path=%22b0%2Faccount%2Fv1%22&data=0x" + hex.EncodeToString(address[:])
	if err := rpcGetLong(n.t, n.nodes[i].ports.RPC, path, &response); err != nil {
		return 0, 0, err
	}
	value, err := base64.StdEncoding.DecodeString(response.Result.Response.Value)
	if err != nil || response.Result.Response.Code != 0 || len(value) != 17 || value[0] != 1 {
		return 0, 0, fmt.Errorf("invalid account response code=%d len=%d err=%v", response.Result.Response.Code, len(value), err)
	}
	return binary.BigEndian.Uint64(value[1:9]), binary.BigEndian.Uint64(value[9:17]), nil
}

func (n *fourNetwork) broadcast(raw []byte) (broadcastResult, error) {
	var response broadcastResult
	err := rpcGetLong(n.t, n.nodes[0].ports.RPC, "/broadcast_tx_commit?tx=0x"+hex.EncodeToString(raw), &response)
	if err != nil {
		return response, err
	}
	if response.Error != nil {
		return response, fmt.Errorf("RPC error: %v", response.Error)
	}
	return response, nil
}

func (n *fourNetwork) block(i int, height int64) (string, string, []string, error) {
	var response struct {
		Result struct {
			BlockID struct {
				Hash string `json:"hash"`
			} `json:"block_id"`
			Block struct {
				Header struct {
					AppHash string `json:"app_hash"`
				} `json:"header"`
				Data struct {
					Txs []string `json:"txs"`
				} `json:"data"`
			} `json:"block"`
		} `json:"result"`
	}
	err := rpcGetLong(n.t, n.nodes[i].ports.RPC, fmt.Sprintf("/block?height=%d", height), &response)
	return response.Result.BlockID.Hash, response.Result.Block.Header.AppHash, response.Result.Block.Data.Txs, err
}

func (n *fourNetwork) commitVoters(i int, height int64) ([]string, error) {
	var response struct {
		Result struct {
			SignedHeader struct {
				Commit struct {
					Signatures []struct {
						BlockIDFlag      uint8  `json:"block_id_flag"`
						ValidatorAddress string `json:"validator_address"`
					} `json:"signatures"`
				} `json:"commit"`
			} `json:"signed_header"`
		} `json:"result"`
	}
	if err := rpcGetLong(n.t, n.nodes[i].ports.RPC, fmt.Sprintf("/commit?height=%d", height), &response); err != nil {
		return nil, err
	}
	if len(response.Result.SignedHeader.Commit.Signatures) != 4 {
		return nil, fmt.Errorf("commit at H%d has %d signatures, expected four slots", height, len(response.Result.SignedHeader.Commit.Signatures))
	}
	var voters []string
	for _, signature := range response.Result.SignedHeader.Commit.Signatures {
		if signature.BlockIDFlag == 2 {
			voters = append(voters, strings.ToUpper(signature.ValidatorAddress))
		}
	}
	return voters, nil
}

func TestB0FourValidatorNetworkTransfer(t *testing.T) {
	net := newFourNetwork(t)
	net.startAll()
	net.waitAll(2, 90*time.Second)
	doc, err := cmttypes.GenesisDocFromFile(filepath.Join(net.nodes[0].home, "config", "genesis.json"))
	if err != nil {
		t.Fatal(err)
	}
	expectedVoters := map[string]bool{}
	for _, validator := range doc.Validators {
		expectedVoters[fmt.Sprintf("%X", validator.Address)] = true
	}
	observedVoters := map[string]bool{}
	var participationHeight int64
	nextCommit := int64(2)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		h, _, err := net.height(0)
		if err == nil {
			for nextCommit <= h {
				voters, err := net.commitVoters(0, nextCommit)
				if err != nil {
					break
				}
				for _, address := range voters {
					if !expectedVoters[address] {
						t.Fatalf("unexpected validator vote at H%d: %s", nextCommit, address)
					}
					observedVoters[address] = true
				}
				participationHeight = nextCommit
				nextCommit++
			}
			if len(observedVoters) == 4 {
				break
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	if len(observedVoters) != 4 {
		t.Fatalf("commits through H%d contain votes from only %d of four validators", participationHeight, len(observedVoters))
	}
	t.Logf("T02: real CometBFT commits through H=%d contain votes from all four distinct validators", participationHeight)
	var sender, receiver b0state.Address
	a, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
	b, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
	copy(sender[:], a)
	copy(receiver[:], b)
	for i := range net.nodes {
		balance, nonce, err := net.account(i, sender)
		if err != nil || balance != 100 || nonce != 0 {
			t.Fatalf("validator %d initial sender=%d/%d err=%v", i, balance, nonce, err)
		}
	}
	bad := []byte{1, 2, 3}
	invalid, err := net.broadcast(bad)
	if err != nil || invalid.Result.CheckTx.Code != 1 {
		t.Fatalf("malformed admission: %+v %v", invalid, err)
	}
	transfer := realNodeSigned(t, receiver, 25, 0)
	badSignature := append([]byte(nil), transfer...)
	badSignature[len(badSignature)-80] ^= 1
	for _, tc := range []struct {
		name string
		wire []byte
		code uint32
	}{
		{"noncanonical", []byte{0xff, 0x00, 0x01}, 1},
		{"wrong-chain", realNodeSignedChain(t, "pluto-other-1", receiver, 25, 0), 2},
		{"invalid-signature", badSignature, 4},
		{"wrong-nonce", realNodeSigned(t, receiver, 25, 1), 6},
		{"insufficient-balance", realNodeSigned(t, receiver, 101, 0), 7},
	} {
		response, err := net.broadcast(tc.wire)
		if err != nil || response.Result.CheckTx.Code != tc.code {
			t.Fatalf("%s CheckTx got=%d want=%d err=%v", tc.name, response.Result.CheckTx.Code, tc.code, err)
		}
	}
	valid, err := net.broadcast(transfer)
	if err != nil || valid.Result.CheckTx.Code != 0 || valid.Result.TxResult.Code != 0 {
		t.Fatalf("valid transfer: %+v %v", valid, err)
	}
	txHeight, err := strconv.ParseInt(valid.Result.Height, 10, 64)
	if err != nil || txHeight < 2 {
		t.Fatalf("transaction height %q %v", valid.Result.Height, err)
	}
	net.waitAll(txHeight+1, 90*time.Second)
	var expectedBlock, expectedApp string
	for i := range net.nodes {
		balance, nonce, err := net.account(i, sender)
		if err != nil || balance != 75 || nonce != 1 {
			t.Fatalf("validator %d sender=%d/%d err=%v", i, balance, nonce, err)
		}
		balance, nonce, err = net.account(i, receiver)
		if err != nil || balance != 25 || nonce != 0 {
			t.Fatalf("validator %d receiver=%d/%d err=%v", i, balance, nonce, err)
		}
		blockHash, appHash, txs, err := net.block(i, txHeight+1)
		if err != nil || blockHash == "" || len(txs) != 0 {
			t.Fatalf("validator %d H+1 block: hash=%s txs=%v err=%v", i, blockHash, txs, err)
		}
		if i == 0 {
			expectedBlock, expectedApp = blockHash, appHash
		} else if blockHash != expectedBlock || appHash != expectedApp {
			t.Fatalf("validator %d matching-height hash mismatch", i)
		}
		_, _, included, err := net.block(i, txHeight)
		if err != nil || len(included) != 1 || included[0] != base64.StdEncoding.EncodeToString(transfer) {
			t.Fatalf("validator %d ordered committed transaction mismatch", i)
		}
	}
	t.Logf("T02/T03/T05: four real validators, transaction H=%d Code=0; common block H=%d hash=%s AppHash(after tx)=%s; sender=75/1 receiver=25/0", txHeight, txHeight+1, expectedBlock, expectedApp)
	// ProcessProposal is exercised directly on an independent application after
	// its real Comet process stops; no mock consensus coordinator is involved.
	net.stopAll()
	for i := range net.nodes {
		app, err := b0app.Open(filepath.Join(net.nodes[i].home, "data", "pluto-b0-v2"), "pluto-b0-v1", net.genesis)
		if err != nil {
			t.Fatal(err)
		}
		state, exists, err := app.Committed()
		if err != nil || !exists {
			t.Fatalf("validator %d committed read: %v", i, err)
		}
		if state.TotalSupply() != 100 {
			t.Fatalf("validator %d total supply=%d", i, state.TotalSupply())
		}
		replayed, err := app.CheckTx(context.Background(), &types.RequestCheckTx{Tx: transfer, Type: types.CheckTxType_New})
		if err != nil || replayed.Code != 6 {
			t.Fatalf("validator %d replay CheckTx=%d err=%v", i, replayed.Code, err)
		}
		candidate := realNodeSigned(t, receiver, 10, 1)
		for _, proposal := range []struct {
			name string
			txs  [][]byte
		}{
			{"malformed", [][]byte{bad}},
			{"invalid-signature", [][]byte{badSignature}},
			{"replay", [][]byte{transfer}},
			{"duplicate-sequence", [][]byte{candidate, candidate}},
			{"valid-then-invalid", [][]byte{candidate, realNodeSigned(t, receiver, 10, 3)}},
		} {
			decision, err := app.ProcessProposal(context.Background(), &types.RequestProcessProposal{Height: state.Height() + 1, Txs: proposal.txs})
			if err != nil || decision.Status != types.ResponseProcessProposal_REJECT {
				t.Fatalf("validator %d %s proposal: %+v %v", i, proposal.name, decision, err)
			}
		}
		unchanged, exists, err := app.Committed()
		if err != nil || !exists || unchanged.AppHash() != state.AppHash() {
			t.Fatalf("validator %d invalid proposals mutated committed state: %v", i, err)
		}
		if err := app.Close(); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("T04: malformed, noncanonical, wrong-chain, bad-signature, wrong-nonce, insufficient-balance and replay CheckTx results; all four stores reject malformed, signature, replay and invalid-sequence ProcessProposal without state mutation")
}

func TestB0FourValidatorQuorumAndCatchup(t *testing.T) {
	seenAcrossRuns := map[string]bool{}
	for run := 0; run < 3; run++ {
		t.Run(fmt.Sprintf("fresh-network-%d", run), func(t *testing.T) {
			net := newFourNetwork(t)
			for i := range net.nodes {
				raw, err := os.ReadFile(filepath.Join(net.nodes[i].home, "config", "priv_validator_key.json"))
				if err != nil {
					t.Fatal(err)
				}
				var key privval.FilePVKey
				if err := cmtjson.Unmarshal(raw, &key); err != nil {
					t.Fatal(err)
				}
				if seenAcrossRuns[string(key.Address)] {
					t.Fatal("validator identity repeated across independently initialized networks")
				}
				seenAcrossRuns[string(key.Address)] = true
			}
			net.startAll()
			net.waitAll(2, 90*time.Second)
			initial, _, err := net.height(0)
			if err != nil {
				t.Fatal(err)
			}
			net.stop(3)
			net.waitSet([]int{0, 1, 2}, initial+3, 60*time.Second)
			three, _, err := net.height(0)
			if err != nil {
				t.Fatal(err)
			}
			net.stop(2)
			// Exclude a final block already in flight at the stop boundary.
			time.Sleep(3 * time.Second)
			before0, _, err0 := net.height(0)
			before1, _, err1 := net.height(1)
			if err0 != nil || err1 != nil {
				t.Fatalf("two-validator status: %v %v", err0, err1)
			}
			time.Sleep(30 * time.Second)
			after0, _, err0 := net.height(0)
			after1, _, err1 := net.height(1)
			if err0 != nil || err1 != nil || after0 != before0 || after1 != before1 {
				t.Fatalf("2/4 incorrectly committed: before=%d/%d after=%d/%d errors=%v/%v", before0, before1, after0, after1, err0, err1)
			}
			net.start(2)
			net.waitSet([]int{0, 1, 2}, before0+3, 60*time.Second)
			restored, _, err := net.height(0)
			if err != nil {
				t.Fatal(err)
			}
			net.start(3)
			net.waitAll(restored, 90*time.Second)
			var wantBlock, wantApp string
			for i := range net.nodes {
				blockHash, appHash, _, err := net.block(i, restored)
				if err != nil || blockHash == "" || appHash == "" {
					t.Fatalf("validator %d restored block H%d: %v", i, restored, err)
				}
				if i == 0 {
					wantBlock, wantApp = blockHash, appHash
				} else if blockHash != wantBlock || appHash != wantApp {
					t.Fatalf("validator %d diverged at matching H%d", i, restored)
				}
			}
			t.Logf("T06–T09 run=%d: 4/4 H=%d; 3/4 H=%d; stable 2/4 %d/%d -> %d/%d over 30s; restored 3/4 H=%d; fourth caught up; matching block=%s AppHash=%s", run, initial, three, before0, before1, after0, after1, restored, wantBlock, wantApp)
		})
	}
}

func TestB0FourValidatorApplicationCommitCrash(t *testing.T) {
	for _, mode := range []string{"before-commit", "post-sync-commit"} {
		t.Run(mode, func(t *testing.T) {
			net := newFourNetwork(t)
			for i := 0; i < 3; i++ {
				net.start(i)
			}
			net.waitSet([]int{0, 1, 2}, 2, 90*time.Second)
			cometBinary, _ := filepath.Abs("../../build/cometbft")
			root := filepath.Dir(net.root)
			ready, blocked := filepath.Join(root, "app.ready"), filepath.Join(root, "app.blocked")
			appLogPath, cometLogPath := filepath.Join(root, "fault-app.log"), filepath.Join(root, "fault-comet.log")
			appLog, err := os.Create(appLogPath)
			if err != nil {
				t.Fatal(err)
			}
			app := exec.Command(os.Args[0], "-test.run=^TestB0M7AsymmetricApplicationCrash$")
			app.Env = append(os.Environ(), "PLUTO_M7_CHILD="+mode, "PLUTO_M7_HOME="+net.nodes[3].home, "PLUTO_M7_READY="+ready, "PLUTO_M7_BLOCKED="+blocked)
			app.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			app.Stdout, app.Stderr = appLog, appLog
			if err := app.Start(); err != nil {
				t.Fatal(err)
			}
			appWaited := false
			t.Cleanup(func() {
				if !appWaited {
					_ = syscall.Kill(-app.Process.Pid, syscall.SIGKILL)
					_ = app.Wait()
				}
				_ = appLog.Close()
			})
			waitMarker(t, ready)
			cometLog, err := os.Create(cometLogPath)
			if err != nil {
				t.Fatal(err)
			}
			comet := exec.Command(cometBinary, "node", "--home", net.nodes[3].home)
			comet.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			comet.Stdout, comet.Stderr = cometLog, cometLog
			if err := comet.Start(); err != nil {
				t.Fatal(err)
			}
			cometWaited := false
			t.Cleanup(func() {
				if !cometWaited {
					_ = syscall.Kill(-comet.Process.Pid, syscall.SIGKILL)
					_ = comet.Wait()
				}
				_ = cometLog.Close()
			})
			t.Cleanup(func() {
				if dir := os.Getenv("PLUTO_4V_ARTIFACTS"); dir != "" {
					_ = os.MkdirAll(dir, 0700)
					for _, path := range []string{appLogPath, cometLogPath, blocked} {
						if data, err := os.ReadFile(path); err == nil {
							_ = os.WriteFile(filepath.Join(dir, mode+"-"+filepath.Base(path)), data, 0600)
						}
					}
				}
			})
			net.waitAll(2, 90*time.Second)
			var receiver b0state.Address
			receiverBytes, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
			copy(receiver[:], receiverBytes)
			wire := realNodeSigned(t, receiver, 25, 0)
			var admission struct {
				Result struct {
					Code uint32 `json:"code"`
				} `json:"result"`
			}
			if err := rpcGetLong(t, net.nodes[0].ports.RPC, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(wire), &admission); err != nil || admission.Result.Code != 0 {
				t.Fatalf("transaction admission: code=%d err=%v", admission.Result.Code, err)
			}
			waitMarker(t, blocked)
			marked := m7ReadMarker(t, blocked, mode)
			if !m7ProcessAlive(comet.Process.Pid) {
				t.Fatal("Comet not alive at application callback marker")
			}
			if err := app.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			if err := app.Wait(); err == nil {
				t.Fatal("application did not SIGKILL")
			}
			appWaited = true
			cometAliveAtKill := m7ProcessAlive(comet.Process.Pid)
			time.Sleep(1200 * time.Millisecond)
			_ = syscall.Kill(-comet.Process.Pid, syscall.SIGKILL)
			_ = comet.Wait()
			cometWaited = true
			_ = cometLog.Close()
			_ = appLog.Close()
			beforeB, beforeC := cometHeights(t, net.nodes[3].home)
			state, err := b0store.ProbeExisting(filepath.Join(net.nodes[3].home, "data", "pluto-b0-v2"), "pluto-b0-v1")
			if err != nil || state.Height() != marked.Height {
				t.Fatalf("fault A mismatch: marker=%+v state=%v err=%v", marked, state, err)
			}
			beforeHash := state.AppHash()
			if hex.EncodeToString(beforeHash[:]) != marked.AppHash {
				t.Fatal("fault AppHash changed during outage")
			}
			if mode == "before-commit" && state.Accounts()[0].Balance != 100 {
				t.Fatal("before-commit applied transaction")
			}
			if mode == "post-sync-commit" && state.Accounts()[0].Balance != 75 {
				t.Fatal("post-sync-commit did not persist transfer")
			}
			net.start(3)
			leaderH, _, err := net.height(0)
			if err != nil {
				t.Fatal(err)
			}
			net.waitAll(leaderH, 90*time.Second)
			// Comet's block H+1 header carries the AppHash returned by
			// application Commit at H. Compare that matching application height.
			compareHeight := state.Height() + 1
			if compareHeight > leaderH {
				net.waitAll(compareHeight, 90*time.Second)
			}
			var recoveredBlock, recoveredApp string
			for i := range net.nodes {
				blockHash, appHash, _, err := net.block(i, compareHeight)
				if err != nil || blockHash == "" || appHash == "" {
					t.Fatalf("validator %d recovered block H%d: %v", i, compareHeight, err)
				}
				if i == 0 {
					recoveredBlock, recoveredApp = blockHash, appHash
				} else if blockHash != recoveredBlock || appHash != recoveredApp {
					t.Fatalf("validator %d recovered matching-height hash mismatch H%d", i, compareHeight)
				}
			}
			if !strings.EqualFold(recoveredApp, marked.AppHash) {
				t.Fatalf("recovered Comet header H%d does not match persisted application AppHash at A=%d: got=%s want=%s", compareHeight, state.Height(), recoveredApp, marked.AppHash)
			}
			var sender b0state.Address
			senderBytes, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
			copy(sender[:], senderBytes)
			for i := range net.nodes {
				balance, nonce, err := net.account(i, sender)
				if err != nil || balance != 75 || nonce != 1 {
					t.Fatalf("validator %d sender after replay=%d/%d err=%v", i, balance, nonce, err)
				}
				balance, nonce, err = net.account(i, receiver)
				if err != nil || balance != 25 || nonce != 0 {
					t.Fatalf("validator %d receiver after replay=%d/%d err=%v", i, balance, nonce, err)
				}
			}
			t.Logf("T10/T11 %s: app PID=%d SIGKILL; Comet PID=%d alive-at-kill=%t; fault B/C/A=%d/%d/%d AppHash(A)=%s; recovered block H=A+1=%d hash=%s header AppHash(A)=%s; all four sender=75/1 receiver=25/0 exactly once", mode, app.Process.Pid, comet.Process.Pid, cometAliveAtKill, beforeB, beforeC, state.Height(), marked.AppHash, compareHeight, recoveredBlock, recoveredApp)
		})
	}
}

func TestB0FourValidatorInitializationRefusals(t *testing.T) {
	for _, mode := range []string{"missing-h0", "corrupt-h0", "validator-key-mismatch", "genesis-mismatch"} {
		t.Run(mode, func(t *testing.T) {
			net := newFourNetwork(t)
			home := net.nodes[0].home
			switch mode {
			case "missing-h0":
				path := filepath.Join(home, "data", "pluto-b0-v2")
				if err := os.Rename(path, path+".held"); err != nil {
					t.Fatal(err)
				}
			case "corrupt-h0":
				path := filepath.Join(home, "data", "pluto-b0-v2")
				db, err := pebble.Open(path, &pebble.Options{})
				if err != nil {
					t.Fatal(err)
				}
				if err := db.Set([]byte("pluto/committed-state/v2"), []byte("corrupt"), pebble.Sync); err != nil {
					t.Fatal(err)
				}
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
			case "validator-key-mismatch":
				path := filepath.Join(home, "config", "priv_validator_key.json")
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(raw, &fields); err != nil {
					t.Fatal(err)
				}
				fields["address"] = json.RawMessage(`"0000000000000000000000000000000000000000"`)
				raw, err = json.Marshal(fields)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
			case "genesis-mismatch":
				path := filepath.Join(home, "config", "genesis.json")
				doc, err := cmttypes.GenesisDocFromFile(path)
				if err != nil {
					t.Fatal(err)
				}
				doc.ChainID = "pluto-conflict-v1"
				if err := doc.SaveAs(path); err != nil {
					t.Fatal(err)
				}
			}
			err := node.Run(context.Background(), home, "/must/not/launch/comet", io.Discard)
			if err == nil || strings.Contains(err.Error(), "start CometBFT") {
				t.Fatalf("%s did not fail closed before Comet launch: %v", mode, err)
			}
			if mode == "missing-h0" && !strings.Contains(err.Error(), "B0 snapshot missing") {
				t.Fatalf("wrong missing-H0 refusal: %v", err)
			}
			if mode == "validator-key-mismatch" && !strings.Contains(err.Error(), "validator key conflicts") {
				t.Fatalf("wrong validator refusal: %v", err)
			}
			t.Logf("T12/T13 %s: production start refused before Comet: %v", mode, err)
		})
	}
}

func TestB0FourValidatorDiagnostics(t *testing.T) {
	net := newFourNetwork(t)
	net.startAll()
	net.waitAll(2, 90*time.Second)
	var sender, receiver b0state.Address
	senderRaw, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
	receiverRaw, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
	copy(sender[:], senderRaw)
	copy(receiver[:], receiverRaw)
	const count = 20
	latencies := make([]time.Duration, 0, count)
	start := time.Now()
	var lastHeight int64
	for nonce := 0; nonce < count; nonce++ {
		wire := realNodeSigned(t, receiver, 1, uint64(nonce))
		begin := time.Now()
		result, err := net.broadcast(wire)
		latencies = append(latencies, time.Since(begin))
		if err != nil || result.Result.CheckTx.Code != 0 || result.Result.TxResult.Code != 0 {
			t.Fatalf("diagnostic transaction %d: %+v %v", nonce, result, err)
		}
		lastHeight, err = strconv.ParseInt(result.Result.Height, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	net.waitAll(lastHeight+1, 90*time.Second)
	for i := range net.nodes {
		balance, nonce, err := net.account(i, sender)
		if err != nil || balance != 80 || nonce != count {
			t.Fatalf("diagnostic validator %d sender=%d/%d err=%v", i, balance, nonce, err)
		}
		balance, nonce, err = net.account(i, receiver)
		if err != nil || balance != 20 || nonce != 0 {
			t.Fatalf("diagnostic validator %d receiver=%d/%d err=%v", i, balance, nonce, err)
		}
		pid := strconv.Itoa(net.nodes[i].pid)
		appSample, appErr := exec.Command("ps", "-p", pid, "-o", "pid=,comm=,%cpu=,rss=").Output()
		childSample, childErr := exec.Command("ps", "--ppid", pid, "-o", "pid=,comm=,%cpu=,rss=").Output()
		t.Logf("resource validator=%d app sample=%q err=%v Comet children=%q err=%v", i, strings.TrimSpace(string(appSample)), appErr, strings.TrimSpace(string(childSample)), childErr)
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	t.Logf("diagnostic sorted per-transaction broadcast_tx_commit wall samples=%v", latencies)
	nearestRank := func(percentile int) time.Duration {
		index := (percentile*len(latencies)+99)/100 - 1
		return latencies[index]
	}
	t.Logf("diagnostic workload=%d sequential amount=1 transfers elapsed=%s throughput=%.3f tx/s broadcast_tx_commit wall latency p50=%s p95=%s p99=%s; all four sender=80/20 receiver=20/0 H>=%d", count, elapsed, float64(count)/elapsed.Seconds(), nearestRank(50), nearestRank(95), nearestRank(99), lastHeight+1)
}

func TestB0FourValidatorCLIInitialization(t *testing.T) {
	root := filepath.Join(t.TempDir(), "four-validator-root")
	genesis, err := hex.DecodeString(b0GoldenGenesis)
	if err != nil {
		t.Fatal(err)
	}
	genesisFile := filepath.Join(t.TempDir(), "genesis-v2.bin")
	if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
		t.Fatal(err)
	}
	// This test initializes homes only; no listeners bind these ports.
	base := node.B0Ports{RPC: 33000, P2P: 34000, ABCI: 35000}
	plutod, _ := filepath.Abs("../../build/plutod")
	cmd := exec.Command(plutod, "init-b0-4v", "--root", root, "--genesis-file", genesisFile, "--rpc-base-port", strconv.Itoa(base.RPC), "--p2p-base-port", strconv.Itoa(base.P2P), "--abci-base-port", strconv.Itoa(base.ABCI))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("four-validator CLI init: %v (%s)", err, output)
	}
	for i := 0; i < 4; i++ {
		home := filepath.Join(root, fmt.Sprintf("validator-%d", i))
		state, err := b0store.ProbeExisting(filepath.Join(home, "data", "pluto-b0-v2"), "pluto-b0-v1")
		if err != nil || state.Height() != 0 {
			t.Fatalf("CLI validator %d H0: %v", i, err)
		}
	}
	retry := exec.Command(plutod, "init-b0-4v", "--root", root, "--genesis-file", genesisFile, "--rpc-base-port", strconv.Itoa(base.RPC), "--p2p-base-port", strconv.Itoa(base.P2P), "--abci-base-port", strconv.Itoa(base.ABCI))
	if output, err := retry.CombinedOutput(); err == nil || !bytes.Contains(output, []byte("already exists")) {
		t.Fatalf("CLI repeated init must fail closed: %v (%s)", err, output)
	}
	t.Log("T01/T14 CLI: explicit four-validator initialization produced four H0 snapshots and refused repeated initialization")
}

func TestB0FourValidatorConcurrentInitialization(t *testing.T) {
	root := filepath.Join(t.TempDir(), "claimed-network")
	genesis, err := hex.DecodeString(b0GoldenGenesis)
	if err != nil {
		t.Fatal(err)
	}
	comet, _ := filepath.Abs("../../build/cometbft")
	ports := [4]node.B0Ports{
		{RPC: 36000, P2P: 37000, ABCI: 38000},
		{RPC: 36010, P2P: 37010, ABCI: 38010},
		{RPC: 36020, P2P: 37020, ABCI: 38020},
		{RPC: 36030, P2P: 37030, ABCI: 38030},
	}
	results := make([]error, 2)
	var start sync.WaitGroup
	start.Add(1)
	var done sync.WaitGroup
	for i := range results {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait()
			results[i] = node.InitB0FourValidators(root, comet, "pluto-b0-v1", genesis, ports, io.Discard)
		}(i)
	}
	start.Done()
	done.Wait()
	succeeded, refused := 0, 0
	for _, err := range results {
		if err == nil {
			succeeded++
		} else if strings.Contains(err.Error(), "already exists") {
			refused++
		} else {
			t.Fatalf("unexpected concurrent initialization error: %v", err)
		}
	}
	if succeeded != 1 || refused != 1 {
		t.Fatalf("concurrent initialization outcomes: success=%d refused=%d", succeeded, refused)
	}
	seen := map[string]bool{}
	var sharedGenesis []byte
	for i := 0; i < 4; i++ {
		home := filepath.Join(root, fmt.Sprintf("validator-%d", i))
		keyBytes, err := os.ReadFile(filepath.Join(home, "config", "priv_validator_key.json"))
		if err != nil {
			t.Fatal(err)
		}
		var key privval.FilePVKey
		if err := cmtjson.Unmarshal(keyBytes, &key); err != nil || seen[string(key.Address)] {
			t.Fatalf("validator %d identity after concurrent initialization: %v", i, err)
		}
		seen[string(key.Address)] = true
		genesisBytes, err := os.ReadFile(filepath.Join(home, "config", "genesis.json"))
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			sharedGenesis = genesisBytes
		} else if !bytes.Equal(sharedGenesis, genesisBytes) {
			t.Fatalf("validator %d genesis differs after concurrent initialization", i)
		}
		state, err := b0store.ProbeExisting(filepath.Join(home, "data", "pluto-b0-v2"), "pluto-b0-v1")
		if err != nil || state.Height() != 0 {
			t.Fatalf("validator %d after concurrent initialization: %v", i, err)
		}
	}
	t.Log("T14: atomic root claim allowed exactly one complete initialization; second claimant did not overwrite four identities or H0 stores")
}
