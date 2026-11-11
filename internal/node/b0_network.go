package node

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cometbft/cometbft/config"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/types"

	"example.com/pluto/internal/classical/genesis"
	"example.com/pluto/internal/classical/store"
	"example.com/pluto/internal/legacy/store"
)

// B0Ports assigns independent localhost endpoints to one validator.
type B0Ports struct{ RPC, P2P, ABCI int }

// InitB0FourValidators exclusively claims a fresh root, generates four
// independent Comet identities, and only then constructs their shared genesis.
// An interrupted root is retained for inspection and cannot be initialized
// again implicitly. Production callers must keep all generated keys private.
func InitB0FourValidators(root, cometBinary, chainID string, binaryGenesis []byte, ports [4]B0Ports, output io.Writer) error {
	if root == "" || cometBinary == "" {
		return errors.New("network root and bundled CometBFT binary are required")
	}
	if _, err := store.Genesis(chainID); err != nil {
		return fmt.Errorf("invalid chain ID: %w", err)
	}
	transport, err := b0genesis.Encode(binaryGenesis, chainID)
	if err != nil {
		return err
	}
	decoded, _, err := b0genesis.Decode(transport, chainID)
	if err != nil || !bytes.Equal(decoded, binaryGenesis) {
		return b0genesis.ErrTransport
	}
	usedPorts := make(map[int]bool, 12)
	for _, triple := range ports {
		for _, port := range []int{triple.RPC, triple.P2P, triple.ABCI} {
			if _, err := address(port); err != nil {
				return err
			}
			if usedPorts[port] {
				return fmt.Errorf("duplicate four-validator port %d", port)
			}
			usedPorts[port] = true
		}
	}
	if _, err := os.Stat(root); err == nil {
		return fmt.Errorf("four-validator root already exists: %s", root)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect four-validator root: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(root), 0700); err != nil {
		return fmt.Errorf("create four-validator root parent: %w", err)
	}
	if err := os.Mkdir(root, 0700); errors.Is(err, os.ErrExist) {
		return fmt.Errorf("four-validator root already exists: %s", root)
	} else if err != nil {
		return fmt.Errorf("claim four-validator root: %w", err)
	}
	var homes [4]string
	var keys [4]privval.FilePVKey
	var nodeIDs [4]string
	for i := range homes {
		homes[i] = filepath.Join(root, fmt.Sprintf("validator-%d", i))
		if err := os.Mkdir(homes[i], 0700); err != nil {
			return fmt.Errorf("create validator home %d: %w", i, err)
		}
		cmd := exec.Command(cometBinary, "init", "--home", homes[i])
		cmd.Stdout, cmd.Stderr = output, output
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("generate validator identity %d: %w", i, err)
		}
		raw, err := os.ReadFile(filepath.Join(homes[i], "config", "priv_validator_key.json"))
		if err != nil {
			return fmt.Errorf("read validator identity %d: %w", i, err)
		}
		if err := cmtjson.Unmarshal(raw, &keys[i]); err != nil || keys[i].PrivKey == nil || !validCometEd25519Key(keys[i].PrivKey) {
			return fmt.Errorf("invalid generated validator identity %d: %v", i, err)
		}
		nodeKey, err := p2p.LoadNodeKey(filepath.Join(homes[i], "config", "node_key.json"))
		if err != nil || nodeKey == nil || nodeKey.PrivKey == nil || !validCometEd25519Key(nodeKey.PrivKey) {
			return fmt.Errorf("invalid generated P2P identity %d: %v", i, err)
		}
		nodeIDs[i] = string(nodeKey.ID())
		for j := 0; j < i; j++ {
			if bytes.Equal(keys[i].Address, keys[j].Address) || nodeIDs[i] == nodeIDs[j] {
				return errors.New("generated validator or P2P identities are duplicated")
			}
		}
	}
	genesis, err := types.GenesisDocFromFile(filepath.Join(homes[0], "config", "genesis.json"))
	if err != nil {
		return fmt.Errorf("read generated genesis: %w", err)
	}
	genesis.ChainID, genesis.InitialHeight, genesis.AppState = chainID, 1, transport
	genesis.Validators = make([]types.GenesisValidator, 4)
	for i := range keys {
		genesis.Validators[i] = types.GenesisValidator{Address: keys[i].Address, PubKey: keys[i].PubKey, Power: 10, Name: fmt.Sprintf("validator-%d", i)}
	}
	if err := genesis.ValidateAndComplete(); err != nil {
		return fmt.Errorf("validate shared four-validator genesis: %w", err)
	}
	digest := sha256.Sum256(binaryGenesis)
	for i, home := range homes {
		if err := genesis.SaveAs(filepath.Join(home, "config", "genesis.json")); err != nil {
			return fmt.Errorf("write shared genesis for validator %d: %w", i, err)
		}
		rpcAddr, _ := address(ports[i].RPC)
		p2pAddr, _ := address(ports[i].P2P)
		abciAddr, _ := address(ports[i].ABCI)
		cfg := config.DefaultConfig()
		cfg.SetRoot(home)
		cfg.ProxyApp, cfg.RPC.ListenAddress, cfg.P2P.ListenAddress = abciAddr, rpcAddr, p2pAddr
		cfg.Consensus.CreateEmptyBlocks = true
		cfg.Consensus.CreateEmptyBlocksInterval = time.Second
		cfg.P2P.AllowDuplicateIP = true // Four test validators share localhost.
		cfg.P2P.AddrBookStrict = false
		var peers []string
		for j := range homes {
			if i != j {
				peers = append(peers, fmt.Sprintf("%s@127.0.0.1:%d", nodeIDs[j], ports[j].P2P))
			}
		}
		cfg.P2P.PersistentPeers = strings.Join(peers, ",")
		config.WriteConfigFile(filepath.Join(home, "config", "config.toml"), cfg)
		dbDir := filepath.Join(home, "data", "pluto-b0-v2")
		committed, err := b0store.Create(dbDir, binaryGenesis, chainID)
		if err != nil {
			return fmt.Errorf("create validator %d B0 H0: %w", i, err)
		}
		if err := committed.Close(); err != nil {
			return fmt.Errorf("close validator %d B0 H0: %w", i, err)
		}
		metadata := Config{ChainID: chainID, ABCIAddress: abciAddr, RPCAddress: rpcAddr, P2PAddress: p2pAddr, Profile: ProfileB0, GenesisSHA256: hex.EncodeToString(digest[:])}
		data, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(home, "config", "pluto.json"), append(data, '\n'), 0600); err != nil {
			return fmt.Errorf("write validator %d metadata: %w", i, err)
		}
		if err := verifyB0Home(home, chainID, transport, binaryGenesis, metadata); err != nil {
			return fmt.Errorf("verify validator %d H0: %w", i, err)
		}
		if err := syncInitArtifacts(home); err != nil {
			return fmt.Errorf("sync validator %d initialization: %w", i, err)
		}
	}
	if err := syncInitPath(filepath.Dir(root)); err != nil {
		return fmt.Errorf("sync four-validator root parent: %w", err)
	}
	return nil
}
