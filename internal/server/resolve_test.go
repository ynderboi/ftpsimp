package server

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveAllowsInside(t *testing.T) {
	root := t.TempDir()
	s := New(root, ":0", nil, Options{AuthOn: false})

	sub := filepath.Join(root, "ok")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := s.resolve("ok")
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, got)
	if err != nil || rel != "ok" {
		// Symlink resolution may change absolute form; ensure still under root.
		r2, err2 := filepath.Rel(mustAbs(t, root), got)
		if err2 != nil || strings.HasPrefix(r2, "..") {
			t.Fatalf("got %q under root %q", got, root)
		}
	}
}

func TestResolveBlocksSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation often needs admin on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "leak")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("cannot create symlink:", err)
	}
	s := New(root, ":0", nil, Options{AuthOn: false})
	if _, err := s.resolve("leak/secret.txt"); err == nil {
		t.Fatal("expected symlink escape to fail")
	}
}

func TestContentDispositionEscapes(t *testing.T) {
	got := contentDisposition(`evil"name.txt`)
	if strings.Contains(got, `filename="evil"name`) {
		t.Fatalf("unsafe disposition: %s", got)
	}
	if !strings.Contains(got, "filename*=") {
		t.Fatalf("missing RFC5987: %s", got)
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	if r, err := filepath.EvalSymlinks(a); err == nil {
		return r
	}
	return a
}
