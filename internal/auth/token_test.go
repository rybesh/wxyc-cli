package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// tokenServer stands in for GET {authBase}/token. It records the bearer it saw
// and how many times it was hit, and returns a JWT whose exp is `expUnix`.
func tokenServer(t *testing.T, expUnix int64, gotBearer *string, hits *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(hits, 1)
		if gotBearer != nil {
			*gotBearer = r.Header.Get("Authorization")
		}
		jwt := makeJWT(t, map[string]any{"sub": "u1", "role": "dj", "exp": expUnix})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"token": jwt})
	}))
}

func newProvider(base, session string, now func() time.Time) *TokenProvider {
	return &TokenProvider{
		AuthBase: base,
		HTTP:     http.DefaultClient,
		Session:  func() (string, error) { return session, nil },
		Skew:     30 * time.Second,
		Now:      now,
	}
}

func TestTokenProvider_ExchangesAndSendsSessionBearer(t *testing.T) {
	var bearer string
	var hits int32
	srv := tokenServer(t, time.Now().Add(time.Hour).Unix(), &bearer, &hits)
	defer srv.Close()

	p := newProvider(srv.URL, "sess-abc", time.Now)
	got, err := p.Token(context.Background())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if got == "" {
		t.Fatal("Token() returned empty JWT")
	}
	if bearer != "Bearer sess-abc" {
		t.Errorf("exchange Authorization = %q, want %q", bearer, "Bearer sess-abc")
	}
}

func TestTokenProvider_CachesUntilExpiry(t *testing.T) {
	var hits int32
	srv := tokenServer(t, time.Now().Add(time.Hour).Unix(), nil, &hits)
	defer srv.Close()

	p := newProvider(srv.URL, "sess", time.Now)
	for i := 0; i < 3; i++ {
		if _, err := p.Token(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if hits != 1 {
		t.Errorf("server hit %d times, want 1 (cached)", hits)
	}
}

func TestTokenProvider_RefreshesWhenExpired(t *testing.T) {
	var hits int32
	base := time.Now()
	// JWT expires 10 minutes out.
	srv := tokenServer(t, base.Add(10*time.Minute).Unix(), nil, &hits)
	defer srv.Close()

	clock := base
	p := newProvider(srv.URL, "sess", func() time.Time { return clock })

	if _, err := p.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Advance past expiry; next call must re-fetch.
	clock = base.Add(20 * time.Minute)
	if _, err := p.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Errorf("server hit %d times, want 2 (refresh after expiry)", hits)
	}
}

func TestTokenProvider_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newProvider(srv.URL, "sess", time.Now)
	if _, err := p.Token(context.Background()); err == nil {
		t.Fatal("Token() = nil error on 401 exchange, want error")
	}
}
