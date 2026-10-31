//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	b0app "example.com/pluto/internal/classical/app"
	b0genesis "example.com/pluto/internal/classical/genesis"
	"example.com/pluto/internal/node"
	"github.com/cometbft/cometbft/abci/server"
	abci "github.com/cometbft/cometbft/abci/types"
)

type genesisResponseGate struct {
	*b0app.App
	synced  chan struct{}
	release chan struct{}
}

func (g *genesisResponseGate) InitChain(ctx context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	resp, err := g.App.InitChain(ctx, req)
	if err == nil {
		close(g.synced)
		<-g.release
	}
	return resp, err
}

func TestB0M11RealInitChainResponseLoss(t *testing.T) {
	plutod, _ := filepath.Abs("../../build/plutod")
	comet, _ := filepath.Abs("../../build/cometbft")
	root := t.TempDir()
	home := filepath.Join(root, "home")
	genesisFile := filepath.Join(root, "genesis.bin")
	genesis, _ := hex.DecodeString(b0GoldenGenesis)
	if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
		t.Fatal(err)
	}
	rpc, p2p, abciPort := 0, 0, 0
	for attempt := 0; attempt < 10; attempt++ {
		rpc, p2p, abciPort = freePort(t), freePort(t), freePort(t)
		if rpc != p2p && rpc != abciPort && p2p != abciPort {
			break
		}
	}
	if rpc == p2p || rpc == abciPort || p2p == abciPort {
		t.Fatal("could not allocate distinct ports")
	}
	args := []string{"init-b0", "--home", home, "--chain-id", "pluto-b0-v1", "--genesis-file", genesisFile,
		"--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abciPort)}
	if out, err := exec.Command(plutod, args...).CombinedOutput(); err != nil {
		t.Fatalf("init: %v %s", err, out)
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
	gate := &genesisResponseGate{App: a, synced: make(chan struct{}), release: make(chan struct{})}
	srv, err := server.NewServer(config.ABCIAddress, "socket", gate)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	firstLog, err := os.Create(filepath.Join(root, "comet-first.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(comet, "node", "--home", home)
	cmd.Stdout, cmd.Stderr = firstLog, firstLog
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		_ = srv.Stop()
		_ = a.Close()
		_ = firstLog.Close()
	})
	select {
	case <-gate.synced:
	case <-time.After(30 * time.Second):
		t.Fatal("real InitChain did not reach post-validation response gate")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("Comet was not killed")
	}
	close(gate.release)
	if err := srv.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	_ = firstLog.Close()
	beforeB, beforeC := cometHeights(t, home)
	reopened, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	before, exists, err := reopened.Committed()
	if err != nil || !exists || before.Height() != 0 {
		t.Fatalf("genesis reopened: %v exists=%v A=%d", err, exists, before.Height())
	}
	accounts := before.Accounts()
	if len(accounts) != 1 || accounts[0].Balance != 100 || accounts[0].Nonce != 0 {
		t.Fatalf("genesis effects=%+v", accounts)
	}
	if beforeB != 0 || beforeC != 0 {
		t.Fatalf("genesis response loss B=%d C=%d A=0", beforeB, beforeC)
	}
	expectedTransport, err := b0genesis.Encode(genesis, "pluto-b0-v1")
	if err != nil {
		t.Fatal(err)
	}
	capture := &capturedB0App{App: reopened, observed: make(chan []byte, 1)}
	second, err := server.NewServer(config.ABCIAddress, "socket", capture)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Start(); err != nil {
		t.Fatal(err)
	}
	defer second.Stop()
	defer reopened.Close()
	restartLog, err := os.Create(filepath.Join(root, "comet-restart.log"))
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
	var retried []byte
	select {
	case retried = <-capture.observed:
	case <-time.After(30 * time.Second):
		t.Fatal("Comet did not retry InitChain")
	}
	if !bytes.Equal(retried, expectedTransport) {
		t.Fatal("retry transport differs from initial genesis")
	}
	b0WaitHeight(t, rpc, 2)
	if err := restart.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := restart.Wait(); err != nil {
		t.Fatal(err)
	}
	after, _, err := reopened.Committed()
	if err != nil {
		t.Fatal(err)
	}
	finalB, finalC := cometHeights(t, home)
	if finalB != finalC || finalC != after.Height() || after.Accounts()[0].Balance != 100 || after.Accounts()[0].Nonce != 0 {
		t.Fatalf("post-retry B=%d C=%d A=%d accounts=%+v", finalB, finalC, after.Height(), after.Accounts())
	}
	if artifactDir := os.Getenv("PLUTO_RECOVERY_ARTIFACTS"); artifactDir != "" {
		if err := os.MkdirAll(artifactDir, 0700); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"comet-first.log", "comet-restart.log"} {
			raw, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(artifactDir, "m11-"+name), raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Logf("D-026 real InitChain response withheld after validating preexisting synced A=0; SIGKILL Comet; before B=%d C=%d A=0 hash=%x sender=100/0; identical JSON/Base64 retry observed; after B=%d C=%d A=%d hash=%x sender=100/0", beforeB, beforeC, before.AppHash(), finalB, finalC, after.Height(), after.AppHash())
}
