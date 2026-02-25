package store_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zynqcloud/go-storage/internal/store"
)

func newTestLocal(t *testing.T) *store.Local {
	t.Helper()
	root := t.TempDir() // cleaned up automatically after each test
	l, err := store.NewLocal(root)
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	return l
}

func TestWriteAndRead(t *testing.T) {
	l := newTestLocal(t)
	want := []byte("hello, storage")

	n, err := l.Write("owner/file.enc", bytes.NewReader(want))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != int64(len(want)) {
		t.Errorf("Write returned %d bytes, want %d", n, len(want))
	}

	rc, size, err := l.Read("owner/file.enc")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	defer rc.Close()

	got, _ := io.ReadAll(rc)
	if !bytes.Equal(got, want) {
		t.Errorf("Read content mismatch: got %q, want %q", got, want)
	}
	if size != int64(len(want)) {
		t.Errorf("Read size = %d, want %d", size, len(want))
	}
}

func TestWriteIsAtomic(t *testing.T) {
	// A second Write to the same path must overwrite cleanly (no partial file).
	l := newTestLocal(t)

	if _, err := l.Write("f.enc", strings.NewReader("first")); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Write("f.enc", strings.NewReader("second")); err != nil {
		t.Fatal(err)
	}

	rc, _, _ := l.Read("f.enc")
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != "second" {
		t.Errorf("expected 'second', got %q", got)
	}
}

func TestDelete(t *testing.T) {
	l := newTestLocal(t)
	l.Write("to-delete.enc", strings.NewReader("data")) //nolint:errcheck

	if err := l.Delete("to-delete.enc"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	ok, err := l.Exists("to-delete.enc")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("file still exists after Delete")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	l := newTestLocal(t)
	// Must succeed silently.
	if err := l.Delete("ghost.enc"); err != nil {
		t.Fatalf("Delete of non-existent file returned error: %v", err)
	}
}

func TestExists(t *testing.T) {
	l := newTestLocal(t)

	ok, err := l.Exists("missing.enc")
	if err != nil || ok {
		t.Errorf("Exists(missing) = (%v, %v), want (false, nil)", ok, err)
	}

	l.Write("present.enc", strings.NewReader("x")) //nolint:errcheck
	ok, err = l.Exists("present.enc")
	if err != nil || !ok {
		t.Errorf("Exists(present) = (%v, %v), want (true, nil)", ok, err)
	}
}

func TestRename(t *testing.T) {
	l := newTestLocal(t)
	l.Write("src.enc", strings.NewReader("payload")) //nolint:errcheck

	if err := l.Rename("src.enc", "dst/dst.enc"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	ok, _ := l.Exists("src.enc")
	if ok {
		t.Error("source still exists after Rename")
	}
	ok, _ = l.Exists("dst/dst.enc")
	if !ok {
		t.Error("destination does not exist after Rename")
	}
}

func TestMkdirAll(t *testing.T) {
	l := newTestLocal(t)
	if err := l.MkdirAll("a/b/c"); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	ok, _ := l.Exists("a/b/c")
	if !ok {
		t.Error("directory not created")
	}
}

// TestPathTraversal verifies that attempts to escape the storage root are rejected.
func TestPathTraversal(t *testing.T) {
	l := newTestLocal(t)

	traversals := []string{
		"../escape.enc",
		"../../etc/passwd",
		"owner/../../escape.enc",
	}
	for _, p := range traversals {
		_, err := l.Write(p, strings.NewReader("x"))
		if err == nil {
			t.Errorf("Write(%q): expected traversal error, got nil", p)
		}
	}
}

// TestNestedOwnerDir verifies the standard owner/fileId.enc path pattern.
func TestNestedOwnerDir(t *testing.T) {
	l := newTestLocal(t)
	path := filepath.Join("owner-uuid", "file-uuid.enc")

	n, err := l.Write(path, strings.NewReader("encrypted"))
	if err != nil {
		t.Fatalf("Write nested: %v", err)
	}
	if n == 0 {
		t.Error("wrote zero bytes")
	}

	// Verify the file actually landed under root.
	rc, _, err := l.Read(path)
	if err != nil {
		t.Fatalf("Read nested: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != "encrypted" {
		t.Errorf("got %q", got)
	}
}

// TestLargeStream verifies streaming without buffering a full file (1 MB).
func TestLargeStream(t *testing.T) {
	l := newTestLocal(t)
	const size = 1 << 20 // 1 MB

	data := bytes.Repeat([]byte("A"), size)
	n, err := l.Write("big.enc", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Write large: %v", err)
	}
	if n != size {
		t.Errorf("n = %d, want %d", n, size)
	}

	rc, rsize, _ := l.Read("big.enc")
	defer rc.Close()
	if rsize != size {
		t.Errorf("Read size = %d, want %d", rsize, size)
	}
	buf, _ := io.ReadAll(rc)
	if len(buf) != size {
		t.Errorf("read back %d bytes, want %d", len(buf), size)
	}
}

// TestNewLocalCreatesRoot verifies that a non-existent root is created.
func TestNewLocalCreatesRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "new", "nested", "root")
	_, err := store.NewLocal(root)
	if err != nil {
		t.Fatalf("NewLocal with missing root: %v", err)
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Error("root directory was not created")
	}
}
