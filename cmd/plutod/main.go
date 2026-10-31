package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"example.com/pluto/internal/b0genesis"
	"example.com/pluto/internal/b1genesis"
	"example.com/pluto/internal/node"
	"example.com/pluto/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "Pluto: use init for the historical Phase 02 profile or init-b0 to create the B0 Genesis H0 snapshot.")
		usage(stdout)
		return 0
	}
	switch args[0] {
	case "help", "-h", "--help":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "help takes no arguments")
			return 2
		}
		usage(stdout)
		return 0
	case "version", "-v", "--version":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "version takes no arguments")
			return 2
		}
		fmt.Fprintf(stdout, "plutod %s\nGo: %s\nPlatform: %s/%s\nRevision: %s\n", version.Number, runtime.Version(), runtime.GOOS, runtime.GOARCH, version.Revision)
		return 0
	case "init":
		flags := flag.NewFlagSet("init", flag.ContinueOnError)
		flags.SetOutput(stderr)
		home := flags.String("home", ".pluto", "node data directory")
		chainID := flags.String("chain-id", "pluto-dev-1", "chain identity")
		rpcPort := flags.Int("rpc-port", 26657, "localhost RPC port")
		p2pPort := flags.Int("p2p-port", 26656, "localhost P2P port")
		abciPort := flags.Int("abci-port", 26658, "localhost ABCI port")
		if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
			fmt.Fprintln(stderr, "invalid init arguments")
			return 2
		}
		if err := node.Init(*home, cometBinary(), *chainID, *rpcPort, *p2pPort, *abciPort, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Initialized Pluto node at %s (chain %s)\n", *home, *chainID)
		return 0
	case "init-b0":
		flags := flag.NewFlagSet("init-b0", flag.ContinueOnError)
		flags.SetOutput(stderr)
		home := flags.String("home", ".pluto-b0", "fresh B0 node data directory")
		chainID := flags.String("chain-id", "pluto-b0-v1", "B0 chain identity")
		genesisFile := flags.String("genesis-file", "", "path to canonical binary Genesis V2")
		rpcPort := flags.Int("rpc-port", 26657, "localhost RPC port")
		p2pPort := flags.Int("p2p-port", 26656, "localhost P2P port")
		abciPort := flags.Int("abci-port", 26658, "localhost ABCI port")
		if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || *genesisFile == "" {
			fmt.Fprintln(stderr, "invalid init-b0 arguments")
			return 2
		}
		info, err := os.Stat(*genesisFile)
		if err != nil || !info.Mode().IsRegular() || info.Size() > b0genesis.MaxBinarySize {
			fmt.Fprintln(stderr, "invalid or oversized binary Genesis V2 file")
			return 1
		}
		binary, err := os.ReadFile(*genesisFile)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := node.InitB0(*home, cometBinary(), *chainID, binary, *rpcPort, *p2pPort, *abciPort, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Initialized Pluto B0 Genesis H0 snapshot at %s (chain %s)\n", *home, *chainID)
		return 0
	case "init-b0-4v":
		flags := flag.NewFlagSet("init-b0-4v", flag.ContinueOnError)
		flags.SetOutput(stderr)
		root := flags.String("root", "", "fresh root for four independent validator homes")
		chainID := flags.String("chain-id", "pluto-b0-v1", "shared B0 chain identity")
		genesisFile := flags.String("genesis-file", "", "path to canonical binary Genesis V2")
		rpcBase := flags.Int("rpc-base-port", 26657, "first localhost RPC port; subsequent validators use increments of 10")
		p2pBase := flags.Int("p2p-base-port", 26656, "first localhost P2P port; subsequent validators use increments of 10")
		abciBase := flags.Int("abci-base-port", 26658, "first localhost ABCI port; subsequent validators use increments of 10")
		if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || *root == "" || *genesisFile == "" {
			fmt.Fprintln(stderr, "invalid init-b0-4v arguments")
			return 2
		}
		info, err := os.Stat(*genesisFile)
		if err != nil || !info.Mode().IsRegular() || info.Size() > b0genesis.MaxBinarySize {
			fmt.Fprintln(stderr, "invalid or oversized binary Genesis V2 file")
			return 1
		}
		binary, err := os.ReadFile(*genesisFile)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		var ports [4]node.B0Ports
		for i := range ports {
			ports[i] = node.B0Ports{RPC: *rpcBase + 10*i, P2P: *p2pBase + 10*i, ABCI: *abciBase + 10*i}
		}
		if err := node.InitB0FourValidators(*root, cometBinary(), *chainID, binary, ports, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Initialized four independent Pluto B0 validators under %s (chain %s)\n", *root, *chainID)
		return 0
	case "init-b1-4v":
		flags := flag.NewFlagSet("init-b1-4v", flag.ContinueOnError)
		flags.SetOutput(stderr)
		root := flags.String("root", "", "fresh root for four independent B1 validator homes")
		chainID := flags.String("chain-id", "pluto-b1-v1", "shared B1 chain identity")
		genesisFile := flags.String("genesis-file", "", "path to canonical binary B1 Genesis V3")
		rpcBase := flags.Int("rpc-base-port", 26657, "first localhost RPC port; subsequent validators use increments of 10")
		p2pBase := flags.Int("p2p-base-port", 26656, "first localhost P2P port; subsequent validators use increments of 10")
		abciBase := flags.Int("abci-base-port", 26658, "first localhost ABCI port; subsequent validators use increments of 10")
		if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || *root == "" || *genesisFile == "" {
			fmt.Fprintln(stderr, "invalid init-b1-4v arguments")
			return 2
		}
		info, err := os.Stat(*genesisFile)
		if err != nil || !info.Mode().IsRegular() || info.Size() > b1genesis.MaxBinarySize {
			fmt.Fprintln(stderr, "invalid or oversized binary B1 Genesis V3 file")
			return 1
		}
		binary, err := os.ReadFile(*genesisFile)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		var ports [4]node.B0Ports
		for i := range ports {
			ports[i] = node.B0Ports{RPC: *rpcBase + 10*i, P2P: *p2pBase + 10*i, ABCI: *abciBase + 10*i}
		}
		if err := node.InitB1FourValidators(*root, cometBinary(), *chainID, binary, ports, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Initialized four independent Pluto B1 validators under %s (chain %s)\n", *root, *chainID)
		return 0
	case "start":
		flags := flag.NewFlagSet("start", flag.ContinueOnError)
		flags.SetOutput(stderr)
		home := flags.String("home", ".pluto", "node data directory")
		if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
			fmt.Fprintln(stderr, "invalid start arguments")
			return 2
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := node.Run(ctx, *home, cometBinary(), stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case "state":
		flags := flag.NewFlagSet("state", flag.ContinueOnError)
		flags.SetOutput(stderr)
		home := flags.String("home", ".pluto", "node data directory")
		if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
			fmt.Fprintln(stderr, "invalid state arguments")
			return 2
		}
		state, err := node.State(*home)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		result := struct {
			ChainID string `json:"chain_id"`
			Version uint32 `json:"state_version"`
			Height  int64  `json:"height"`
			AppHash string `json:"app_hash"`
		}{state.ChainID, state.Version, state.Height, hex.EncodeToString(state.AppHash[:])}
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: plutod [init|init-b0|init-b0-4v|init-b1-4v|start|state|help|version] [flags]")
	fmt.Fprintln(w, "init selects Phase 02; init-b0 initializes one B0 validator; init-b0-4v and init-b1-4v initialize independent four-validator networks.")
}

func cometBinary() string {
	self, err := os.Executable()
	if err != nil {
		return "cometbft-unavailable"
	}
	return filepath.Join(filepath.Dir(self), "cometbft")
}
