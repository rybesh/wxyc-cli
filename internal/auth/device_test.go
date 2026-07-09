package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// deviceServer scripts /device/code once and /device/token as a sequence of
// canned responses (one per poll), so tests drive the full flow deterministically.
func deviceServer(t *testing.T, code map[string]any, tokenSeq []struct {
	status int
	body   map[string]any
}) (*httptest.Server, *int32) {
	t.Helper()
	var polls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/device/code":
			var body map[string]any
			raw, _ := io.ReadAll(r.Body)
			json.Unmarshal(raw, &body)
			if body["client_id"] == nil {
				t.Errorf("/device/code missing client_id")
			}
			json.NewEncoder(w).Encode(code)
		case "/device/token":
			i := int(atomic.AddInt32(&polls, 1)) - 1
			if i >= len(tokenSeq) {
				i = len(tokenSeq) - 1
			}
			w.WriteHeader(tokenSeq[i].status)
			json.NewEncoder(w).Encode(tokenSeq[i].body)
		default:
			http.NotFound(w, r)
		}
	}))
	return srv, &polls
}

func tok(status int, body map[string]any) struct {
	status int
	body   map[string]any
} {
	return struct {
		status int
		body   map[string]any
	}{status, body}
}

func newDeviceStrategy(base string, out io.Writer) DeviceStrategy {
	return DeviceStrategy{
		AuthBase: base,
		HTTP:     http.DefaultClient,
		ClientID: "wxyc-cli",
		Out:      out,
		Sleep:    func(int) {}, // no real delay in tests
	}
}

func TestDeviceStrategy_PollsUntilApproved(t *testing.T) {
	code := map[string]any{
		"device_code": "dev-123", "user_code": "WXYC-1234",
		"verification_uri":          "https://dj.wxyc.org/device-auth",
		"verification_uri_complete": "https://dj.wxyc.org/device-auth?user_code=WXYC-1234",
		"expires_in":                300, "interval": 5,
	}
	seq := []struct {
		status int
		body   map[string]any
	}{
		tok(400, map[string]any{"error": "authorization_pending"}),
		tok(400, map[string]any{"error": "authorization_pending"}),
		tok(200, map[string]any{"access_token": "sess-from-device", "token_type": "Bearer", "expires_in": 43200}),
	}
	srv, polls := deviceServer(t, code, seq)
	defer srv.Close()

	var out bytes.Buffer
	s := newDeviceStrategy(srv.URL, &out)
	token, err := s.Login(context.Background())
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token != "sess-from-device" {
		t.Errorf("token = %q, want sess-from-device", token)
	}
	if *polls != 3 {
		t.Errorf("polled %d times, want 3", *polls)
	}
	// The user must be shown the code to approve on their phone.
	if !strings.Contains(out.String(), "WXYC-1234") {
		t.Errorf("instructions missing user_code:\n%s", out.String())
	}
}

func TestDeviceStrategy_SlowDownWidensInterval(t *testing.T) {
	code := map[string]any{"device_code": "d", "user_code": "U", "verification_uri": "x", "expires_in": 300, "interval": 5}
	seq := []struct {
		status int
		body   map[string]any
	}{
		tok(400, map[string]any{"error": "authorization_pending"}),
		tok(400, map[string]any{"error": "slow_down"}),
		tok(200, map[string]any{"access_token": "ok", "token_type": "Bearer", "expires_in": 43200}),
	}
	srv, _ := deviceServer(t, code, seq)
	defer srv.Close()

	var slept []int
	s := newDeviceStrategy(srv.URL, io.Discard)
	s.Sleep = func(d int) { slept = append(slept, d) }
	if _, err := s.Login(context.Background()); err != nil {
		t.Fatal(err)
	}
	// First sleep is the base 5s (after pending); after slow_down it must
	// widen (RFC 8628 §3.5).
	if len(slept) != 2 || slept[0] != 5 || slept[1] != 10 {
		t.Errorf("expected sleeps [5 10] (base then widened), got %v", slept)
	}
}

func TestDeviceStrategy_Denied(t *testing.T) {
	code := map[string]any{"device_code": "d", "user_code": "U", "verification_uri": "x", "expires_in": 300, "interval": 5}
	seq := []struct {
		status int
		body   map[string]any
	}{tok(400, map[string]any{"error": "access_denied"})}
	srv, _ := deviceServer(t, code, seq)
	defer srv.Close()

	s := newDeviceStrategy(srv.URL, io.Discard)
	_, err := s.Login(context.Background())
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("err = %v, want a denial error", err)
	}
}

func TestDeviceStrategy_Expired(t *testing.T) {
	code := map[string]any{"device_code": "d", "user_code": "U", "verification_uri": "x", "expires_in": 300, "interval": 5}
	seq := []struct {
		status int
		body   map[string]any
	}{tok(400, map[string]any{"error": "expired_token"})}
	srv, _ := deviceServer(t, code, seq)
	defer srv.Close()

	s := newDeviceStrategy(srv.URL, io.Discard)
	_, err := s.Login(context.Background())
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("err = %v, want an expiry error", err)
	}
}
