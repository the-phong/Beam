//go:build integration

package integration

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"example.com/pluto/internal/b0app"
	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/node"
	"github.com/cometbft/cometbft/abci/server"
)

func TestB0M6CometPostCommitAckFailpoint(t *testing.T) {
	plutod, _ := filepath.Abs("../../build/plutod")
	comet, _ := filepath.Abs("../../build/cometbft")
	root := t.TempDir()
	home := filepath.Join(root, "home")
	genesisFile := filepath.Join(root, "genesis.bin")
	genesis, _ := hex.DecodeString(b0GoldenGenesis)
	if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
		t.Fatal(err)
	}
	rpc, p2p, abciPort := freePort(t), freePort(t), freePort(t)
	if rpc == p2p || rpc == abciPort || p2p == abciPort {
		t.Skip("ephemeral port collision")
	}
	args := []string{"init-b0", "--home", home, "--chain-id", "pluto-b0-v1", "--genesis-file", genesisFile,
		"--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abciPort)}
	if out, err := exec.Command(plutod, args...).CombinedOutput(); err != nil {
		t.Fatalf("init: %v %s", err, out)
	}
	configFile := filepath.Join(home, "config", "config.toml")
	configRaw, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configRaw), "create_empty_blocks = true") {
		t.Fatal("cannot pin first transaction block")
	}
	configRaw = []byte(strings.Replace(string(configRaw), "create_empty_blocks = true", "create_empty_blocks = false", 1))
	if err := os.WriteFile(configFile, configRaw, 0600); err != nil {
		t.Fatal(err)
	}
	config, err := node.ReadConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	dbDir := filepath.Join(home, "data", "pluto-b0-v2")
	a, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := server.NewServer(config.ABCIAddress, "socket", a)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()
	defer a.Close()
	firstLogPath := filepath.Join(root, "comet-first.log")
	firstLog, err := os.Create(firstLogPath)
	if err != nil {
		t.Fatal(err)
	}
	index := os.Getenv("PLUTO_M6_FAIL_INDEX")
	if index == "" {
		index = "7"
	}
	cmd := exec.Command(comet, "node", "--home", home)
	cmd.Env = append(os.Environ(), "FAIL_TEST_INDEX="+index)
	cmd.Stdout, cmd.Stderr = firstLog, firstLog
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		_ = firstLog.Close()
	})
	deadline := time.Now().Add(30 * time.Second)
	for {
		var status statusResponse
		if err := rpcGet(t, rpc, "/status", &status); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Comet RPC not ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
	var receiver, sender b0state.Address
	aBytes, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
	bBytes, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
	copy(sender[:], aBytes)
	copy(receiver[:], bBytes)
	wire := realNodeSigned(t, receiver, 25, 0)
	var admission struct {
		Result struct {
			Code uint32 `json:"code"`
		} `json:"result"`
	}
	if err := rpcGet(t, rpc, "/broadcast_tx_sync?tx=0x"+hex.EncodeToString(wire), &admission); err != nil {
		t.Fatal(err)
	}
	if admission.Result.Code != 0 {
		t.Fatalf("admission=%d", admission.Result.Code)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatal("failpoint did not terminate Comet")
	}
	_ = firstLog.Close()
	firstRaw, err := os.ReadFile(firstLogPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeB, beforeC := cometHeights(t, home)
	before, exists, err := a.Committed()
	if err != nil || !exists {
		t.Fatalf("app state: %v", err)
	}
	response, responseErr := cometFinalizedResponse(t, home, beforeB)
	t.Logf("M6 exploratory index=%s B=%d C=%d A=%d hash=%x finalized_response_err=%v log_tail=%s", index, beforeB, beforeC, before.Height(), before.AppHash(), responseErr, string(firstRaw[max(0, len(firstRaw)-1500):]))
	beforeHash := before.AppHash()
	if !strings.Contains(string(firstRaw), "*** fail-test "+index+" ***") || !strings.Contains(string(firstRaw), "committed state") || beforeB != 1 || beforeC != 0 || before.Height() != 1 || responseErr != nil || len(response.TxResults) != 1 || response.TxResults[0].Code != 0 || !bytes.Equal(response.AppHash, beforeHash[:]) {
		t.Fatal("M6 post-ack/pre-state-save boundary was not reached")
	}
	account, _ := before.Account(sender)
	receiverAccount, found := before.Account(receiver)
	if account.Balance != 75 || account.Nonce != 1 || !found || receiverAccount.Balance != 25 {
		t.Fatalf("pre-restart transfer=%+v receiver=%+v", account, receiverAccount)
	}
	restartLogPath := filepath.Join(root, "comet-restart.log")
	restartLog, err := os.Create(restartLogPath)
	if err != nil {
		t.Fatal(err)
	}
	restart := exec.Command(comet, "node", "--home", home)
	restart.Stdout, restart.Stderr = restartLog, restartLog
	if err := restart.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if restart.ProcessState == nil {
			_ = restart.Process.Kill()
			_ = restart.Wait()
		}
		_ = restartLog.Close()
	})
	deadline = time.Now().Add(30 * time.Second)
	for {
		var status statusResponse
		if err := rpcGet(t, rpc, "/status", &status); err == nil {
			height, heightErr := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64)
			if heightErr == nil && height == beforeB && status.Result.NodeInfo.Network == "pluto-b0-v1" {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("restart RPC did not expose the expected chain and block height")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := restart.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := restart.Wait(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatal(err)
		}
		status, ok := exitErr.Sys().(syscall.WaitStatus)
		if !ok || !status.Signaled() || status.Signal() != syscall.SIGTERM {
			t.Fatal(err)
		}
		// RPC can become ready before CometBFT installs its SIGTERM trap.
		// A direct SIGTERM exit is acceptable; replayed state is checked below.
	}
	after, _, err := a.Committed()
	if err != nil {
		t.Fatal(err)
	}
	finalB, finalC := cometHeights(t, home)
	account, _ = after.Account(sender)
	receiverAccount, found = after.Account(receiver)
	if finalB != finalC || finalC != after.Height() || account.Balance != 75 || account.Nonce != 1 || !found || receiverAccount.Balance != 25 || receiverAccount.Nonce != 0 {
		t.Fatalf("post-restart B=%d C=%d A=%d sender=%+v receiver=%+v", finalB, finalC, after.Height(), account, receiverAccount)
	}
	restartRaw, err := os.ReadFile(restartLogPath)
	if err != nil {
		t.Fatal(err)
	}
	if artifactDir := os.Getenv("PLUTO_RECOVERY_ARTIFACTS"); artifactDir != "" {
		if err := os.MkdirAll(artifactDir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(artifactDir, "m6-comet-first.log"), firstRaw, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(artifactDir, "m6-comet-restart.log"), restartRaw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("M6 FAIL_TEST_INDEX=%s after successful Commit response before Comet state Save; before B=%d C=%d A=%d hash=%x sender=75/1 receiver=25/0; restart mock replay; after B=%d C=%d A=%d hash=%x sender=75/1 receiver=25/0", index, beforeB, beforeC, before.Height(), before.AppHash(), finalB, finalC, after.Height(), after.AppHash())
}
