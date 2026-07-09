package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SignInStrategy obtains a session token. Implementations differ in how the
// user proves identity (password now; device-authorization / OIDC later), but
// all yield the same session token the rest of the CLI depends on.
type SignInStrategy interface {
	Login(ctx context.Context) (sessionToken string, err error)
}

// PasswordStrategy signs in with email/username + password via better-auth's
// /sign-in endpoints and returns the session token from the set-auth-token
// response header (the bearer plugin). The password is used once here and not
// retained.
type PasswordStrategy struct {
	AuthBase string
	HTTP     *http.Client
	Ident    string
	Password string
}

func (s PasswordStrategy) Login(ctx context.Context) (string, error) {
	path, field := "/sign-in/username", "username"
	if strings.Contains(s.Ident, "@") {
		path, field = "/sign-in/email", "email"
	}

	payload, err := json.Marshal(map[string]string{field: s.Ident, "password": s.Password})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.AuthBase+path, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("signing in: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return "", fmt.Errorf("sign-in failed: HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	token := res.Header.Get("set-auth-token")
	if token == "" {
		return "", fmt.Errorf("sign-in succeeded but no session token was returned (missing set-auth-token header)")
	}
	return token, nil
}
