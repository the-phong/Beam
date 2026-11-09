//go:build integration && fourval

package integration

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/cometbft/cometbft/abci/server"
	"github.com/cometbft/cometbft/config"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	cmttypes "github.com/cometbft/cometbft/types"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/researchv2"
	"example.com/pluto/internal/researchv2app"
	"example.com/pluto/internal/researchv2corpus"
	"example.com/pluto/internal/researchv2genesis"
	"example.com/pluto/internal/researchv2store"
)

func s04Account(t *testing.T, n *fourNetwork, i int, address b0state.Address) (uint64, uint64) {
	t.Helper()
	var response struct {
		Result struct {
			Response struct {
				Code  uint32 `json:"code"`
				Value string `json:"value"`
			} `json:"response"`
		} `json:"result"`
	}
	path := "/abci_query?path=%22rv2%2Faccount%2Fv1%22&data=0x" + hex.EncodeToString(address[:])
	if err := rpcGetLong(t, n.nodes[i].ports.RPC, path, &response); err != nil {
		t.Fatal(err)
	}
	value, err := base64.StdEncoding.DecodeString(response.Result.Response.Value)
	if err != nil || response.Result.Response.Code != 0 || len(value) != 17 || value[0] != 1 {
		t.Fatalf("validator %d account response code=%d len=%d err=%v", i, response.Result.Response.Code, len(value), err)
	}
	return binary.BigEndian.Uint64(value[1:9]), binary.BigEndian.Uint64(value[9:17])
}

func s04Duplicate(t *testing.T, n *fourNetwork, raw []byte) string {
	t.Helper()
	client := http.Client{Timeout: 15 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/broadcast_tx_commit?tx=0x%s", n.nodes[0].ports.RPC, hex.EncodeToString(raw)))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode == http.StatusInternalServerError && bytes.Contains(body, []byte("tx already exists in cache")) {
		return "comet_cache"
	}
	if response.StatusCode == http.StatusOK {
		var result broadcastResult
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal(err)
		}
		if result.Result.CheckTx.Code == uint32(researchv2.CodeInvalidNonce) {
			return "app_nonce"
		}
	}
	t.Fatalf("duplicate was not rejected: HTTP %d body=%s", response.StatusCode, body)
	return ""
}

func TestS04FourValidatorProfiles(t *testing.T) {
	comet, _ := filepath.Abs("../../build/cometbft")
	if info, err := os.Stat(comet); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("build CometBFT first: %v", err)
	}
	corpus, err := researchv2corpus.Read(filepath.Join("..", "..", "research-v2", "corpus", "s04_public_corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	binaryGenesis, err := hex.DecodeString(corpus.GenesisHex)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []researchv2.Profile{researchv2.ProfileC, researchv2.ProfileCP, researchv2.ProfileP, researchv2.ProfileH} {
		t.Run(fmt.Sprintf("profile-%d", profile), func(t *testing.T) {
			var raw []byte
			for _, item := range corpus.Cases {
				if item.Profile == profile && item.ExpectedCode == researchv2.CodeOK {
					raw, err = hex.DecodeString(item.BytesHex)
					if err != nil {
						t.Fatal(err)
					}
					break
				}
			}
			if len(raw) == 0 {
				t.Fatal("profile missing valid corpus case")
			}
			tx, err := researchv2.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			transport, err := researchv2genesis.Encode(binaryGenesis, corpus.ChainID)
			if err != nil {
				t.Fatal(err)
			}
			n := &fourNetwork{t: t, root: filepath.Join(t.TempDir(), "network"), genesis: binaryGenesis}
			if err := os.Mkdir(n.root, 0700); err != nil {
				t.Fatal(err)
			}
			used := map[int]bool{}
			var keys [4]privval.FilePVKey
			var ids [4]string
			for i := range n.nodes {
				n.nodes[i].home = filepath.Join(n.root, fmt.Sprintf("validator-%d", i))
				for _, ptr := range []*int{&n.nodes[i].ports.RPC, &n.nodes[i].ports.P2P, &n.nodes[i].ports.ABCI} {
					for {
						p := freePort(t)
						if !used[p] {
							*ptr = p
							used[p] = true
							break
						}
					}
				}
				cmd := exec.Command(comet, "init", "--home", n.nodes[i].home)
				cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
				if err := cmd.Run(); err != nil {
					t.Fatal(err)
				}
				raw, err := os.ReadFile(filepath.Join(n.nodes[i].home, "config", "priv_validator_key.json"))
				if err != nil {
					t.Fatal(err)
				}
				if err := cmtjson.Unmarshal(raw, &keys[i]); err != nil {
					t.Fatal(err)
				}
				nodeKey, err := p2p.LoadNodeKey(filepath.Join(n.nodes[i].home, "config", "node_key.json"))
				if err != nil {
					t.Fatal(err)
				}
				ids[i] = string(nodeKey.ID())
				for j := 0; j < i; j++ {
					if bytes.Equal(keys[i].Address, keys[j].Address) || ids[i] == ids[j] {
						t.Fatal("duplicate validator identity")
					}
				}
			}
			genesis, err := cmttypes.GenesisDocFromFile(filepath.Join(n.nodes[0].home, "config", "genesis.json"))
			if err != nil {
				t.Fatal(err)
			}
			genesis.ChainID, genesis.InitialHeight, genesis.AppState = corpus.ChainID, 1, transport
			genesis.Validators = make([]cmttypes.GenesisValidator, 4)
			for i := range keys {
				genesis.Validators[i] = cmttypes.GenesisValidator{Address: keys[i].Address, PubKey: keys[i].PubKey, Power: 10, Name: fmt.Sprintf("validator-%d", i)}
			}
			if err := genesis.ValidateAndComplete(); err != nil {
				t.Fatal(err)
			}
			var apps [4]*researchv2app.App
			var stopServers []func() error
			t.Cleanup(func() {
				n.stopAll()
				for _, stop := range stopServers {
					_ = stop()
				}
				for _, app := range apps {
					if app != nil {
						_ = app.Close()
					}
				}
			})
			for i := range n.nodes {
				home := n.nodes[i].home
				if err := genesis.SaveAs(filepath.Join(home, "config", "genesis.json")); err != nil {
					t.Fatal(err)
				}
				cfg := config.DefaultConfig()
				cfg.SetRoot(home)
				cfg.ProxyApp = fmt.Sprintf("tcp://127.0.0.1:%d", n.nodes[i].ports.ABCI)
				cfg.RPC.ListenAddress = fmt.Sprintf("tcp://127.0.0.1:%d", n.nodes[i].ports.RPC)
				cfg.P2P.ListenAddress = fmt.Sprintf("tcp://127.0.0.1:%d", n.nodes[i].ports.P2P)
				cfg.Consensus.CreateEmptyBlocks = true
				cfg.Consensus.CreateEmptyBlocksInterval = time.Second
				cfg.P2P.AllowDuplicateIP = true
				cfg.P2P.AddrBookStrict = false
				var peers []string
				for j := range n.nodes {
					if j != i {
						peers = append(peers, fmt.Sprintf("%s@127.0.0.1:%d", ids[j], n.nodes[j].ports.P2P))
					}
				}
				cfg.P2P.PersistentPeers = strings.Join(peers, ",")
				config.WriteConfigFile(filepath.Join(home, "config", "config.toml"), cfg)
				dir := filepath.Join(home, "data", "pluto-rv2-v1")
				store, err := researchv2store.Create(dir, binaryGenesis, corpus.ChainID, profile)
				if err != nil {
					t.Fatal(err)
				}
				if err := store.Close(); err != nil {
					t.Fatal(err)
				}
				apps[i], err = researchv2app.Open(dir, binaryGenesis, corpus.ChainID, profile)
				if err != nil {
					t.Fatal(err)
				}
				abciServer, err := server.NewServer(cfg.ProxyApp, "socket", apps[i])
				if err != nil {
					t.Fatal(err)
				}
				if err := abciServer.Start(); err != nil {
					t.Fatal(err)
				}
				stopServers = append(stopServers, abciServer.Stop)
				logPath := filepath.Join(filepath.Dir(n.root), fmt.Sprintf("validator-%d.log", i))
				logFile, err := os.Create(logPath)
				if err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command(comet, "node", "--home", home)
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				cmd.Stdout, cmd.Stderr = logFile, logFile
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				n.nodes[i].cmd, n.nodes[i].pid, n.nodes[i].log, n.nodes[i].done = cmd, cmd.Process.Pid, logFile, make(chan error, 1)
				n.nodes[i].logs = []string{logPath}
				go func() { n.nodes[i].done <- cmd.Wait() }()
			}
			n.waitAll(2, 90*time.Second)
			result, err := n.broadcast(raw)
			if err != nil || result.Result.CheckTx.Code != 0 || result.Result.TxResult.Code != 0 {
				t.Fatalf("valid transfer: %+v %v", result, err)
			}
			h, err := strconv.ParseInt(result.Result.Height, 10, 64)
			if err != nil || h < 2 {
				t.Fatalf("invalid tx height %q: %v", result.Result.Height, err)
			}
			n.waitAll(h+1, 90*time.Second)
			expected, err := researchv2.ParseGenesis(binaryGenesis, corpus.ChainID)
			if err != nil {
				t.Fatal(err)
			}
			for height := int64(1); height < h; height++ {
				expected, err = expected.AdvanceBlock()
				if err != nil {
					t.Fatal(err)
				}
			}
			expected, resultOracle, err := researchv2.EvaluateTrusted(expected, profile, raw)
			if err != nil || resultOracle.Code != researchv2.CodeOK {
				t.Fatalf("state oracle: %+v %v", resultOracle, err)
			}
			expected, err = expected.AdvanceBlock()
			if err != nil {
				t.Fatal(err)
			}
			expectedDigest := expected.AppHash()
			expectedHash := strings.ToUpper(hex.EncodeToString(expectedDigest[:]))
			var blockHash, appHash string
			for i := range n.nodes {
				b, a, txs, err := n.block(i, h)
				if err != nil || len(txs) != 1 || txs[0] != base64.StdEncoding.EncodeToString(raw) {
					t.Fatalf("validator %d tx block: %v txs=%d", i, err, len(txs))
				}
				if i == 0 {
					blockHash, appHash = b, a
				} else if b != blockHash || a != appHash {
					t.Fatalf("validator %d block/AppHash divergence", i)
				}
				balance, nonce := s04Account(t, n, i, b0state.Address(tx.Sender))
				if balance != 93 || nonce != 1 {
					t.Fatalf("validator %d sender=%d/%d", i, balance, nonce)
				}
				balance, nonce = s04Account(t, n, i, b0state.Address(tx.Receiver))
				if balance != 7 || nonce != 0 {
					t.Fatalf("validator %d receiver=%d/%d", i, balance, nonce)
				}
			}
			if blockHash == "" || appHash == "" {
				t.Fatal("missing block or AppHash")
			}
			n.waitAll(h+8, 90*time.Second)
			observed := map[string]bool{}
			for height := h + 1; height <= h+8; height++ {
				voters, err := n.commitVoters(0, height)
				if err != nil || len(voters) < 3 {
					t.Fatalf("commit H%d: %v voters=%v", height, err, voters)
				}
				for _, voter := range voters {
					observed[voter] = true
				}
			}
			if len(observed) != 4 {
				t.Fatalf("only %d/4 validator identities voted in eight commits", len(observed))
			}
			duplicate := s04Duplicate(t, n, raw)
			var postBlock string
			for i := range n.nodes {
				b, postHash, txs, err := n.block(i, h+1)
				if err != nil || len(txs) != 0 || postHash != expectedHash {
					t.Fatalf("validator %d post-state block: %v txs=%d apphash=%s expected=%s", i, err, len(txs), postHash, expectedHash)
				}
				if i == 0 {
					postBlock = b
				} else if b != postBlock {
					t.Fatalf("validator %d post-state block hash diverged", i)
				}
			}
			t.Logf("profile=%d tx_height=%d block_hash=%s app_hash=%s voter_identities=%d sender=93/1 receiver=7/0 duplicate=%s", profile, h, blockHash, appHash, len(observed), duplicate)
		})
	}
}
