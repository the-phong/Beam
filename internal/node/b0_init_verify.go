package node

import (
	"bytes"
	stded25519 "crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	cfg "github.com/cometbft/cometbft/config"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/types"
	"github.com/spf13/viper"
)

// verifyCometInitArtifacts validates the generated configuration and identity
// files without calling Comet's LoadFilePV, which exits the process on a bad
// file. It never regenerates or repairs an artifact.
func verifyCometInitArtifacts(home string, expected Config, genesis *types.GenesisDoc) error {
	return verifyCometArtifacts(home, expected, genesis, true)
}

// verifyCometRunArtifacts checks the same identity binding on restart while
// allowing CometBFT's last-sign state to have advanced with committed blocks.
func verifyCometRunArtifacts(home string, expected Config, genesis *types.GenesisDoc) error {
	return verifyCometArtifacts(home, expected, genesis, false)
}

func verifyCometArtifacts(home string, expected Config, genesis *types.GenesisDoc, fresh bool) error {
	configFile := filepath.Join(home, "config", "config.toml")
	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read CometBFT configuration: %w", err)
	}
	for _, key := range []string{"proxy_app", "abci", "genesis_file", "priv_validator_key_file", "priv_validator_state_file", "node_key_file", "rpc.laddr", "p2p.laddr"} {
		if !v.IsSet(key) {
			return fmt.Errorf("CometBFT configuration missing %s", key)
		}
	}
	cometConfig := cfg.DefaultConfig()
	if err := v.Unmarshal(cometConfig); err != nil {
		return fmt.Errorf("decode CometBFT configuration: %w", err)
	}
	cometConfig.SetRoot(home)
	if err := cometConfig.ValidateBasic(); err != nil {
		return fmt.Errorf("invalid CometBFT configuration: %w", err)
	}
	if cometConfig.ProxyApp != expected.ABCIAddress || cometConfig.RPC.ListenAddress != expected.RPCAddress ||
		cometConfig.P2P.ListenAddress != expected.P2PAddress || cometConfig.ABCI != "socket" ||
		!cometConfig.Consensus.CreateEmptyBlocks {
		return errors.New("CometBFT configuration conflicts with initialized B0 addresses or runtime")
	}
	paths := map[string]string{
		"genesis":              cometConfig.GenesisFile(),
		"priv_validator_key":   cometConfig.PrivValidatorKeyFile(),
		"priv_validator_state": cometConfig.PrivValidatorStateFile(),
		"node_key":             cometConfig.NodeKeyFile(),
	}
	for name, actual := range paths {
		wantDir, wantFile := "config", name+".json"
		if name == "genesis" {
			wantFile = "genesis.json"
		} else if name == "priv_validator_state" {
			wantDir = "data"
		}
		if filepath.Clean(actual) != filepath.Join(home, wantDir, wantFile) {
			return fmt.Errorf("CometBFT %s path conflicts with initialized B0 home", name)
		}
	}
	if err := verifyCometValidator(paths["priv_validator_key"], paths["priv_validator_state"], genesis, fresh); err != nil {
		return err
	}
	if err := verifyCometNodeKey(paths["node_key"]); err != nil {
		return err
	}
	return nil
}

func validCometEd25519Key(key interface{ Bytes() []byte }) bool {
	priv, ok := key.(cmted25519.PrivKey)
	if !ok || len(priv) != cmted25519.PrivateKeySize {
		return false
	}
	return bytes.Equal(stded25519.NewKeyFromSeed(priv[:stded25519.SeedSize]), priv)
}

func verifyCometValidator(keyPath, statePath string, genesis *types.GenesisDoc, fresh bool) error {
	rawKey, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("read CometBFT validator key: %w", err)
	}
	var key privval.FilePVKey
	if err := cmtjson.Unmarshal(rawKey, &key); err != nil {
		return fmt.Errorf("decode CometBFT validator key: %w", err)
	}
	if key.PrivKey == nil || !validCometEd25519Key(key.PrivKey) || key.PubKey == nil {
		return errors.New("CometBFT validator key is incomplete or invalid")
	}
	pub := key.PrivKey.PubKey()
	if !bytes.Equal(key.PubKey.Bytes(), pub.Bytes()) || !bytes.Equal(key.Address, pub.Address()) {
		return errors.New("CometBFT validator key conflicts with genesis validator")
	}
	if len(genesis.Validators) != 1 && len(genesis.Validators) != 4 {
		return errors.New("CometBFT genesis must contain one or four validators")
	}
	matched := 0
	seen := make(map[string]bool, len(genesis.Validators))
	for _, validator := range genesis.Validators {
		if validator.PubKey == nil || validator.Power <= 0 || !bytes.Equal(validator.Address, validator.PubKey.Address()) || seen[string(validator.Address)] {
			return errors.New("CometBFT genesis validator set is incomplete or duplicated")
		}
		seen[string(validator.Address)] = true
		if len(genesis.Validators) == 4 && validator.Power != genesis.Validators[0].Power {
			return errors.New("CometBFT four-validator voting power must be equal")
		}
		if bytes.Equal(validator.PubKey.Bytes(), pub.Bytes()) && bytes.Equal(validator.Address, pub.Address()) {
			matched++
		}
	}
	if matched != 1 {
		return errors.New("CometBFT validator key conflicts with genesis validator")
	}
	rawState, err := os.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("read CometBFT validator initialization state: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawState, &fields); err != nil {
		return fmt.Errorf("decode CometBFT validator initialization state: %w", err)
	}
	for _, name := range []string{"height", "round", "step"} {
		value, exists := fields[name]
		if !exists || bytes.Equal(value, []byte("null")) {
			return fmt.Errorf("CometBFT validator initialization state missing %s", name)
		}
	}
	var state privval.FilePVLastSignState
	if err := cmtjson.Unmarshal(rawState, &state); err != nil {
		return fmt.Errorf("decode CometBFT validator initialization state: %w", err)
	}
	if state.Height < 0 || (fresh && (state.Height != 0 || state.Round != 0 || state.Step != 0 || len(state.Signature) != 0 || len(state.SignBytes) != 0)) {
		return errors.New("CometBFT validator initialization state is not fresh")
	}
	return nil
}

func verifyCometNodeKey(path string) error {
	key, err := p2p.LoadNodeKey(path)
	if err != nil {
		return fmt.Errorf("load CometBFT node key: %w", err)
	}
	if key == nil || key.PrivKey == nil || !validCometEd25519Key(key.PrivKey) || key.ID() == "" {
		return errors.New("CometBFT node key is incomplete or invalid")
	}
	return nil
}
