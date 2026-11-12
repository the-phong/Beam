package b0app

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"

	"example.com/pluto/internal/classical/genesis"
	"example.com/pluto/internal/classical/store"
)

const goldenGenesis = "504c55544f47322100000002000b706c75746f2d62302d763100000001b9631c0f8d9d2d9d56bd6151559f3339e14eb36a0000000000000064"

func fixture(t *testing.T) ([]byte, []byte) {
	t.Helper()
	binary, err := hex.DecodeString(goldenGenesis)
	if err != nil {
		t.Fatal(err)
	}
	transport, err := b0genesis.Encode(binary, "pluto-b0-v1")
	if err != nil {
		t.Fatal(err)
	}
	return binary, transport
}

func initialize(t *testing.T, path string) (*App, [32]byte) {
	t.Helper()
	binary, transport := fixture(t)
	store, err := b0store.Create(path, binary, "pluto-b0-v1")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	a, err := Open(path, "pluto-b0-v1", binary)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := a.InitChain(context.Background(), &abci.RequestInitChain{
		InitialHeight: 1, ChainId: "pluto-b0-v1", AppStateBytes: transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	var hash [32]byte
	copy(hash[:], resp.AppHash)
	return a, hash
}

func TestInitializeDurableIdempotentAndReproducible(t *testing.T) {
	path1 := filepath.Join(t.TempDir(), "b0")
	path2 := filepath.Join(t.TempDir(), "b0")
	a, hash := initialize(t, path1)
	b, secondHash := initialize(t, path2)
	defer b.Close()
	if hash != secondHash || hex.EncodeToString(hash[:]) != "e480c8b300c5ac1d662628f90334aae185ad38ed791c59ab14e3aa42abf083da" {
		t.Fatalf("cross-instance AppHash mismatch %x / %x", hash, secondHash)
	}
	state, exists, err := a.Committed()
	if err != nil || !exists || state.Height() != 0 || state.TotalSupply() != 100 {
		t.Fatalf("committed state: exists=%v err=%v", exists, err)
	}
	accounts := state.Accounts()
	if len(accounts) != 1 || accounts[0].Balance != 100 || accounts[0].Nonce != 0 {
		t.Fatalf("accounts: %+v", accounts)
	}
	binary, transport := fixture(t)
	retry, err := a.InitChain(context.Background(), &abci.RequestInitChain{InitialHeight: 1, ChainId: "pluto-b0-v1", AppStateBytes: transport})
	if err != nil || !bytes.Equal(retry.AppHash, hash[:]) {
		t.Fatalf("identical retry: %v", err)
	}
	conflicting := bytes.Clone(binary)
	conflicting[len(conflicting)-1]++
	conflictingTransport, err := b0genesis.Encode(conflicting, "pluto-b0-v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.InitChain(context.Background(), &abci.RequestInitChain{
		InitialHeight: 1, ChainId: "pluto-b0-v1", AppStateBytes: conflictingTransport,
	}); !errors.Is(err, ErrGenesisConflict) {
		t.Fatalf("committed genesis conflict: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if other, err := Open(path1, "pluto-b0-v1", conflicting); !errors.Is(err, ErrGenesisConflict) {
		if err == nil {
			_ = other.Close()
		}
		t.Fatalf("reopen with conflicting genesis: %v", err)
	}
	reopened, err := Open(path1, "pluto-b0-v1", binary)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	info, err := reopened.Info(context.Background(), &abci.RequestInfo{})
	if err != nil || info.LastBlockHeight != 0 || !bytes.Equal(info.LastBlockAppHash, hash[:]) {
		t.Fatalf("reopened Info: %+v err=%v", info, err)
	}
	retry, err = reopened.InitChain(context.Background(), &abci.RequestInitChain{InitialHeight: 1, ChainId: "pluto-b0-v1", AppStateBytes: transport})
	if err != nil || !bytes.Equal(retry.AppHash, hash[:]) {
		t.Fatalf("reopened identical retry: %v", err)
	}
}

func TestInitConflictAndInterruptedDirectory(t *testing.T) {
	binary, transport := fixture(t)
	path := filepath.Join(t.TempDir(), "b0")
	a, err := Open(path, "pluto-b0-v1", binary)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	changed := bytes.Clone(binary)
	changed[len(changed)-1]++
	changedTransport, err := b0genesis.Encode(changed, "pluto-b0-v1")
	if err != nil {
		t.Fatal(err)
	}
	for _, req := range []*abci.RequestInitChain{
		{InitialHeight: 2, ChainId: "pluto-b0-v1", AppStateBytes: transport},
		{InitialHeight: 1, ChainId: "wrong-chain", AppStateBytes: transport},
		{InitialHeight: 1, ChainId: "pluto-b0-v1", AppStateBytes: changedTransport},
		{InitialHeight: 1, ChainId: "pluto-b0-v1", AppStateBytes: binary},
	} {
		if _, err := a.InitChain(context.Background(), req); !errors.Is(err, ErrGenesisConflict) {
			t.Fatalf("conflicting InitChain error %v", err)
		}
	}
	if _, exists, err := a.Committed(); err != nil || exists {
		t.Fatalf("conflict changed state: exists=%v err=%v", exists, err)
	}
	if _, err := a.InitChain(context.Background(), &abci.RequestInitChain{
		InitialHeight: 1, ChainId: "pluto-b0-v1", AppStateBytes: transport,
	}); !errors.Is(err, b0store.ErrMissing) {
		t.Fatalf("missing H0 InitChain was not rejected: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("conflict created database: %v", err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path, "pluto-b0-v1", binary); err == nil {
		t.Fatal("interrupted initialization directory was silently reset")
	}
	if _, err := b0store.OpenExisting(path, "pluto-b0-v1"); err == nil {
		t.Fatal("incomplete B0 database was accepted")
	}
}

func TestMalformedTransactionDoesNotChangeCommittedState(t *testing.T) {
	a, hash := initialize(t, filepath.Join(t.TempDir(), "b0"))
	defer a.Close()
	resp, err := a.CheckTx(context.Background(), &abci.RequestCheckTx{Tx: []byte("arbitrary")})
	if err != nil || resp.Code != 1 || resp.Log != "malformed" || resp.Codespace != resultCodespace {
		t.Fatalf("malformed transaction result: %+v %v", resp, err)
	}
	info, err := a.Info(context.Background(), &abci.RequestInfo{})
	if err != nil || info.LastBlockHeight != 0 || !bytes.Equal(info.LastBlockAppHash, hash[:]) {
		t.Fatalf("transaction changed committed state: %+v %v", info, err)
	}
}

func TestFinalizeBlockHeightAndPendingOrder(t *testing.T) {
	a, genesisHash := initialize(t, filepath.Join(t.TempDir(), "b0"))
	defer a.Close()
	ctx := context.Background()
	for _, req := range []*abci.RequestFinalizeBlock{
		{Height: 0}, {Height: 2},
	} {
		if _, err := a.FinalizeBlock(ctx, req); err == nil {
			t.Fatalf("accepted invalid height or nonempty block: %+v", req)
		}
	}
	first, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first.AppHash, genesisHash[:]) {
		t.Fatalf("height-one pending hash: %+v %v", first, err)
	}
	if _, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1}); !errors.Is(err, b0store.ErrPending) {
		t.Fatalf("duplicate pending finalization: %v", err)
	}
	info, err := a.Info(ctx, &abci.RequestInfo{})
	if err != nil || info.LastBlockHeight != 0 || !bytes.Equal(info.LastBlockAppHash, genesisHash[:]) {
		t.Fatalf("pending state became committed before Commit: %+v %v", info, err)
	}
	if _, err := a.Commit(ctx, &abci.RequestCommit{}); err != nil {
		t.Fatal(err)
	}
	info, err = a.Info(ctx, &abci.RequestInfo{})
	if err != nil || info.LastBlockHeight != 1 || !bytes.Equal(info.LastBlockAppHash, first.AppHash) {
		t.Fatalf("committed height-one state: %+v %v", info, err)
	}
	for _, height := range []int64{0, 1, 3} {
		if _, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: height}); err == nil {
			t.Fatalf("accepted stale or skipped height %d", height)
		}
	}
	if _, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 2}); err != nil {
		t.Fatalf("next height after Commit: %v", err)
	}
}

func TestConcurrentDuplicateFinalizeBlock(t *testing.T) {
	a, genesisHash := initialize(t, filepath.Join(t.TempDir(), "b0"))
	defer a.Close()
	const callers = 8
	start := make(chan struct{})
	results := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := a.FinalizeBlock(context.Background(), &abci.RequestFinalizeBlock{Height: 1})
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	success, pending := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, b0store.ErrPending):
			pending++
		default:
			t.Fatalf("unexpected callback result: %v", err)
		}
	}
	if success != 1 || pending != callers-1 {
		t.Fatalf("non-atomic duplicate handling: success=%d pending=%d", success, pending)
	}
	info, err := a.Info(context.Background(), &abci.RequestInfo{})
	if err != nil || info.LastBlockHeight != 0 || !bytes.Equal(info.LastBlockAppHash, genesisHash[:]) {
		t.Fatalf("concurrent finalization exposed pending state: %+v %v", info, err)
	}
}

func TestFinalizeCommitCallbackInterleaving(t *testing.T) {
	a, _ := initialize(t, filepath.Join(t.TempDir(), "b0"))
	defer a.Close()
	type result struct {
		method string
		err    error
	}
	ready := make(chan struct{}, 2)
	results := make(chan result, 2)
	a.mu.Lock() // Queue both callbacks before allowing either write lock.
	go func() {
		ready <- struct{}{}
		_, err := a.FinalizeBlock(context.Background(), &abci.RequestFinalizeBlock{Height: 1})
		results <- result{method: "FinalizeBlock", err: err}
	}()
	go func() {
		ready <- struct{}{}
		_, err := a.Commit(context.Background(), &abci.RequestCommit{})
		results <- result{method: "Commit", err: err}
	}()
	<-ready
	<-ready
	a.mu.Unlock()
	first, second := <-results, <-results
	var finalizeErr, commitErr error
	for _, got := range []result{first, second} {
		if got.method == "FinalizeBlock" {
			finalizeErr = got.err
		} else {
			commitErr = got.err
		}
	}
	if finalizeErr != nil {
		t.Fatalf("FinalizeBlock failed in callback ordering: %v", finalizeErr)
	}
	state, exists, err := a.Committed()
	if err != nil || !exists {
		t.Fatalf("committed read: exists=%v err=%v", exists, err)
	}
	_, hasPending, err := a.store.Pending()
	if err != nil {
		t.Fatal(err)
	}
	if commitErr == nil {
		if state.Height() != 1 || hasPending {
			t.Fatalf("successful Commit did not promote exactly one block: height=%d pending=%v", state.Height(), hasPending)
		}
	} else if errors.Is(commitErr, b0store.ErrNoPending) {
		if state.Height() != 0 || !hasPending {
			t.Fatalf("early Commit changed state: height=%d pending=%v", state.Height(), hasPending)
		}
	} else {
		t.Fatalf("unexpected Commit error: %v", commitErr)
	}
}

func TestEmptyBlockCheckpointRejectsDifferentGenesisAfterHeightZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b0")
	a, _ := initialize(t, path)
	ctx := context.Background()
	if _, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Commit(ctx, &abci.RequestCommit{}); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	binary, _ := fixture(t)
	changed := bytes.Clone(binary)
	changed[len(changed)-1]++ // Same chain ID; different allocation.
	if other, err := Open(path, "pluto-b0-v1", changed); !errors.Is(err, ErrGenesisConflict) {
		if err == nil {
			_ = other.Close()
		}
		t.Fatalf("alternate genesis accepted at height one: %v", err)
	}
	reopened, err := Open(path, "pluto-b0-v1", binary)
	if err != nil {
		t.Fatalf("original genesis rejected at height one: %v", err)
	}
	defer reopened.Close()
}

func TestGenesisProcessKilledAfterSyncBeforeResponse(t *testing.T) {
	if os.Getenv("PLUTO_B0_GENESIS_KILL_CHILD") == "1" {
		path := os.Getenv("PLUTO_B0_GENESIS_KILL_DIR")
		marker := os.Getenv("PLUTO_B0_GENESIS_KILL_MARKER")
		a, _ := initialize(t, path) // Explicit D-024 Create synced H0 before validation-only InitChain.
		if err := os.WriteFile(marker, []byte("genesis sync completed"), 0600); err != nil {
			t.Fatal(err)
		}
		_ = a // Keep the database open; the parent sends SIGKILL.
		for {
			time.Sleep(time.Hour)
		}
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "b0")
	marker := filepath.Join(dir, "after-sync.marker")
	cmd := exec.Command(os.Args[0], "-test.run=^TestGenesisProcessKilledAfterSyncBeforeResponse$")
	cmd.Env = append(os.Environ(), "PLUTO_B0_GENESIS_KILL_CHILD=1",
		"PLUTO_B0_GENESIS_KILL_DIR="+path, "PLUTO_B0_GENESIS_KILL_MARKER="+marker)
	var childOutput bytes.Buffer
	cmd.Stdout, cmd.Stderr = &childOutput, &childOutput
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatalf("child did not complete synced genesis write: %s", childOutput.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("child was not terminated by SIGKILL")
	}
	binary, transport := fixture(t)
	a, err := Open(path, "pluto-b0-v1", binary)
	if err != nil {
		t.Fatalf("reopen synced height-zero snapshot after SIGKILL: %v", err)
	}
	defer a.Close()
	info, err := a.Info(context.Background(), &abci.RequestInfo{})
	if err != nil || info.LastBlockHeight != 0 || hex.EncodeToString(info.LastBlockAppHash) != "e480c8b300c5ac1d662628f90334aae185ad38ed791c59ab14e3aa42abf083da" {
		t.Fatalf("recovered genesis Info: %+v %v", info, err)
	}
	response, err := a.InitChain(context.Background(), &abci.RequestInitChain{
		InitialHeight: 1, ChainId: "pluto-b0-v1", AppStateBytes: transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(response.AppHash, info.LastBlockAppHash) {
		t.Fatalf("identical InitChain retry after SIGKILL: %+v %v", response, err)
	}
	t.Log("SIGKILL after explicit synced H0 creation in standalone subprocess: complete height-zero state recovered; identical InitChain retry succeeded; no CometBFT call was involved")
}

func TestABCIPathsFailClosedAfterUncertainCommit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b0")
	a, _ := initialize(t, path)
	if err := a.store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := b0store.OpenExistingWithOptions(path, "pluto-b0-v1", b0store.Options{
		AfterSync: func() error { return errors.New("injected response loss after sync") },
	})
	if err != nil {
		t.Fatal(err)
	}
	a.store = store // isolated test-only replacement before any concurrent call
	ctx := context.Background()
	if _, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Commit(ctx, &abci.RequestCommit{}); err == nil {
		t.Fatal("ambiguous Commit succeeded")
	}
	for name, call := range map[string]func() error{
		"Info":            func() error { _, err := a.Info(ctx, &abci.RequestInfo{}); return err },
		"Query":           func() error { _, err := a.Query(ctx, &abci.RequestQuery{}); return err },
		"CheckTx":         func() error { _, err := a.CheckTx(ctx, &abci.RequestCheckTx{Tx: []byte{1}}); return err },
		"PrepareProposal": func() error { _, err := a.PrepareProposal(ctx, &abci.RequestPrepareProposal{Height: 1}); return err },
		"ProcessProposal": func() error { _, err := a.ProcessProposal(ctx, &abci.RequestProcessProposal{Height: 1}); return err },
		"FinalizeBlock":   func() error { _, err := a.FinalizeBlock(ctx, &abci.RequestFinalizeBlock{Height: 1}); return err },
		"Commit":          func() error { _, err := a.Commit(ctx, &abci.RequestCommit{}); return err },
	} {
		if err := call(); !errors.Is(err, b0store.ErrUncertain) {
			t.Fatalf("%s exposed uncertain state: %v", name, err)
		}
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	genesis, _ := fixture(t)
	reopened, err := Open(path, "pluto-b0-v1", genesis)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	info, err := reopened.Info(ctx, &abci.RequestInfo{})
	if err != nil || info.LastBlockHeight != 1 {
		t.Fatalf("validated reopen: %+v %v", info, err)
	}
}
