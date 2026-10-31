package store

import (
	"path/filepath"
	"testing"
)

func TestDurableReloadAndExclusiveOpen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "db")
	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s, err := db.Load(); err != nil || s != nil {
		t.Fatalf("uninitialized: %v %v", s, err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("concurrent writable open succeeded")
	}
	s, _ := Genesis("pluto-test")
	if err := db.Save(s); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	got, err := db.Load()
	if err != nil || got == nil || *got != s {
		t.Fatalf("reload: %v %v", got, err)
	}
}
