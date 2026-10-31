package b0store

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/pluto/internal/b0state"
	"example.com/pluto/internal/tx"
	"github.com/cockroachdb/pebble"
)

const chain = "pluto-b0-v1"

type vector struct {
	Wire     string `json:"wire_hex"`
	Sender   string `json:"sender_address_hex"`
	Receiver string `json:"receiver_hex"`
}

func fixture(t *testing.T) ([]byte, []byte, b0state.Address, b0state.Address) {
	t.Helper()
	data, err := osReadFile("../../research/evidence/phase_03b/golden_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v []vector
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	var a, b b0state.Address
	aBytes, _ := hex.DecodeString(v[0].Sender)
	bBytes, _ := hex.DecodeString(v[0].Receiver)
	copy(a[:], aBytes)
	copy(b[:], bBytes)
	wire, _ := hex.DecodeString(v[0].Wire)
	genesis, err := b0state.EncodeGenesis(chain, []b0state.Allocation{{Address: a, Balance: 100}})
	if err != nil {
		t.Fatal(err)
	}
	return genesis, wire, a, b
}

// Keep file reads behind one helper so fixtures are never generated from the
// implementation under test in the persistence tests.
func osReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

func TestCreateCommitRestartAndIsolation(t *testing.T) {
	genesis, wire, a, b := fixture(t)
	dir := filepath.Join(t.TempDir(), "b0")
	s, err := Create(dir, genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Create(dir, genesis, chain); err != ErrExisting {
		t.Fatal("repeat init", err)
	}
	if _, err := OpenExisting(dir, chain); err == nil {
		t.Fatal("Pebble lock not enforced")
	}
	initial, err := s.Committed()
	if err != nil || initial.Height() != 0 {
		t.Fatal(err)
	}
	codes, pendingHash, err := s.Prepare([][]byte{wire})
	if err != nil || len(codes) != 1 || codes[0] != tx.CodeOK {
		t.Fatal(codes, err)
	}
	if _, _, err := s.Prepare(nil); err != ErrPending {
		t.Fatal("duplicate pending", err)
	}
	committed, _ := s.Committed()
	if committed.Height() != 0 || committed.AppHash() != initial.AppHash() || len(committed.Accounts()) != 1 {
		t.Fatal("pending leaked")
	}
	pending, yes, err := s.Pending()
	if err != nil || !yes || pending.Height() != 1 || pending.AppHash() != pendingHash {
		t.Fatal("pending state")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenExisting(dir, chain)
	if err != nil {
		t.Fatal(err)
	}
	committed, _ = s.Committed()
	if committed.Height() != 0 || committed.AppHash() != initial.AppHash() {
		t.Fatal("uncommitted state survived restart")
	}
	if _, yes, _ := s.Pending(); yes {
		t.Fatal("pending survived restart")
	}
	if _, _, err := s.Prepare([][]byte{wire}); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit(); err != ErrNoPending {
		t.Fatal("duplicate commit", err)
	}
	committed, _ = s.Committed()
	aa, _ := committed.Account(a)
	bb, _ := committed.Account(b)
	if committed.Height() != 1 || aa.Balance != 75 || aa.Nonce != 1 || bb.Balance != 25 {
		t.Fatal("committed transfer")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenExisting(dir, chain)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, _ := s.Committed()
	if reloaded.AppHash() != committed.AppHash() || reloaded.Height() != 1 {
		t.Fatal("restart mismatch")
	}
	codes, _, err = s.Prepare(nil)
	if err != nil || len(codes) != 0 {
		t.Fatal(err)
	}
	if err := s.Commit(); err != nil {
		t.Fatal(err)
	}
	empty, _ := s.Committed()
	if empty.Height() != 2 || empty.AppHash() == reloaded.AppHash() || empty.TotalSupply() != 100 {
		t.Fatal("empty block")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenExisting(dir, chain)
	if err != nil {
		t.Fatal(err)
	}
	latest, _ := s.Committed()
	if latest.Height() != 2 || latest.AppHash() != empty.AppHash() {
		t.Fatal("empty restart")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Committed(); err != ErrClosed {
		t.Fatal("closed read", err)
	}
}

func TestFaultBoundaries(t *testing.T) {
	genesis, wire, _, _ := fixture(t)
	dir := filepath.Join(t.TempDir(), "b0")
	s, err := Create(dir, genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	calls := 0
	s, err = OpenExistingWithOptions(dir, chain, Options{BeforeSync: func() error {
		calls++
		if calls == 1 {
			return errors.New("injected before write")
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Prepare([][]byte{wire}); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit(); err == nil {
		t.Fatal("expected prewrite error")
	}
	old, err := s.Committed()
	if err != nil || old.Height() != 0 {
		t.Fatal("promoted before write")
	}
	prewritePending, yes, err := s.Pending()
	if err != nil || !yes || prewritePending.Height() != 1 {
		t.Fatal("lost pending after prewrite error")
	}
	if err := s.Commit(); err != nil {
		t.Fatal("retry before-write failure", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenExisting(dir, chain)
	if err != nil {
		t.Fatal(err)
	}
	current, _ := s.Committed()
	if current.Height() != 1 {
		t.Fatal("successful retry not durable")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenExistingWithOptions(dir, chain, Options{AfterSync: func() error { return errors.New("injected after sync") }})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Prepare(nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit(); err == nil {
		t.Fatal("expected postwrite error")
	}
	if snapshot, err := s.Committed(); err != ErrUncertain || snapshot.ChainID() != "" || snapshot.AppHash() != ([32]byte{}) {
		t.Fatal("committed read did not fail closed", err)
	}
	if snapshot, yes, err := s.Pending(); err != ErrUncertain || yes || snapshot.ChainID() != "" || snapshot.AppHash() != ([32]byte{}) {
		t.Fatal("pending read did not fail closed", err)
	}
	if old.Height() != 0 || current.Height() != 1 || s.committed.Height() != 1 || s.pending == nil || s.pending.Height() != 2 {
		t.Fatal("previous snapshots or internal pending/committed state changed unexpectedly")
	}
	if err := s.Commit(); err != ErrUncertain {
		t.Fatal("uncertain retry", err)
	}
	if _, _, err := s.Prepare(nil); err != ErrUncertain {
		t.Fatal("uncertain prepare", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenExisting(dir, chain)
	if err != nil {
		t.Fatal(err)
	}
	recovered, _ := s.Committed()
	if recovered.Height() != 2 {
		t.Fatal("post-sync recovery must accept complete new record")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentCommittedReadsDuringCandidateLifecycle(t *testing.T) {
	genesis, wire, _, _ := fixture(t)
	s, err := Create(filepath.Join(t.TempDir(), "b0"), genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	initial, _ := s.Committed()
	var group sync.WaitGroup
	for i := 0; i < 12; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for j := 0; j < 20; j++ {
				state, readErr := s.Committed()
				if readErr != nil || (state.Height() != 0 && state.Height() != 1) || state.Validate() != nil {
					t.Errorf("invalid concurrent committed snapshot: %v", readErr)
					return
				}
			}
		}()
	}
	if _, _, err := s.Prepare([][]byte{wire}); err != nil {
		t.Fatal(err)
	}
	before, _ := s.Committed()
	if before.AppHash() != initial.AppHash() {
		t.Fatal("pending visible")
	}
	if err := s.Commit(); err != nil {
		t.Fatal(err)
	}
	group.Wait()
	committed, _ := s.Committed()
	if committed.Height() != 1 {
		t.Fatal("commit missing")
	}
}

func TestConcurrentReadersFailClosedOnUncertainCommit(t *testing.T) {
	genesis, wire, _, _ := fixture(t)
	dir := filepath.Join(t.TempDir(), "b0")
	s, err := Create(dir, genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	s, err = OpenExistingWithOptions(dir, chain, Options{AfterSync: func() error {
		close(entered)
		<-release
		return errors.New("injected post-sync response error")
	}})
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.Committed()
	if err != nil || before.Height() != 0 {
		t.Fatal("normal committed read", err)
	}
	if _, _, err := s.Prepare([][]byte{wire}); err != nil {
		t.Fatal(err)
	}
	prepared, yes, err := s.Pending()
	if err != nil || !yes || prepared.Height() != 1 {
		t.Fatal("normal pending read", err)
	}
	commitResult := make(chan error, 1)
	go func() { commitResult <- s.Commit() }()
	select {
	case <-entered:
	case <-time.After(20 * time.Second):
		close(release)
		t.Fatal("post-sync hook was not reached")
	}
	const readers = 24
	results := make(chan error, readers*2)
	var group sync.WaitGroup
	for i := 0; i < readers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			committed, readErr := s.Committed()
			if readErr != ErrUncertain || committed.ChainID() != "" {
				results <- fmt.Errorf("committed read: %v", readErr)
			}
			pending, available, readErr := s.Pending()
			if readErr != ErrUncertain || available || pending.ChainID() != "" {
				results <- fmt.Errorf("pending read: %v", readErr)
			}
		}()
	}
	close(release)
	if err := <-commitResult; err == nil {
		t.Fatal("expected post-sync error")
	}
	group.Wait()
	close(results)
	for err := range results {
		t.Error(err)
	}
	if before.Validate() != nil || prepared.Validate() != nil || before.Height() != 0 || prepared.Height() != 1 ||
		s.committed.Height() != 0 || s.pending == nil || s.pending.Height() != 1 ||
		s.committed.Validate() != nil || s.pending.Validate() != nil {
		t.Fatal("snapshots mutated or pending candidate lost")
	}
	if _, _, err := s.Prepare(nil); err != ErrUncertain {
		t.Fatal("uncertain prepare", err)
	}
	if err := s.Commit(); err != ErrUncertain {
		t.Fatal("uncertain commit", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenExisting(dir, chain)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	recovered, err := reopened.Committed()
	if err != nil || recovered.Height() != 1 || recovered.AppHash() != prepared.AppHash() {
		t.Fatal("reopen did not validate complete new snapshot", err)
	}
}

func TestProcessKillRecovery(t *testing.T) {
	if stage := os.Getenv("PLUTO_B0_KILL_CHILD"); stage != "" {
		genesis, wire, _, _ := fixture(t)
		s, err := Create(os.Getenv("PLUTO_B0_KILL_DIR"), genesis, chain)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := s.Prepare([][]byte{wire}); err != nil {
			t.Fatal(err)
		}
		if stage == "committed" {
			if err := s.Commit(); err != nil {
				t.Fatal(err)
			}
		}
		fmt.Println("B0_READY")
		time.Sleep(time.Hour) // Parent kills this disposable child process.
		return
	}
	for _, stage := range []string{"pending", "committed"} {
		t.Run(stage, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "b0")
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessKillRecovery$")
			cmd.Env = append(os.Environ(), "PLUTO_B0_KILL_CHILD="+stage, "PLUTO_B0_KILL_DIR="+dir)
			pipe, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			ready := make(chan string, 1)
			go func() { line, _ := bufio.NewReader(pipe).ReadString('\n'); ready <- strings.TrimSpace(line) }()
			select {
			case line := <-ready:
				if line != "B0_READY" {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
					t.Fatalf("child not ready: %q %s", line, stderr.String())
				}
			case <-ctx.Done():
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				t.Fatal("child readiness timed out")
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait() // Signal termination is expected.
			s, err := OpenExisting(dir, chain)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			state, err := s.Committed()
			if err != nil || state.Validate() != nil {
				t.Fatal("invalid recovered state", err)
			}
			want := int64(0)
			if stage == "committed" {
				want = 1
			}
			if state.Height() != want {
				t.Fatalf("recovered height %d, want %d", state.Height(), want)
			}
		})
	}
}

func TestCorruptPhysicalRecordsFailClosed(t *testing.T) {
	genesis, _, a, b := fixture(t)
	cases := map[string]func([]byte) []byte{
		"magic":   func(r []byte) []byte { r[0] = 0; return r },
		"version": func(r []byte) []byte { r[11] = 3; return r },
		"short":   func(r []byte) []byte { return r[:len(r)-1] },
		"long":    func(r []byte) []byte { return append(r, 0) },
		"hash":    func(r []byte) []byte { r[len(r)-1] ^= 1; return r },
		"balance": func(r []byte) []byte { r[len(r)-32-16] ^= 1; return r },
		"nonce":   func(r []byte) []byte { r[len(r)-32-1] ^= 1; return r },
		"supply":  func(r []byte) []byte { r[8+6+len(chain)+15] ^= 1; return r },
		"count":   func(r []byte) []byte { binary.BigEndian.PutUint32(r[8+6+len(chain)+16:], 65537); return r },
		"chain":   func(r []byte) []byte { r[8+6] = 0xff; return r },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "b0")
			s, err := Create(dir, genesis, chain)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			db, err := pebble.Open(dir, &pebble.Options{ErrorIfNotExists: true})
			if err != nil {
				t.Fatal(err)
			}
			raw, closer, err := db.Get(committedKey)
			if err != nil {
				t.Fatal(err)
			}
			bad := mutate(bytes.Clone(raw))
			if err := closer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := db.Set(committedKey, bad, pebble.Sync); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			if opened, err := OpenExisting(dir, chain); err == nil {
				opened.Close()
				t.Fatal("accepted corrupt record")
			}
		})
	}
	// V1 key is never silently treated as an absent V2 state.
	dir := filepath.Join(t.TempDir(), "v1")
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Set(v1Key, []byte("PLUTO02!"), pebble.Sync); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenExisting(dir, chain); err != ErrV1 {
		t.Fatal("V1 not rejected", err)
	}
	if _, err := OpenExisting(filepath.Join(t.TempDir(), "absent"), chain); err == nil {
		t.Fatal("absent DB accepted")
	}
	// A valid record with the wrong explicit chain context also fails.
	dir = filepath.Join(t.TempDir(), "wrong-chain")
	s, err := Create(dir, genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenExisting(dir, "other"); err == nil {
		t.Fatal("wrong chain accepted")
	}
	_ = a
	_ = b
}

func TestPhysicalOrderingOverflowAndHeight(t *testing.T) {
	genesis, _, a, _ := fixture(t)
	dir := filepath.Join(t.TempDir(), "b0")
	s, err := Create(dir, genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	base, _ := b0state.ParseGenesis(genesis, chain)
	makeCorrupt := func(logical []byte) {
		hash := sha256.Sum256(append([]byte("PLUTO/B0/STATE/V2\x00"), logical...))
		record := append([]byte(recordMagic), logical...)
		record = append(record, hash[:]...)
		db, err := pebble.Open(dir, &pebble.Options{ErrorIfNotExists: true})
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Set(committedKey, record, pebble.Sync); err != nil {
			t.Fatal(err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		if opened, err := OpenExisting(dir, chain); err == nil {
			opened.Close()
			t.Fatal("corrupt ordered state accepted")
		}
	}
	// Manufacture two accounts with a valid physical length and matching hash
	// but unsorted / duplicate records. Public NewState would reject these.
	baseLogical, _ := base.LogicalBytes()
	accountStart := 6 + len(chain) + 20
	for _, duplicate := range []bool{false, true} {
		logical := append(bytes.Clone(baseLogical), make([]byte, 36)...)
		binary.BigEndian.PutUint32(logical[6+len(chain)+16:], 2)
		copy(logical[accountStart+36:], logical[accountStart:accountStart+36])
		binary.BigEndian.PutUint64(logical[accountStart+36+20:], 0)
		if !duplicate {
			clear(logical[accountStart+36 : accountStart+36+20])
		}
		makeCorrupt(logical)
	}
	// Sum overflows even though the stored digest was recomputed to match.
	logical := append(bytes.Clone(baseLogical), make([]byte, 36)...)
	binary.BigEndian.PutUint32(logical[6+len(chain)+16:], 2)
	binary.BigEndian.PutUint64(logical[accountStart+20:], math.MaxUint64)
	copy(logical[accountStart+36:accountStart+36+20], a[:])
	logical[accountStart+36+19]++
	binary.BigEndian.PutUint64(logical[accountStart+36+20:], 1)
	makeCorrupt(logical)
	// A mismatched supply cannot be rescued by recomputing the digest.
	logical = bytes.Clone(baseLogical)
	binary.BigEndian.PutUint64(logical[6+len(chain)+8:], 99)
	makeCorrupt(logical)
	max, _ := b0state.NewState(chain, math.MaxInt64, 100, []b0state.Account{{Address: a, Balance: 100}})
	record, _ := encodeRecord(max)
	db, err := pebble.Open(dir, &pebble.Options{ErrorIfNotExists: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Set(committedKey, record, pebble.Sync); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenExisting(dir, chain)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Prepare(nil); err != b0state.ErrHeightOverflow {
		t.Fatal("height overflow", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func FuzzRecord(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("PLUTOS2!"))
	f.Fuzz(func(t *testing.T, raw []byte) { _, _ = decodeRecord(raw, "c") })
}

func TestLifecycleAndStorageErrorBranches(t *testing.T) {
	genesis, wire, _, _ := fixture(t)
	if _, err := Create("", genesis, chain); err != ErrMissing {
		t.Fatal(err)
	}
	if _, err := Create(filepath.Join(t.TempDir(), "bad"), []byte("invalid"), chain); err == nil {
		t.Fatal("bad genesis")
	}
	if _, err := OpenExisting("", chain); err != ErrMissing {
		t.Fatal(err)
	}
	parent := filepath.Join(t.TempDir(), "regular-file")
	if err := os.WriteFile(parent, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(filepath.Join(parent, "child"), genesis, chain); err == nil {
		t.Fatal("bad parent accepted")
	}

	emptyDir := filepath.Join(t.TempDir(), "empty-db")
	db, err := pebble.Open(emptyDir, &pebble.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenExisting(emptyDir, chain); err != ErrMissing {
		t.Fatal("missing record", err)
	}

	dir := filepath.Join(t.TempDir(), "b0")
	s, err := Create(dir, genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Pending(); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit(); err != ErrNoPending {
		t.Fatal(err)
	}
	if _, _, err := s.Prepare([][]byte{wire}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Prepare(nil); err != ErrPending {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Pending(); err != ErrClosed {
		t.Fatal(err)
	}
	if _, _, err := s.Prepare(nil); err != ErrClosed {
		t.Fatal(err)
	}
	if err := s.Commit(); err != ErrClosed {
		t.Fatal(err)
	}
	if err := s.Close(); err != ErrClosed {
		t.Fatal(err)
	}

	s, err = OpenExisting(dir, chain)
	if err != nil {
		t.Fatal(err)
	}
	s.pending = &b0state.State{} // synthetic invalid internal candidate
	if err := s.Commit(); err != b0state.ErrState {
		t.Fatal("invalid pending", err)
	}
	s.pending = nil
	s.committed = b0state.State{} // synthetic invalid committed state
	if _, _, err := s.Prepare(nil); err != b0state.ErrState {
		t.Fatal("invalid committed", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = OpenExisting(dir, chain)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Prepare(nil); err != nil {
		t.Fatal(err)
	}
	// Inject a returned write error without closing Pebble behind Store's lock.
	// A real I/O error can have an ambiguous durable outcome; this hook models
	// the old-record outcome while AfterSync tests the new-record outcome.
	s.syncWrite = func([]byte, []byte) error { return errors.New("injected Set error") }
	if err := s.Commit(); err == nil {
		t.Fatal("injected Set error accepted")
	}
	if snapshot, err := s.Committed(); err != ErrUncertain || snapshot.ChainID() != "" || snapshot.AppHash() != ([32]byte{}) {
		t.Fatal("Set error committed read did not fail closed", err)
	}
	if snapshot, yes, err := s.Pending(); err != ErrUncertain || yes || snapshot.ChainID() != "" || snapshot.AppHash() != ([32]byte{}) {
		t.Fatal("Set error pending read did not fail closed", err)
	}
	if s.committed.Height() != 0 || s.pending == nil || s.pending.Height() != 1 {
		t.Fatal("Set error promoted or discarded candidate")
	}
	if err := s.Commit(); err != ErrUncertain {
		t.Fatal("uncertain Set retry", err)
	}
	if _, _, err := s.Prepare(nil); err != ErrUncertain {
		t.Fatal("uncertain Set prepare", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenExisting(dir, chain)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := reopened.Committed()
	if err != nil || recovered.Height() != 0 {
		t.Fatal("old state recovery", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	// Oversized values are rejected before copying or allocating account arrays.
	dir = filepath.Join(t.TempDir(), "oversized")
	s, err = Create(dir, genesis, chain)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = pebble.Open(dir, &pebble.Options{ErrorIfNotExists: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Set(committedKey, bytes.Repeat([]byte{1}, maxRecordSize+1), pebble.Sync); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if opened, err := OpenExisting(dir, chain); err == nil {
		opened.Close()
		t.Fatal("oversized record accepted")
	}
}
