//go:build integration

package integration

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/tx"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

func realNodeSigned(t *testing.T, receiver b0state.Address, amount, nonce uint64) []byte {
	return realNodeSignedChain(t, "pluto-b0-v1", receiver, amount, nonce)
}

func realNodeSignedChain(t *testing.T, chain string, receiver b0state.Address, amount, nonce uint64) []byte {
	t.Helper()
	var secret [32]byte
	secret[31] = 1 // Public TEST ONLY scalar; never a validator key.
	key := secp256k1.PrivKeyFromBytes(secret[:])
	record := tx.Transaction{ChainID: chain, Receiver: receiver, Amount: amount, Nonce: nonce}
	copy(record.Sender[:], key.PubKey().SerializeCompressed())
	digest, err := tx.SigningDigest(record)
	if err != nil {
		t.Fatal(err)
	}
	sig := ecdsa.Sign(key, digest[:])
	r, s := sig.R(), sig.S()
	r.PutBytes(&record.R)
	s.PutBytes(&record.S)
	wire, err := tx.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

type broadcastResult struct {
	Result struct {
		CheckTx struct {
			Code uint32 `json:"code"`
			Log  string `json:"log"`
		} `json:"check_tx"`
		TxResult struct {
			Code uint32 `json:"code"`
			Log  string `json:"log"`
		} `json:"tx_result"`
		Hash   string `json:"hash"`
		Height string `json:"height"`
	} `json:"result"`
	Error any `json:"error"`
}

func rpcGetLong(t *testing.T, port int, path string, target any) error {
	t.Helper()
	client := http.Client{Timeout: 15 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, path))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("RPC HTTP %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func TestB0RealTransactionCommitAndRestart(t *testing.T) {
	plutod, err := filepath.Abs("../../build/plutod")
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(t.TempDir(), "b0-home")
	genesisFile := filepath.Join(t.TempDir(), "genesis-v2.bin")
	genesis, err := hex.DecodeString(b0GoldenGenesis)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(genesisFile, genesis, 0600); err != nil {
		t.Fatal(err)
	}
	rpc, p2p, abciPort := freePort(t), freePort(t), freePort(t)
	if rpc == p2p || rpc == abciPort || p2p == abciPort {
		t.Skip("ephemeral port collision")
	}
	initArgs := []string{"init-b0", "--home", home, "--chain-id", "pluto-b0-v1", "--genesis-file", genesisFile,
		"--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abciPort)}
	if out, err := exec.Command(plutod, initArgs...).CombinedOutput(); err != nil {
		t.Fatalf("init B0: %v: %s", err, out)
	}
	var process *exec.Cmd
	start := func(label string) {
		file, err := os.Create(filepath.Join(t.TempDir(), label+".log"))
		if err != nil {
			t.Fatal(err)
		}
		process = exec.Command(plutod, "start", "--home", home)
		process.Stdout, process.Stderr = file, file
		if err := process.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = file.Close() })
	}
	stop := func() {
		if process == nil {
			return
		}
		_ = process.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- process.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("stop B0: %v", err)
			}
		case <-time.After(15 * time.Second):
			_ = process.Process.Kill()
			<-done
			t.Error("B0 stop timeout")
		}
		process = nil
	}
	t.Cleanup(stop)
	start("first")
	b0WaitHeight(t, rpc, 2)
	var sender, receiver b0state.Address
	addrA, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
	addrB, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
	copy(sender[:], addrA)
	copy(receiver[:], addrB)
	broadcast := func(raw []byte) broadcastResult {
		t.Helper()
		var r broadcastResult
		if err := rpcGetLong(t, rpc, "/broadcast_tx_commit?tx=0x"+hex.EncodeToString(raw), &r); err != nil {
			t.Fatal(err)
		}
		if r.Error != nil {
			t.Fatalf("broadcast RPC error: %+v", r.Error)
		}
		return r
	}
	query := func(address b0state.Address) (uint64, uint64) {
		t.Helper()
		var response struct {
			Result struct {
				Response struct {
					Code  uint32 `json:"code"`
					Value string `json:"value"`
				} `json:"response"`
			} `json:"result"`
		}
		path := "/abci_query?path=%22b0%2Faccount%2Fv1%22&data=0x" + hex.EncodeToString(address[:])
		if err := rpcGet(t, rpc, path, &response); err != nil {
			t.Fatal(err)
		}
		value, err := base64.StdEncoding.DecodeString(response.Result.Response.Value)
		if err != nil || response.Result.Response.Code != 0 || len(value) != 17 || value[0] != 1 {
			t.Fatalf("account query: %+v %v", response, err)
		}
		return binary.BigEndian.Uint64(value[1:9]), binary.BigEndian.Uint64(value[9:17])
	}
	first := realNodeSigned(t, receiver, 25, 0)
	bad := []byte{1, 2, 3}
	invalid := broadcast(bad)
	if invalid.Result.CheckTx.Code != 1 {
		t.Fatalf("malformed CheckTx: %+v", invalid)
	}
	badSignature := append([]byte(nil), first...)
	badSignature[len(badSignature)-80] ^= 1
	for _, item := range []struct {
		raw  []byte
		code uint32
	}{
		{realNodeSignedChain(t, "pluto-other-1", receiver, 25, 0), 2},
		{badSignature, 4},
		{realNodeSigned(t, receiver, 25, 1), 6},
		{realNodeSigned(t, receiver, 101, 0), 7},
	} {
		response := broadcast(item.raw)
		if response.Result.CheckTx.Code != item.code {
			t.Fatalf("negative CheckTx code=%d want=%d", response.Result.CheckTx.Code, item.code)
		}
	}
	valid := broadcast(first)
	cometHash := tx.CometHash(first)
	if valid.Result.CheckTx.Code != 0 || valid.Result.TxResult.Code != 0 || !strings.EqualFold(valid.Result.Hash, hex.EncodeToString(cometHash[:])) {
		t.Fatalf("valid broadcast: %+v", valid)
	}
	height, err := strconv.ParseInt(valid.Result.Height, 10, 64)
	if err != nil || height < 1 {
		t.Fatalf("broadcast height: %+v %v", valid, err)
	}
	if balance, nonce := query(sender); balance != 75 || nonce != 1 {
		t.Fatalf("sender after transfer: %d/%d", balance, nonce)
	}
	if balance, nonce := query(receiver); balance != 25 || nonce != 0 {
		t.Fatalf("receiver after transfer: %d/%d", balance, nonce)
	}
	var block blockResponse
	if err := rpcGet(t, rpc, fmt.Sprintf("/block?height=%d", height), &block); err != nil {
		t.Fatal(err)
	}
	if len(block.Result.Block.Data.Txs) != 1 || block.Result.Block.Data.Txs[0] != base64.StdEncoding.EncodeToString(first) {
		t.Fatalf("included transaction bytes: %+v", block)
	}
	b0WaitHeight(t, rpc, height+1)
	if err := rpcGet(t, rpc, fmt.Sprintf("/block?height=%d", height+1), &block); err != nil {
		t.Fatal(err)
	}
	expected, err := b0state.NewState("pluto-b0-v1", height, 100, []b0state.Account{{Address: sender, Balance: 75, Nonce: 1}, {Address: receiver, Balance: 25}})
	if err != nil {
		t.Fatal(err)
	}
	hash := expected.AppHash()
	if !strings.EqualFold(block.Result.Block.Header.AppHash, hex.EncodeToString(hash[:])) {
		t.Fatalf("header H+1 does not contain state hash after H: %s expected %X", block.Result.Block.Header.AppHash, hash)
	}
	t.Logf("transaction H=%d CometHash=%s; header H+1 AppHash=%X; Pluto TxID is separate", height, valid.Result.Hash, hash)
	stop()
	start("restart")
	b0WaitHeight(t, rpc, height+2)
	if balance, nonce := query(sender); balance != 75 || nonce != 1 {
		t.Fatalf("sender after restart: %d/%d", balance, nonce)
	}
	replay := broadcast(first)
	if replay.Result.CheckTx.Code != 6 {
		t.Fatalf("replay accepted: %+v", replay)
	}
	second := broadcast(realNodeSigned(t, receiver, 10, 1))
	if second.Result.CheckTx.Code != 0 || second.Result.TxResult.Code != 0 {
		t.Fatalf("post-restart tx: %+v", second)
	}
	if balance, nonce := query(sender); balance != 65 || nonce != 2 {
		t.Fatalf("sender after second transfer: %d/%d", balance, nonce)
	}
	if balance, nonce := query(receiver); balance != 35 || nonce != 0 {
		t.Fatalf("receiver after second transfer: %d/%d", balance, nonce)
	}
}
