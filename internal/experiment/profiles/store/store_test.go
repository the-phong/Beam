package researchv2store_test

import (
	"errors"
	"path/filepath"
	"testing"

	"example.com/pluto/internal/experiment/profiles"
	"example.com/pluto/internal/experiment/profiles/store"
	"example.com/pluto/internal/experiment/profiles/testfixture"
)

func TestS04ProfileGenesisBindingAndUncertainCommit(t *testing.T) {
	f, err := researchv2test.New("rv2-store-test")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "snapshot")
	s, err := researchv2store.Create(dir, f.Genesis, f.ChainID, researchv2.ProfileH)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := researchv2store.OpenExisting(dir, f.ChainID, f.Genesis, researchv2.ProfileC); err == nil {
		t.Fatal("wrong profile accepted")
	}
	other, err := researchv2test.New(f.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := researchv2store.OpenExisting(dir, f.ChainID, other.Genesis, researchv2.ProfileH); err == nil {
		t.Fatal("wrong genesis accepted")
	}
	s, err = researchv2store.OpenExistingWithOptions(dir, f.ChainID, f.Genesis, researchv2.ProfileH, researchv2store.Options{AfterSync: func() error { return errors.New("injected lost response") }})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.Signed(researchv2.ProfileH, 7, 0)
	if err != nil {
		t.Fatal(err)
	}
	results, _, err := s.PrepareTrusted([][]byte{raw}, false)
	if err != nil || len(results) != 1 || results[0].Code != researchv2.CodeOK {
		t.Fatalf("prepare: %v %+v", err, results)
	}
	if err := s.Commit(); err == nil {
		t.Fatal("injected uncertain commit succeeded")
	}
	if _, err := s.Committed(); !errors.Is(err, researchv2store.ErrUncertain) {
		t.Fatalf("committed read: %v", err)
	}
	if _, _, err := s.Pending(); !errors.Is(err, researchv2store.ErrUncertain) {
		t.Fatalf("pending read: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = researchv2store.OpenExisting(dir, f.ChainID, f.Genesis, researchv2.ProfileH)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, err := s.Committed()
	if err != nil || state.Height() != 1 {
		t.Fatalf("reopen validated height: %v %d", err, state.Height())
	}
	account, _ := state.Account(f.Sender.Binding.Account)
	if account.Balance != 93 || account.Nonce != 1 {
		t.Fatalf("recovered account %+v", account)
	}
}
