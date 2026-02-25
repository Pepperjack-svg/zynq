package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Local stores files on the local filesystem under a configurable root directory.
//
// Cross-platform notes:
//   - Uses filepath (not path) throughout so the OS separator is always correct.
//   - File permission bits (0o750 / 0o640) are silently ignored on Windows; ACLs
//     govern access there. The values are retained so Unix deployments remain secure.
//   - os.Rename is used for atomic writes. On Windows it calls MoveFileExW with
//     MOVEFILE_REPLACE_EXISTING, which is safe on the same volume.
type Local struct {
	root string
}

// NewLocal creates a Local backend rooted at root, creating the directory if needed.
func NewLocal(root string) (*Local, error) {
	// Use os.MkdirAll so the call is idempotent across restarts.
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create storage root %q: %w", root, err)
	}
	// Resolve to an absolute path so all subsequent filepath.Rel checks are stable.
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}
	return &Local{root: absRoot}, nil
}

// abs resolves a caller-supplied logical path to a concrete filesystem path.
//
// Design:
//   - filepath.FromSlash converts any forward slashes to the OS separator so
//     the code handles Unix-style logical paths on Windows without issue.
//   - We do NOT prepend a separator before calling filepath.Clean.
//     The old pattern filepath.Clean("/"+path) creates an absolute path fragment;
//     on Windows filepath.Join(root, `\foo`) treats `\foo` as absolute and returns
//     `\foo`, discarding root entirely.
//   - filepath.Rel verifies the result still lives under root, catching any edge
//     cases that slip past isValidID() in the handler layer.
func (l *Local) abs(path string) (string, error) {
	joined := filepath.Join(l.root, filepath.Clean(filepath.FromSlash(path)))
	rel, err := filepath.Rel(l.root, joined)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %q escapes storage root", path)
	}
	return joined, nil
}

// Write streams r to path using a temp-file + atomic rename.
func (l *Local) Write(path string, r io.Reader) (int64, error) {
	dest, err := l.abs(path)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return 0, fmt.Errorf("mkdir %q: %w", filepath.Dir(dest), err)
	}

	tmp := dest + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return 0, fmt.Errorf("open tmp %q: %w", tmp, err)
	}

	n, werr := io.Copy(f, r)
	cerr := f.Close()

	if werr != nil {
		os.Remove(tmp) //nolint:errcheck
		return 0, fmt.Errorf("stream write: %w", werr)
	}
	if cerr != nil {
		os.Remove(tmp) //nolint:errcheck
		return 0, fmt.Errorf("flush: %w", cerr)
	}

	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return 0, fmt.Errorf("rename to %q: %w", dest, err)
	}
	return n, nil
}

// Read opens path for sequential reading. Caller must close the returned ReadCloser.
func (l *Local) Read(path string) (io.ReadCloser, int64, error) {
	abs, err := l.abs(path)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}
	return f, info.Size(), nil
}

// Delete removes path recursively. Silently succeeds on ENOENT.
func (l *Local) Delete(path string) error {
	abs, err := l.abs(path)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(abs); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Exists reports whether path exists under root.
func (l *Local) Exists(path string) (bool, error) {
	abs, err := l.abs(path)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(abs)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// Rename moves src to dst atomically where the OS permits.
func (l *Local) Rename(src, dst string) error {
	absSrc, err := l.abs(src)
	if err != nil {
		return err
	}
	absDst, err := l.abs(dst)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absDst), 0o750); err != nil {
		return err
	}
	return os.Rename(absSrc, absDst)
}

// MkdirAll creates path and all parents under root.
func (l *Local) MkdirAll(path string) error {
	abs, err := l.abs(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(abs, 0o750)
}
