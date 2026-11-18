//go:build integration

package integration

import (
	"context"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"example.com/pluto/internal/classical/app"
	"example.com/pluto/internal/classical/state"
	"github.com/cockroachdb/pebble"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	sm "github.com/cometbft/cometbft/state"
	blockstore "github.com/cometbft/cometbft/store"
)

func TestB0PinnedHeightAndHashMismatchFailClosed(t *testing.T) {
	for _, mode := range []string{"app-ahead", "same-height-different-hash", "consensus-ahead-blockstore", "consensus-hash-tamper", "app-behind-unavailable", "app-behind-pruned", "replay-hash-conflict"} {
		t.Run(mode, func(t *testing.T) {
			plutod, _ := filepath.Abs("../../build/plutod")
			home := filepath.Join(t.TempDir(), "home")
			genesisFile := filepath.Join(t.TempDir(), "genesis.bin")
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
			args := []string{"init-b0", "--home", home, "--chain-id", "pluto-b0-v1", "--genesis-file", genesisFile,
				"--rpc-port", strconv.Itoa(rpc), "--p2p-port", strconv.Itoa(p2p), "--abci-port", strconv.Itoa(abciPort)}
			if out, err := exec.Command(plutod, args...).CombinedOutput(); err != nil {
				t.Fatalf("init: %v: %s", err, out)
			}
			logFile, err := os.Create(filepath.Join(t.TempDir(), "first.log"))
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(plutod, "start", "--home", home)
			cmd.Stdout, cmd.Stderr = logFile, logFile
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
				_ = logFile.Close()
			})
			b0WaitHeight(t, rpc, 2)
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Wait(); err != nil {
				t.Fatal(err)
			}
			_ = logFile.Close()
			dbDir := filepath.Join(home, "data", "pluto-b0-v2")
			a, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
			if err != nil {
				t.Fatal(err)
			}
			state, _, err := a.Committed()
			if err != nil {
				t.Fatal(err)
			}
			beforeHeight := state.Height()
			if err := a.Close(); err != nil {
				t.Fatal(err)
			}
			blockHeight, consensusHeight := cometHeights(t, home)
			injectedA, injectedHash := beforeHeight, state.AppHash()
			if blockHeight != beforeHeight || consensusHeight != beforeHeight {
				t.Fatalf("initial mismatch B=%d C=%d A=%d", blockHeight, consensusHeight, beforeHeight)
			}
			if mode == "app-ahead" {
				a, err = b0app.Open(dbDir, "pluto-b0-v1", genesis)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := a.FinalizeBlock(context.Background(), &abci.RequestFinalizeBlock{Height: beforeHeight + 1}); err != nil {
					t.Fatal(err)
				}
				if _, err := a.Commit(context.Background(), &abci.RequestCommit{}); err != nil {
					t.Fatal(err)
				}
				advanced, _, err := a.Committed()
				if err != nil {
					t.Fatal(err)
				}
				injectedHash = advanced.AppHash()
				if err := a.Close(); err != nil {
					t.Fatal(err)
				}
				injectedA = beforeHeight + 1
			} else if mode == "same-height-different-hash" || mode == "app-behind-unavailable" || mode == "app-behind-pruned" || mode == "replay-hash-conflict" {
				var sender, receiver b0state.Address
				ab, _ := hex.DecodeString("b9631c0f8d9d2d9d56bd6151559f3339e14eb36a")
				bb, _ := hex.DecodeString("e4affb78013f3a929c0e667d632e6fc9fc09732c")
				copy(sender[:], ab)
				copy(receiver[:], bb)
				alteredHeight := beforeHeight
				if mode == "app-behind-unavailable" || mode == "replay-hash-conflict" {
					alteredHeight = beforeHeight - 1
				}
				altered, err := b0state.NewState("pluto-b0-v1", alteredHeight, 100, []b0state.Account{{Address: sender, Balance: 99}, {Address: receiver, Balance: 1}})
				if mode == "app-behind-pruned" {
					altered, err = b0state.ParseGenesis(genesis, "pluto-b0-v1")
					alteredHeight = 0
				}
				if err != nil {
					t.Fatal(err)
				}
				logical, err := altered.LogicalBytes()
				if err != nil {
					t.Fatal(err)
				}
				hash := altered.AppHash()
				injectedA, injectedHash = alteredHeight, hash
				record := append(append([]byte("PLUTOS2!"), logical...), hash[:]...)
				db, err := pebble.Open(dbDir, &pebble.Options{ErrorIfNotExists: true})
				if err != nil {
					t.Fatal(err)
				}
				if err := db.Set([]byte("pluto/committed-state/v2"), record, pebble.Sync); err != nil {
					t.Fatal(err)
				}
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
				// The alternate record is internally canonical and integrity-valid;
				// only the external Comet state/history can detect the mismatch.
				probe, err := b0app.Open(dbDir, "pluto-b0-v1", genesis)
				if err != nil {
					t.Fatalf("valid alternate record rejected before Comet handshake: %v", err)
				}
				if err := probe.Close(); err != nil {
					t.Fatal(err)
				}
				if mode == "app-behind-unavailable" {
					blockDB, err := dbm.NewDB("blockstore", dbm.GoLevelDBBackend, filepath.Join(home, "data"))
					if err != nil {
						t.Fatal(err)
					}
					// Remove the latest block while retaining C. This gives A<C
					// with missing history and must never silently reinitialize.
					bs := blockstore.NewBlockStore(blockDB)
					if err := bs.DeleteLatestBlock(); err != nil {
						t.Fatal(err)
					}
					if err := blockDB.Close(); err != nil {
						t.Fatal(err)
					}
				} else if mode == "app-behind-pruned" {
					stateDB, err := dbm.NewDB("state", dbm.GoLevelDBBackend, filepath.Join(home, "data"))
					if err != nil {
						t.Fatal(err)
					}
					consensus, err := sm.NewStore(stateDB, sm.StoreOptions{}).Load()
					if err != nil {
						t.Fatal(err)
					}
					if err := stateDB.Close(); err != nil {
						t.Fatal(err)
					}
					blockDB, err := dbm.NewDB("blockstore", dbm.GoLevelDBBackend, filepath.Join(home, "data"))
					if err != nil {
						t.Fatal(err)
					}
					bs := blockstore.NewBlockStore(blockDB)
					pruned, _, err := bs.PruneBlocks(beforeHeight, consensus)
					if err != nil || pruned != 1 || bs.Base() != beforeHeight {
						t.Fatalf("prune failed count=%d base=%d err=%v", pruned, bs.Base(), err)
					}
					if err := blockDB.Close(); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				stateDB, err := dbm.NewDB("state", dbm.GoLevelDBBackend, filepath.Join(home, "data"))
				if err != nil {
					t.Fatal(err)
				}
				store := sm.NewStore(stateDB, sm.StoreOptions{})
				consensus, err := store.Load()
				if err != nil {
					t.Fatal(err)
				}
				if mode == "consensus-ahead-blockstore" {
					consensus.LastBlockHeight++
				} else {
					consensus.AppHash = append([]byte(nil), consensus.AppHash...)
					consensus.AppHash[0] ^= 1
				}
				if err := store.Save(consensus); err != nil {
					t.Fatal(err)
				}
				if err := stateDB.Close(); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			out, probeErr := exec.CommandContext(ctx, plutod, "start", "--home", home).CombinedOutput()
			if ctx.Err() != nil || probeErr == nil {
				t.Fatalf("mismatch did not fail closed: mode=%s context=%v err=%v output=%s", mode, ctx.Err(), probeErr, out)
			}
			if mode == "app-ahead" && !strings.Contains(string(out), "higher than core") {
				t.Fatalf("wrong app-ahead failure: %s", out)
			}
			if mode == "same-height-different-hash" {
				if !strings.Contains(string(out), "state.AppHash does not match AppHash after replay") {
					t.Fatalf("wrong hash mismatch failure: %s", out)
				}
			}
			expected := map[string]string{
				"consensus-ahead-blockstore": "StateBlockHeight (3) > StoreBlockHeight (2)",
				"consensus-hash-tamper":      "state.AppHash does not match AppHash after replay",
				"app-behind-unavailable":     "StateBlockHeight (2) > StoreBlockHeight (1)",
				"app-behind-pruned":          "app block height (0) is too far below block store base (2)",
				"replay-hash-conflict":       "state.AppHash does not match AppHash after replay",
			}
			if expected[mode] != "" && !strings.Contains(string(out), expected[mode]) {
				t.Fatalf("wrong %s failure: %s", mode, out)
			}
			if artifactDir := os.Getenv("PLUTO_RECOVERY_ARTIFACTS"); artifactDir != "" {
				if err := os.MkdirAll(artifactDir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(artifactDir, "mismatch-"+mode+".log"), out, 0600); err != nil {
					t.Fatal(err)
				}
			}
			finalB, finalC := cometHeights(t, home)
			t.Logf("%s: real Comet refused restart; original B=%d C=%d A=%d hash=%x sender=100/0 receiver=absent; injected B=%d C=%d A=%d hash=%x; process exit=%v output=%q", mode, blockHeight, consensusHeight, beforeHeight, state.AppHash(), finalB, finalC, injectedA, injectedHash, probeErr, out)
		})
	}
}
