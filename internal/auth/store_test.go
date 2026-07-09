package auth

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestFileStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := FileStore{Dir: dir}

	if _, err := s.Load("dj"); !errors.Is(err, ErrNoSession) {
		t.Fatalf("Load on empty = %v, want ErrNoSession", err)
	}
	if err := s.Save("dj", "sess-token"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("dj")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sess-token" {
		t.Errorf("Load = %q, want sess-token", got)
	}
	if err := s.Clear("dj"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("dj"); !errors.Is(err, ErrNoSession) {
		t.Errorf("Load after Clear = %v, want ErrNoSession", err)
	}
}

func TestFileStore_FileIsPrivate(t *testing.T) {
	dir := t.TempDir()
	s := FileStore{Dir: dir}
	if err := s.Save("dj", "secret"); err != nil {
		t.Fatal(err)
	}
	var info fs.FileInfo
	err := filepath.Walk(dir, func(p string, fi fs.FileInfo, err error) error {
		if err == nil && !fi.IsDir() {
			info = fi
		}
		return err
	})
	if err != nil || info == nil {
		t.Fatalf("could not stat token file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("token file perms = %o, want 600", perm)
	}
}

func TestFileStore_ClearMissingIsNoError(t *testing.T) {
	s := FileStore{Dir: t.TempDir()}
	if err := s.Clear("never-saved"); err != nil {
		t.Errorf("Clear on missing = %v, want nil", err)
	}
}

func TestKeyringStore_RoundTrip(t *testing.T) {
	keyring.MockInit()
	s := KeyringStore{}

	if _, err := s.Load("dj"); !errors.Is(err, ErrNoSession) {
		t.Fatalf("Load on empty = %v, want ErrNoSession", err)
	}
	if err := s.Save("dj", "kr-token"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("dj")
	if err != nil {
		t.Fatal(err)
	}
	if got != "kr-token" {
		t.Errorf("Load = %q, want kr-token", got)
	}
	if err := s.Clear("dj"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("dj"); !errors.Is(err, ErrNoSession) {
		t.Errorf("Load after Clear = %v, want ErrNoSession", err)
	}
}

// Ensure the file fallback lands somewhere real when Dir is defaulted.
func TestDefaultFileStoreDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := defaultStoreDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "wxyc-cli" {
		t.Errorf("dir = %q, want .../wxyc-cli", dir)
	}
}
