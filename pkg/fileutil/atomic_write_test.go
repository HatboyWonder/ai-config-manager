package fileutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteCreatesAndReplacesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	path := filepath.Join(dir, "state.json")
	if err := AtomicWrite(path, []byte("first"), 0644); err != nil {
		t.Fatalf("first AtomicWrite() failed: %v", err)
	}

	if err := AtomicWrite(path, []byte("second"), 0644); err != nil {
		t.Fatalf("second AtomicWrite() failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading result: %v", err)
	}
	if !bytes.Equal(got, []byte("second")) {
		t.Fatalf("content = %q, want %q", string(got), "second")
	}
}

func TestAtomicWriteDoesNotCreateParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "state.json")

	if err := AtomicWrite(path, []byte("data"), 0644); err == nil {
		t.Fatalf("expected error for missing parent directory")
	}
}

func TestAtomicWriteAppliesPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	path := filepath.Join(dir, "manifest.yaml")
	if err := AtomicWrite(path, []byte("version: 1\n"), 0600); err != nil {
		t.Fatalf("AtomicWrite() failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o, want %o", info.Mode().Perm(), os.FileMode(0600))
	}
}
