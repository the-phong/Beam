package node

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	"github.com/cometbft/cometbft/abci/server"
	"github.com/cometbft/cometbft/types"

	"example.com/pluto/internal/classical/app"
	"example.com/pluto/internal/classical/genesis"
	"example.com/pluto/internal/classical/state"
	"example.com/pluto/internal/classical/store"
)

// runB0 is an explicit genesis-only profile. It keeps the Phase 02 runtime
// and database path untouched while retaining the same child lifecycle.
func runB0(ctx context.Context, home, cometBinary string, config Config, output io.Writer) error {
	genesis, err := types.GenesisDocFromFile(filepath.Join(home, "config", "genesis.json"))
	if err != nil {
		return fmt.Errorf("read CometBFT genesis: %w", err)
	}
	if genesis.ChainID != config.ChainID || genesis.InitialHeight != 1 {
		return errors.New("B0 genesis conflicts with Pluto node metadata")
	}
	binary, _, err := b0genesis.Decode(genesis.AppState, config.ChainID)
	if err != nil {
		return fmt.Errorf("decode B0 genesis transport: %w", err)
	}
	digest := sha256.Sum256(binary)
	if hex.EncodeToString(digest[:]) != config.GenesisSHA256 {
		return errors.New("B0 genesis digest conflicts with Pluto node metadata")
	}
	if _, err := os.Stat(filepath.Join(home, "data", "pluto")); err == nil {
		return errors.New("Phase 02 application database is present; refusing B0 profile")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect Phase 02 database: %w", err)
	}
	dbDir := filepath.Join(home, "data", "pluto-b0-v2")
	if _, err := os.Stat(dbDir); errors.Is(err, os.ErrNotExist) {
		return errors.New("B0 snapshot missing; start requires an existing initialized B0 store")
	} else if err != nil {
		return fmt.Errorf("inspect B0 database: %w", err)
	}
	state, err := b0store.ProbeExisting(dbDir, config.ChainID)
	if err != nil {
		return fmt.Errorf("validate existing B0 snapshot read-only: %w", err)
	}
	expected, err := b0state.ParseGenesis(binary, config.ChainID)
	if err != nil {
		return fmt.Errorf("validate configured B0 genesis: %w", err)
	}
	if state.TotalSupply() != expected.TotalSupply() ||
		(state.Height() == 0 && (state.AppHash() != expected.AppHash() || !slices.Equal(state.Accounts(), expected.Accounts()))) {
		return b0app.ErrGenesisConflict
	}
	if err := verifyCometRunArtifacts(home, config, genesis); err != nil {
		return fmt.Errorf("validate CometBFT identity and configuration: %w", err)
	}
	a, err := b0app.Open(dbDir, config.ChainID, binary)
	if err != nil {
		return fmt.Errorf("open existing B0 snapshot: %w", err)
	}
	defer a.Close()
	if _, exists, err := a.Committed(); err != nil {
		return fmt.Errorf("validate B0 snapshot: %w", err)
	} else if !exists {
		return fmt.Errorf("B0 snapshot missing after open: %w", b0store.ErrMissing)
	}
	srv, err := server.NewServer(config.ABCIAddress, "socket", a)
	if err != nil {
		return err
	}
	if err := srv.Start(); err != nil {
		return fmt.Errorf("start B0 ABCI server: %w", err)
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
