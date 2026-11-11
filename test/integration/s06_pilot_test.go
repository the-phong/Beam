//go:build integration && fourval

package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	researchv2 "example.com/pluto/internal/profiles"
	researchv2app "example.com/pluto/internal/profiles/app"
	researchv2genesis "example.com/pluto/internal/profiles/genesis"
	"example.com/pluto/internal/profiles/load"
	researchv2store "example.com/pluto/internal/profiles/store"
	"example.com/pluto/internal/telemetry"
)

type s06Network struct {
	n            *fourNetwork
	metricsPorts [4]int
	apps         [4]*researchv2app.App
	recorders    [4]*telemetry.Recorder
	observers    [4]*exec.Cmd
	observerLogs [4]*os.File
	observerErrs [4]*os.File
	nodes        [4]load.WatchedNode
}

func startS06Network(t *testing.T, c load.Config, f *load.PilotFixture, profile researchv2.Profile) *s06Network {
	t.Helper()
	comet, _ := filepath.Abs("../../build/cometbft-s05")
	if _, err := os.Stat(comet); err != nil {
		t.Fatalf("build instrumented Comet first: %v", err)
	}
	transport, err := researchv2genesis.Encode(f.Genesis, f.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	pilot := &s06Network{n: &fourNetwork{t: t, root: filepath.Join(t.TempDir(), "network"), genesis: f.Genesis}}
	n := pilot.n
	if err := os.Mkdir(n.root, 0700); err != nil {
		t.Fatal(err)
	}
	used := map[int]bool{}
	var keys [4]privval.FilePVKey
	var ids [4]string
	for i := range n.nodes {
		n.nodes[i].home = filepath.Join(n.root, fmt.Sprintf("validator-%d", i))
		for _, ptr := range []*int{&n.nodes[i].ports.RPC, &n.nodes[i].ports.P2P, &n.nodes[i].ports.ABCI, &pilot.metricsPorts[i]} {
			for {
				port := freePort(t)
				if !used[port] {
					*ptr = port
					used[port] = true
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
	genesis.ChainID, genesis.InitialHeight, genesis.AppState = f.ChainID, 1, transport
	genesis.Validators = make([]cmttypes.GenesisValidator, 4)
	for i := range keys {
		genesis.Validators[i] = cmttypes.GenesisValidator{Address: keys[i].Address, PubKey: keys[i].PubKey, Power: 10, Name: fmt.Sprintf("validator-%d", i)}
	}
	if err := genesis.ValidateAndComplete(); err != nil {
		t.Fatal(err)
	}
	genesis.ConsensusParams.Block.MaxBytes = c.Network.MaxBlockBytes
	if err := genesis.ValidateAndComplete(); err != nil {
		t.Fatal(err)
	}
	var stopServers []func() error
	t.Cleanup(func() {
		for i, cmd := range pilot.observers {
			if cmd != nil && cmd.Process != nil {
				_ = cmd.Process.Signal(syscall.SIGTERM)
				_, _ = cmd.Process.Wait()
			}
			if pilot.observerLogs[i] != nil {
				_ = pilot.observerLogs[i].Close()
			}
			if pilot.observerErrs[i] != nil {
				_ = pilot.observerErrs[i].Close()
			}
		}
		n.stopAll()
		for _, stop := range stopServers {
			_ = stop()
		}
		for _, app := range pilot.apps {
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
		cfg.Mempool.Type = c.Network.MempoolType
		cfg.Mempool.Size = c.Network.MempoolSize
		cfg.Mempool.MaxTxsBytes = c.Network.MempoolMaxBytes
		cfg.Mempool.CacheSize = c.Network.MempoolCacheSize
		cfg.Mempool.MaxTxBytes = c.Network.MempoolMaxTxBytes
		cfg.Mempool.Recheck = c.Network.MempoolRecheck
		cfg.Mempool.Broadcast = c.Network.MempoolBroadcast
		cfg.Consensus.CreateEmptyBlocks = c.Network.CreateEmptyBlocks
		cfg.Consensus.CreateEmptyBlocksInterval = time.Duration(c.Network.EmptyBlockIntervalMS) * time.Millisecond
		cfg.Consensus.TimeoutCommit = time.Duration(c.Network.TimeoutCommitMS) * time.Millisecond
		cfg.P2P.AllowDuplicateIP = true
		cfg.P2P.AddrBookStrict = false
		cfg.Instrumentation.Prometheus = true
		cfg.Instrumentation.PrometheusListenAddr = fmt.Sprintf("127.0.0.1:%d", pilot.metricsPorts[i])
		var peers []string
		for j := range n.nodes {
			if j != i {
				peers = append(peers, fmt.Sprintf("%s@127.0.0.1:%d", ids[j], n.nodes[j].ports.P2P))
			}
		}
		cfg.P2P.PersistentPeers = strings.Join(peers, ",")
		config.WriteConfigFile(filepath.Join(home, "config", "config.toml"), cfg)
		dir := filepath.Join(home, "data", "pluto-profiles")
		store, err := researchv2store.Create(dir, f.Genesis, f.ChainID, profile)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
		pilot.apps[i], err = researchv2app.Open(dir, f.Genesis, f.ChainID, profile)
		if err != nil {
			t.Fatal(err)
		}
		pilot.recorders[i] = telemetry.NewRecorder(2048)
		pilot.apps[i].SetRecorder(pilot.recorders[i])
		abciServer, err := server.NewServer(cfg.ProxyApp, "socket", pilot.apps[i])
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
	return pilot
}

func (p *s06Network) startObservers(t *testing.T, c load.Config) {
	t.Helper()
	observer, _ := filepath.Abs("../../build/profile-observer")
	for i := range p.n.nodes {
		outPath := filepath.Join(filepath.Dir(p.n.root), fmt.Sprintf("observer-%d.jsonl", i))
		errPath := filepath.Join(filepath.Dir(p.n.root), fmt.Sprintf("observer-%d.err", i))
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		stderr, err := os.OpenFile(errPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		n := &p.n.nodes[i]
		rpc := fmt.Sprintf("http://127.0.0.1:%d", n.ports.RPC)
		cmd := exec.Command(observer, "-rpc", rpc, "-metrics", fmt.Sprintf("http://127.0.0.1:%d", p.metricsPorts[i]),
			"-comet-pid", fmt.Sprint(n.pid), "-app-pid", fmt.Sprint(os.Getpid()), "-interval", fmt.Sprintf("%dms", c.Network.CollectorIntervalMS), "-count", "0")
		cmd.Stdout, cmd.Stderr = out, stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		p.observers[i], p.observerLogs[i], p.observerErrs[i] = cmd, out, stderr
		p.nodes[i] = load.WatchedNode{RPC: rpc, ObserverFile: outPath, ObserverPID: cmd.Process.Pid, CometPID: n.pid, AppPID: os.Getpid()}
	}
}

func TestS06IsolatedOpenLoopPilot(t *testing.T) {
	configPath, _ := filepath.Abs("../../research/config/s06_pilot.json")
	c, configHash, err := load.ReadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := load.NewPilotFixture("pluto-s06-pilot", 90)
	if err != nil {
		t.Fatal(err)
	}
	jobs := []struct {
		name     string
		profile  researchv2.Profile
		regime   string
		waveform string
		rate     int
	}{
		{"C-equal-bytes", researchv2.ProfileC, load.EqualByteRate, "constant", 0},
		{"CP-equal-bytes", researchv2.ProfileCP, load.EqualByteRate, "constant", 0},
		{"P-equal-tx", researchv2.ProfileP, load.EqualTxRate, "constant", 2},
		{"H-equal-tx", researchv2.ProfileH, load.EqualTxRate, "constant", 2},
		{"H-valid-high", researchv2.ProfileH, load.EqualTxRate, "constant", 6},
		{"H-fixed-plus-invalid", researchv2.ProfileH, load.FixedLegitimatePlusAdversary, "constant", 0},
		{"H-burst", researchv2.ProfileH, load.EqualTxRate, "burst", 2},
	}
	seed := int64(611)
	rand.New(rand.NewSource(seed)).Shuffle(len(jobs), func(i, j int) { jobs[i], jobs[j] = jobs[j], jobs[i] })
	for order, job := range jobs {
		t.Run(job.name, func(t *testing.T) {
			plan, err := load.BuildPilotPlan(fixture, job.profile, job.regime, job.waveform, fmt.Sprintf("s06-pilot-%02d-%s", order, job.name), seed, c, configHash, job.rate)
			if err != nil {
				t.Fatal(err)
			}
			pilot := startS06Network(t, c, fixture, job.profile)
			pilot.startObservers(t, c)
			root := filepath.Dir(pilot.n.root)
			planPath := filepath.Join(root, "plan.json")
			if err := load.WritePlan(planPath, plan, c, configHash); err != nil {
				t.Fatal(err)
			}
			genesisPath := filepath.Join(root, "genesis.bin")
			if err := os.WriteFile(genesisPath, fixture.Genesis, 0600); err != nil {
				t.Fatal(err)
			}
			nodesPath := filepath.Join(root, "nodes.json")
			nodeBytes, err := json.Marshal(pilot.nodes)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(nodesPath, nodeBytes, 0600); err != nil {
				t.Fatal(err)
			}
			killPath := filepath.Join(root, "STOP")
			deadline := time.Now().Add(10 * time.Second)
			for {
				ready := true
				for _, watched := range pilot.nodes {
					info, err := os.Stat(watched.ObserverFile)
					if err != nil || info.Size() == 0 {
						ready = false
						break
					}
				}
				if ready {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("observers never produced samples")
				}
				time.Sleep(100 * time.Millisecond)
			}
			loadBinary, _ := filepath.Abs("../../build/profile-load")
			resultPath := filepath.Join(root, "result.json")
			cmd := exec.Command(loadBinary, "-config", configPath, "-plan", planPath, "-genesis", genesisPath, "-nodes", nodesPath,
				"-rpc", pilot.nodes[0].RPC, "-data-root", pilot.n.root, "-kill-file", killPath, "-result", resultPath, "-allocated-cores", fmt.Sprint(runtime.NumCPU()))
			commandOutput, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("isolated load failed: %v output=%s", err, commandOutput)
			}
			var result load.Result
			rawResult, err := os.ReadFile(resultPath)
			if err != nil || json.Unmarshal(rawResult, &result) != nil || result.Status != load.StatusOK || result.ActualSendCount != len(plan.Events) || result.ConfigSHA256 != configHash || result.GenesisSHA256 != plan.GenesisSHA256 {
				t.Fatalf("invalid run result: err=%v status=%s planned=%d actual=%d", err, result.Status, len(plan.Events), result.ActualSendCount)
			}
			for _, item := range result.Events {
				if item.ActualOffsetNS == nil || item.ActualCode == nil || *item.ActualCode != item.ExpectedCode {
					t.Fatalf("offered or oracle mismatch for %s", item.ID)
				}
			}
			pilot.n.waitAll(8, 30*time.Second)
			var expectedReceiver uint64
			for _, e := range plan.Events {
				if e.Traffic == "legitimate" {
					expectedReceiver++
				}
			}
			for i := range pilot.n.nodes {
				receiverBalance, receiverNonce := s04Account(t, pilot.n, i, fixture.Receiver.Binding.Account)
				if receiverBalance != expectedReceiver || receiverNonce != 0 {
					t.Fatalf("validator %d receiver state %d/%d want %d/0", i, receiverBalance, receiverNonce, expectedReceiver)
				}
				for _, e := range plan.Events {
					if e.Traffic != "legitimate" {
						continue
					}
					balance, nonce := s04Account(t, pilot.n, i, fixture.Senders[e.AccountSlot].Binding.Account)
					if balance != 1 || nonce != 1 {
						t.Fatalf("validator %d sender slot %d state %d/%d", i, e.AccountSlot, balance, nonce)
					}
				}
			}
			var p2pSeen bool
			for _, watched := range pilot.nodes {
				raw, err := os.ReadFile(watched.ObserverFile)
				if err != nil {
					t.Fatal(err)
				}
				for _, line := range bytes.Split(raw, []byte{'\n'}) {
					var sample telemetry.NodeSample
					if json.Unmarshal(line, &sample) == nil && sample.P2PIngressTxTotal != nil && *sample.P2PIngressTxTotal > 0 {
						p2pSeen = true
					}
				}
			}
			if !p2pSeen {
				t.Fatal("controlled RPC ingress never produced separately observed P2P relay")
			}
			h := sha256.Sum256(rawResult)
			t.Logf("pilot order=%d seed=%d profile=%d regime=%s waveform=%s planned=%d actual=%d legitimate_committed=%d result_sha256=%s", order, seed, job.profile, job.regime, job.waveform, len(plan.Events), result.ActualSendCount, expectedReceiver, hex.EncodeToString(h[:]))
		})
	}
}
