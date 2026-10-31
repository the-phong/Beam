package node

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cometbft/cometbft/abci/server"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/types"

	"example.com/pluto/internal/b1app"
	"example.com/pluto/internal/b1genesis"
	"example.com/pluto/internal/b1state"
	"example.com/pluto/internal/b1store"
)

func verifyB1Home(home, chainID string, transport, binary []byte, expected Config) error {
	actual, err := ReadConfig(home)
	if err != nil {
		return fmt.Errorf("verify B1 metadata: %w", err)
	}
	if actual != expected {
		return errors.New("Pluto metadata does not match initialized B1 profile")
	}
	doc, err := types.GenesisDocFromFile(filepath.Join(home, "config", "genesis.json"))
	if err != nil {
		return fmt.Errorf("verify B1 CometBFT genesis: %w", err)
	}
	if doc.ChainID != chainID || doc.InitialHeight != 1 || !bytes.Equal(doc.AppState, transport) {
		return errors.New("CometBFT genesis does not match canonical B1 transport")
	}
	if err := verifyCometInitArtifacts(home, expected, doc); err != nil {
		return err
	}
	if err := verifyB1ValidatorBinding(home, expected); err != nil {
		return err
	}
	state, err := b1store.ProbeExisting(filepath.Join(home, "data", "pluto-b1-v1"), chainID, binary)
	if err != nil {
		return fmt.Errorf("verify B1 H0 snapshot: %w", err)
	}
	expectedState, err := b1state.ParseGenesis(binary, chainID)
	if err != nil || state.Height() != 0 || state.AppHash() != expectedState.AppHash() {
		return errors.New("B1 H0 snapshot conflicts with canonical genesis")
	}
	return nil
}

// runB1 requires a complete B1 record before starting the ABCI server. It
// neither creates H0 nor falls back to B0 or the Phase 02 application.
func runB1(ctx context.Context, home, cometBinary string, config Config, output io.Writer) error {
	genesis, err := types.GenesisDocFromFile(filepath.Join(home, "config", "genesis.json"))
	if err != nil {
		return fmt.Errorf("read B1 CometBFT genesis: %w", err)
	}
	if genesis.ChainID != config.ChainID || genesis.InitialHeight != 1 {
		return errors.New("B1 genesis conflicts with Pluto metadata")
	}
	binary, _, err := b1genesis.Decode(genesis.AppState, config.ChainID)
	if err != nil {
		return fmt.Errorf("decode B1 genesis transport: %w", err)
	}
	digest := sha256.Sum256(binary)
	if hex.EncodeToString(digest[:]) != config.GenesisSHA256 {
		return errors.New("B1 genesis digest conflicts with Pluto metadata")
	}
	for _, foreign := range []string{"pluto", "pluto-b0-v2"} {
		if _, err := os.Stat(filepath.Join(home, "data", foreign)); err == nil {
			return errors.New("foreign application database present; refusing B1 profile")
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect foreign application database: %w", err)
		}
	}
	dbDir := filepath.Join(home, "data", "pluto-b1-v1")
	state, err := b1store.ProbeExisting(dbDir, config.ChainID, binary)
	if err != nil {
		return fmt.Errorf("B1 snapshot missing or invalid; start requires an existing B1 store: %w", err)
	}
	expected, err := b1state.ParseGenesis(binary, config.ChainID)
	if err != nil {
		return err
	}
	if state.TotalSupply() != expected.TotalSupply() || (state.Height() == 0 && state.AppHash() != expected.AppHash()) {
		return b1app.ErrGenesisConflict
	}
	if err := verifyCometRunArtifacts(home, config, genesis); err != nil {
		return fmt.Errorf("validate B1 CometBFT identity and configuration: %w", err)
	}
	if err := verifyB1ValidatorBinding(home, config); err != nil {
		return err
	}
	a, err := b1app.Open(dbDir, config.ChainID, binary)
	if err != nil {
		return fmt.Errorf("open existing B1 snapshot: %w", err)
	}
	defer a.Close()
	srv, err := server.NewServer(config.ABCIAddress, "socket", a)
	if err != nil {
		return err
	}
	if err := srv.Start(); err != nil {
		return fmt.Errorf("start B1 ABCI server: %w", err)
	}
	defer srv.Stop()
	cmd := exec.Command(cometBinary, "node", "--home", home)
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start B1 CometBFT: %w", err)
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

func verifyB1ValidatorBinding(home string, config Config) error {
	raw, err := os.ReadFile(filepath.Join(home, "config", "priv_validator_key.json"))
	if err != nil {
		return fmt.Errorf("read B1 validator key artifact: %w", err)
	}
	var key privval.FilePVKey
	if err := cmtjson.Unmarshal(raw, &key); err != nil {
		return fmt.Errorf("decode B1 validator key artifact: %w", err)
	}
	if hex.EncodeToString(key.Address) != config.ValidatorAddress {
		return errors.New("B1 validator key conflicts with this home's genesis identity binding")
	}
	return nil
}
