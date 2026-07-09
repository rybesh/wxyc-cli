package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// TokenProvider exchanges a persisted session token for a short-lived signed
// JWT via GET {AuthBase}/token, and caches the JWT until it nears expiry.
//
// It is safe for concurrent use.
type TokenProvider struct {
	AuthBase string                 // e.g. https://api.wxyc.org/auth
	HTTP     *http.Client           // request client (injectable for tests)
	Session  func() (string, error) // returns the current session token
	Skew     time.Duration          // refresh this long before exp
	Now      func() time.Time       // clock (injectable for tests)

	mu     sync.Mutex
	jwt    string
	claims Claims
}

func (p *TokenProvider) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

// Token returns a cached JWT if it is still fresh, otherwise exchanges the
// session token for a new one.
func (p *TokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.jwt != "" && !p.claims.Expired(p.now(), p.Skew) {
		return p.jwt, nil
	}
	return p.refreshLocked(ctx)
}

// Refresh forces a new exchange regardless of cache state. Used by the
// transport when the backend rejects the current JWT with 401.
func (p *TokenProvider) Refresh(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refreshLocked(ctx)
}

func (p *TokenProvider) refreshLocked(ctx context.Context) (string, error) {
	session, err := p.Session()
	if err != nil {
		return "", fmt.Errorf("reading session token: %w", err)
	}
	if session == "" {
		return "", fmt.Errorf("no session token; run `wxyc login` first")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.AuthBase+"/token", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+session)

	res, err := p.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchanging session for JWT: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return "", fmt.Errorf("token exchange failed: HTTP %d: %s", res.StatusCode, body)
	}

	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}
	if payload.Token == "" {
		return "", fmt.Errorf("token exchange returned empty token")
	}

	claims, err := ParseClaims(payload.Token)
	if err != nil {
		return "", fmt.Errorf("parsing exchanged JWT: %w", err)
	}
	p.jwt = payload.Token
	p.claims = claims
	return p.jwt, nil
}
