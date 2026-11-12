// Package node manages a local CometBFT process and Pluto ABCI socket server.
package node

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/cometbft/cometbft/abci/server"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/types"

	"example.com/pluto/internal/classical/app"
	"example.com/pluto/internal/classical/genesis"
	"example.com/pluto/internal/classical/state"
	"example.com/pluto/internal/classical/store"
	"example.com/pluto/internal/legacy/app"
	"example.com/pluto/internal/legacy/store"
)

const ProfileB0 = "b0-v2"
const ProfileB1 = "b1-hybrid-v1"

type Config struct {
	ChainID       string `json:"chain_id"`
	ABCIAddress   string `json:"abci_address"`
	RPCAddress    string `json:"rpc_address"`
	P2PAddress    string `json:"p2p_address"`
	Profile       string `json:"profile,omitempty"`
	GenesisSHA256 string `json:"genesis_sha256,omitempty"`
	// B1 binds this home's validator identity to one specific genesis member.
	// It is omitted in accepted Phase 02/B0 metadata bytes.
	ValidatorAddress string `json:"validator_address,omitempty"`
}

func address(port int) (string, error) {
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("invalid port %d", port)
	}
	return "tcp://127.0.0.1:" + strconv.Itoa(port), nil
}

func Init(home, cometBinary, chainID string, rpcPort, p2pPort, abciPort int, output io.Writer) error {
	return initNode(home, cometBinary, chainID, rpcPort, p2pPort, abciPort, nil, "", "", nil, nil, output)
}

// InitB0 explicitly creates a fresh B0 profile. The source binary is parsed
// before touching the home; CometBFT receives only D-025 JSON/Base64 bytes.
func InitB0(home, cometBinary, chainID string, binaryGenesis []byte, rpcPort, p2pPort, abciPort int, output io.Writer) error {
	return initB0WithHooks(home, cometBinary, chainID, binaryGenesis, rpcPort, p2pPort, abciPort, nil, output)
}

// b0InitHooks injects only disposable test failures. Production passes nil.
type b0InitHooks struct {
	BeforeHome, AfterHome, AfterCometInit, BeforeStore, AfterStore, BeforeMetadata, AfterMetadata, BeforeReturn func() error
	StoreOptions                                                                                                b0store.Options
}

func initB0WithHooks(home, cometBinary, chainID string, binaryGenesis []byte, rpcPort, p2pPort, abciPort int, hooks *b0InitHooks, output io.Writer) error {
	transport, err := b0genesis.Encode(binaryGenesis, chainID)
	if err != nil {
		return err
	}
	decoded, _, err := b0genesis.Decode(transport, chainID)
	if err != nil || !bytes.Equal(decoded, binaryGenesis) {
		return b0genesis.ErrTransport
	}
	digest := sha256.Sum256(binaryGenesis)
	return initNode(home, cometBinary, chainID, rpcPort, p2pPort, abciPort, transport, ProfileB0, hex.EncodeToString(digest[:]), binaryGenesis, hooks, output)
}

func initNode(home, cometBinary, chainID string, rpcPort, p2pPort, abciPort int, transport []byte, profile, genesisSHA256 string, binaryGenesis []byte, hooks *b0InitHooks, output io.Writer) error {
	if home == "" || cometBinary == "" {
		return errors.New("home and bundled CometBFT binary are required")
	}
	if _, err := store.Genesis(chainID); err != nil {
		return fmt.Errorf("invalid chain ID: %w", err)
	}
	if rpcPort == p2pPort || rpcPort == abciPort || p2pPort == abciPort {
		return errors.New("RPC, P2P, and ABCI ports must differ")
	}
	rpcAddr, err := address(rpcPort)
	if err != nil {
		return err
	}
	p2pAddr, err := address(p2pPort)
	if err != nil {
		return err
	}
	abciAddr, err := address(abciPort)
	if err != nil {
		return err
	}
	if _, err := os.Stat(home); err == nil {
		return fmt.Errorf("node home already exists: %s", home)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect node home: %w", err)
	}
	if err := runB0InitHook(hooks, func(h *b0InitHooks) func() error { return h.BeforeHome }, "before home creation"); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(home), 0700); err != nil {
		return fmt.Errorf("create node home parent: %w", err)
	}
	// The final directory must be claimed atomically. Both callers may pass
	// the preliminary Stat above, but only one can pass this exclusive Mkdir.
	if err := os.Mkdir(home, 0700); errors.Is(err, os.ErrExist) {
		return fmt.Errorf("node home already exists: %s", home)
	} else if err != nil {
		return fmt.Errorf("create node home: %w", err)
	}
	if err := runB0InitHook(hooks, func(h *b0InitHooks) func() error { return h.AfterHome }, "after home creation"); err != nil {
		return err
	}
	cmd := exec.Command(cometBinary, "init", "--home", home)
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("initialize CometBFT files: %w", err)
	}
	if err := runB0InitHook(hooks, func(h *b0InitHooks) func() error { return h.AfterCometInit }, "after CometBFT configuration creation"); err != nil {
		return err
	}
	genesisFile := filepath.Join(home, "config", "genesis.json")
	genesis, err := types.GenesisDocFromFile(genesisFile)
	if err != nil {
		return fmt.Errorf("read generated genesis: %w", err)
	}
	genesis.ChainID = chainID
	genesis.InitialHeight = 1
	genesis.AppState = transport
	if err := genesis.ValidateAndComplete(); err != nil {
		return fmt.Errorf("validate generated genesis: %w", err)
	}
	if err := genesis.SaveAs(genesisFile); err != nil {
		return fmt.Errorf("write generated genesis: %w", err)
	}
	config := cfg.DefaultConfig()
	config.SetRoot(home)
	config.ProxyApp = abciAddr
	config.RPC.ListenAddress = rpcAddr
	config.P2P.ListenAddress = p2pAddr
	config.Consensus.CreateEmptyBlocks = true
	config.Consensus.CreateEmptyBlocksInterval = time.Second
	cfg.WriteConfigFile(filepath.Join(home, "config", "config.toml"), config)
	if profile == ProfileB0 {
		if err := runB0InitHook(hooks, func(h *b0InitHooks) func() error { return h.BeforeStore }, "before B0 snapshot creation"); err != nil {
			return err
		}
		options := b0store.Options{}
		if hooks != nil {
			options = hooks.StoreOptions
		}
		dbDir := filepath.Join(home, "data", "pluto-b0-v2")
		created, err := b0store.CreateWithOptions(dbDir, binaryGenesis, chainID, options)
		if err != nil {
			return fmt.Errorf("create B0 H0 snapshot: %w", err)
		}
		if err := created.Close(); err != nil {
			return fmt.Errorf("close B0 H0 snapshot: %w", err)
		}
		if err := runB0InitHook(hooks, func(h *b0InitHooks) func() error { return h.AfterStore }, "after B0 snapshot sync"); err != nil {
			return err
		}
		if err := verifyB0H0(dbDir, chainID, binaryGenesis); err != nil {
			return err
		}
		if err := runB0InitHook(hooks, func(h *b0InitHooks) func() error { return h.BeforeMetadata }, "before Pluto metadata finalization"); err != nil {
			return err
		}
	}
	metadata := Config{ChainID: chainID, ABCIAddress: abciAddr, RPCAddress: rpcAddr, P2PAddress: p2pAddr, Profile: profile, GenesisSHA256: genesisSHA256}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(home, "config", "pluto.json"), append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("write Pluto node metadata: %w", err)
	}
	if profile == ProfileB0 {
		if err := runB0InitHook(hooks, func(h *b0InitHooks) func() error { return h.AfterMetadata }, "after Pluto metadata write"); err != nil {
			return err
		}
		if err := verifyB0Home(home, chainID, transport, binaryGenesis, metadata); err != nil {
			return err
		}
		if err := syncInitArtifacts(home); err != nil {
			return err
		}
		if err := runB0InitHook(hooks, func(h *b0InitHooks) func() error { return h.BeforeReturn }, "after initialization sync before response"); err != nil {
			return err
		}
	}
	return nil
}

func runB0InitHook(hooks *b0InitHooks, selectHook func(*b0InitHooks) func() error, stage string) error {
	if hooks == nil {
		return nil
	}
	hook := selectHook(hooks)
	if hook == nil {
		return nil
	}
	if err := hook(); err != nil {
		return fmt.Errorf("D-026 initialization %s: %w", stage, err)
	}
	return nil
}

func verifyB0H0(dbDir, chainID string, genesis []byte) error {
	expected, err := b0state.ParseGenesis(genesis, chainID)
	if err != nil {
		return fmt.Errorf("validate B0 Genesis V2: %w", err)
	}
	a, err := b0app.Open(dbDir, chainID, genesis)
	if err != nil {
		return fmt.Errorf("reopen B0 H0 snapshot: %w", err)
	}
	state, exists, readErr := a.Committed()
	closeErr := a.Close()
	if readErr != nil {
		return fmt.Errorf("read B0 H0 snapshot: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close verified B0 H0 snapshot: %w", closeErr)
	}
	if !exists || state.Height() != 0 || state.AppHash() != expected.AppHash() ||
		state.TotalSupply() != expected.TotalSupply() || !slices.Equal(state.Accounts(), expected.Accounts()) {
		return errors.New("B0 H0 snapshot does not match canonical Genesis V2")
	}
	return nil
}

func verifyB0Home(home, chainID string, transport, genesis []byte, expected Config) error {
	actual, err := ReadConfig(home)
	if err != nil {
		return fmt.Errorf("verify Pluto metadata: %w", err)
	}
	if actual != expected {
		return errors.New("Pluto metadata does not match initialized B0 profile")
	}
	doc, err := types.GenesisDocFromFile(filepath.Join(home, "config", "genesis.json"))
	if err != nil {
		return fmt.Errorf("verify CometBFT genesis: %w", err)
	}
	if doc.ChainID != chainID || doc.InitialHeight != 1 || !bytes.Equal(doc.AppState, transport) {
		return errors.New("CometBFT genesis does not match canonical B0 transport")
	}
	if err := verifyCometInitArtifacts(home, expected, doc); err != nil {
		return err
	}
	if err := verifyB0H0(filepath.Join(home, "data", "pluto-b0-v2"), chainID, genesis); err != nil {
		return err
	}
	return nil
}

func ReadConfig(home string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(home, "config", "pluto.json"))
	if err != nil {
		return Config{}, fmt.Errorf("read Pluto node metadata: %w", err)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("decode Pluto node metadata: %w", err)
	}
	if _, err := store.Genesis(config.ChainID); err != nil {
		return Config{}, fmt.Errorf("invalid Pluto node metadata: %w", err)
	}
	if config.Profile != "" && config.Profile != ProfileB0 && config.Profile != ProfileB1 {
		return Config{}, errors.New("unknown Pluto runtime profile")
	}
	if config.Profile == ProfileB0 || config.Profile == ProfileB1 {
		digest, err := hex.DecodeString(config.GenesisSHA256)
		if err != nil || len(digest) != sha256.Size {
			if config.Profile == ProfileB1 {
				return Config{}, errors.New("invalid B1 genesis digest in Pluto metadata")
			}
			return Config{}, errors.New("invalid B0 genesis digest in Pluto metadata")
		}
	}
	if config.Profile == ProfileB1 {
		address, err := hex.DecodeString(config.ValidatorAddress)
		if err != nil || len(address) != 20 || hex.EncodeToString(address) != config.ValidatorAddress {
			return Config{}, errors.New("invalid B1 validator address binding in Pluto metadata")
		}
	}
	return config, nil
}

func State(home string) (*store.State, error) {
	config, err := ReadConfig(home)
	if err != nil {
		return nil, err
	}
	if config.Profile == ProfileB0 || config.Profile == ProfileB1 {
		return nil, errors.New("Phase 02 state command is unavailable for B0/B1 profile")
	}
	dbDir := filepath.Join(home, "data", "pluto")
	if _, err := os.Stat(dbDir); err != nil {
		return nil, fmt.Errorf("application database is absent: %w", err)
	}
	db, err := store.Open(dbDir)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	state, err := db.Load()
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, errors.New("application state has not been initialized")
	}
	return state, nil
}

// Run starts the Pluto socket server before launching the pinned CometBFT
// executable. Cancellation terminates only the child process started here.
func Run(ctx context.Context, home, cometBinary string, output io.Writer) error {
	config, err := ReadConfig(home)
	if err != nil {
		return err
	}
	if config.Profile == ProfileB0 {
		return runB0(ctx, home, cometBinary, config, output)
	}
	if config.Profile == ProfileB1 {
		return runB1(ctx, home, cometBinary, config, output)
	}
	genesis, err := types.GenesisDocFromFile(filepath.Join(home, "config", "genesis.json"))
	if err != nil {
		return fmt.Errorf("read CometBFT genesis: %w", err)
	}
	if genesis.ChainID != config.ChainID || genesis.InitialHeight != 1 || len(genesis.AppState) != 0 {
		return errors.New("CometBFT genesis conflicts with Pluto node metadata")
	}
	dbDir := filepath.Join(home, "data", "pluto")
	if _, err := os.Stat(dbDir); errors.Is(err, os.ErrNotExist) {
		if _, cometErr := os.Stat(filepath.Join(home, "data", "state.db")); cometErr == nil {
			return errors.New("Pluto database is absent while CometBFT state exists; refusing implicit reset")
		}
	} else if err != nil {
		return fmt.Errorf("inspect Pluto database: %w", err)
	}
	a, err := app.Open(dbDir)
	if err != nil {
		return err
	}
	defer a.Close()
	if a.Committed() == nil {
		if _, cometErr := os.Stat(filepath.Join(home, "data", "state.db")); cometErr == nil {
			return errors.New("Pluto committed state is absent while CometBFT state exists")
		}
	}
	if state := a.Committed(); state != nil && state.ChainID != config.ChainID {
		return errors.New("persisted application chain ID conflicts with genesis")
	}
	srv, err := server.NewServer(config.ABCIAddress, "socket", a)
	if err != nil {
		return err
	}
	if err := srv.Start(); err != nil {
		return fmt.Errorf("start ABCI server: %w", err)
	}
	defer srv.Stop()
	cmd := exec.Command(cometBinary, "node", "--home", home)
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start CometBFT: %w", err)
	}
	result := make(chan error, 1)
	go func() { result <- cmd.Wait() }()
	select {
	case err := <-result:
		return fmt.Errorf("CometBFT exited: %w", err)
	case <-ctx.Done():
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("signal CometBFT: %w", err)
		}
		select {
		case err := <-result:
			if err != nil {
				return fmt.Errorf("stop CometBFT: %w", err)
			}
			return nil
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
			<-result
			return errors.New("CometBFT did not stop within 10 seconds")
		}
	}
}
