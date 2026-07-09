package cmd

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rybesh/wxyc-cli/internal/auth"
	"github.com/rybesh/wxyc-cli/internal/safety"
)

// runCLI executes the command tree with args and returns stdout and the error.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out, errbuf bytes.Buffer
	app := &App{stdout: &out, stderr: &errbuf}
	root := newRoot(app)
	root.SetArgs(args)
	root.SetOut(&out)
	root.SetErr(&errbuf)
	err := root.Execute()
	return out.String(), err
}

func TestWriteGate_BlocksMutatingCommandByDefault(t *testing.T) {
	// A static JWT avoids the keyring/store and network for this path; the
	// gate must reject before RunE, so no request is ever attempted.
	t.Setenv("WXYC_JWT", "unused")
	t.Setenv("WXYC_ALLOW_WRITE", "")

	_, err := runCLI(t, "bin", "add", "45029")
	var blocked *safety.BlockedError
	if !errors.As(err, &blocked) {
		t.Fatalf("err = %v, want *BlockedError", err)
	}
	if mapExit(err) != ExitBlocked {
		t.Errorf("exit = %d, want ExitBlocked(%d)", mapExit(err), ExitBlocked)
	}
}

func TestWriteGate_AllowsMutatingCommandWhenUnlocked(t *testing.T) {
	var posted int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/djs/bin" {
			atomic.AddInt32(&posted, 1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("WXYC_API_URL", srv.URL)
	t.Setenv("WXYC_JWT", "unused")

	_, err := runCLI(t, "bin", "add", "45029", "--write")
	if err != nil {
		t.Fatalf("unlocked write err = %v", err)
	}
	if posted != 1 {
		t.Errorf("POST /djs/bin hit %d times, want 1", posted)
	}
}

func TestWriteGate_EnvUnlockAlsoWorks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("WXYC_API_URL", srv.URL)
	t.Setenv("WXYC_JWT", "unused")
	t.Setenv("WXYC_ALLOW_WRITE", "1")

	if _, err := runCLI(t, "bin", "add", "1"); err != nil {
		t.Errorf("WXYC_ALLOW_WRITE=1 should permit write, got %v", err)
	}
}

func TestRotation_JSONKeepsFullShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/library/rotation" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`[{"artist_name":"X","rotation_bin":"Heavy","reconciled_identity":{"discogs_artist_id":42}}]`))
	}))
	defer srv.Close()
	t.Setenv("WXYC_API_URL", srv.URL)
	t.Setenv("WXYC_JWT", "unused")

	out, err := runCLI(t, "library", "rotation", "--json")
	if err != nil {
		t.Fatal(err)
	}
	// Nested identity the table never renders must survive in --json.
	if !strings.Contains(out, "reconciled_identity") || !strings.Contains(out, "42") {
		t.Errorf("--json dropped nested fields:\n%s", out)
	}
}

func TestLoginDevice_StoresSessionToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/device/code":
			w.Write([]byte(`{"device_code":"d","user_code":"WXYC-9999",` +
				`"verification_uri":"https://dj.wxyc.org/device-auth","expires_in":300,"interval":5}`))
		case "/auth/device/token":
			// Approve on the first poll so no sleep occurs.
			w.Write([]byte(`{"access_token":"sess-device-xyz","token_type":"Bearer","expires_in":43200}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Isolate the credential store to a temp dir (file fallback; no keychain).
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("WXYC_API_URL", srv.URL)

	out, err := runCLI(t, "login", "--device")
	if err != nil {
		t.Fatalf("login --device err = %v", err)
	}
	if !strings.Contains(out, "signed in") {
		t.Errorf("stdout = %q, want success line", out)
	}

	// The device session token must be persisted for the default profile.
	got, err := auth.FileStore{Dir: filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "wxyc-cli")}.Load("default")
	if err != nil {
		t.Fatalf("session not stored: %v", err)
	}
	if got != "sess-device-xyz" {
		t.Errorf("stored token = %q, want sess-device-xyz", got)
	}
}

func TestReadCommand_NeedsNoUnlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"entries":[{"id":1,"entry_type":"track","artist_name":"A","track_title":"T","album_title":"Al"}]}`))
	}))
	defer srv.Close()
	t.Setenv("WXYC_API_URL", srv.URL)
	t.Setenv("WXYC_JWT", "unused")

	out, err := runCLI(t, "flowsheet", "tail", "--json")
	if err != nil {
		t.Fatalf("read err = %v", err)
	}
	if !strings.Contains(out, `"artist_name": "A"`) {
		t.Errorf("json output missing entry:\n%s", out)
	}
}
