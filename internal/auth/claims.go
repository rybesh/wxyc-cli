package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Audience models the JWT `aud` claim, which the spec allows to be either a
// single string or an array of strings.
type Audience []string

func (a *Audience) UnmarshalJSON(b []byte) error {
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		*a = Audience{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return err
	}
	*a = many
	return nil
}

// Claims is the subset of the better-auth JWT payload the CLI cares about.
//
// The CLI does not verify the token signature — the backend is the sole
// authority and rejects anything invalid with 401. These claims are read only
// to display identity (whoami) and to decide when to proactively refresh.
type Claims struct {
	Sub   string   `json:"sub"`
	Email string   `json:"email"`
	Role  string   `json:"role"`
	Iss   string   `json:"iss"`
	Aud   Audience `json:"aud"`
	Exp   int64    `json:"exp"`
	Iat   int64    `json:"iat"`
}

// ParseClaims decodes the payload segment of a JWT without verifying it.
func ParseClaims(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, fmt.Errorf("malformed JWT: expected 3 segments, got %d", len(parts))
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("decoding JWT payload: %w", err)
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return Claims{}, fmt.Errorf("parsing JWT payload: %w", err)
	}
	return c, nil
}

// Expired reports whether the token is at or past its expiry, treating a token
// within skew of expiry as already expired so the caller refreshes before a
// request rather than racing the boundary.
func (c Claims) Expired(now time.Time, skew time.Duration) bool {
	return c.Exp <= now.Add(skew).Unix()
}
