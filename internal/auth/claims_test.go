package auth

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

// makeJWT builds an unsigned-looking token: header.payload.signature where only
// the payload segment carries meaning. The CLI never verifies the signature, so
// the header and signature are arbitrary.
func makeJWT(t *testing.T, payload map[string]any) string {
	t.Helper()
	enc := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return enc(map[string]string{"alg": "EdDSA", "typ": "JWT"}) + "." + enc(payload) + ".sig"
}

func TestParseClaims_ReadsFields(t *testing.T) {
	tok := makeJWT(t, map[string]any{
		"sub":   "abc123",
		"email": "dj@example.com",
		"role":  "dj",
		"iss":   "https://api.wxyc.org",
		"aud":   "https://api.wxyc.org",
		"exp":   1783608994,
		"iat":   1783608094,
	})
	c, err := ParseClaims(tok)
	if err != nil {
		t.Fatalf("ParseClaims() error = %v", err)
	}
	if c.Sub != "abc123" || c.Email != "dj@example.com" || c.Role != "dj" {
		t.Errorf("unexpected claims: %+v", c)
	}
	if c.Exp != 1783608994 {
		t.Errorf("Exp = %d, want 1783608994", c.Exp)
	}
}

func TestParseClaims_AudienceStringOrArray(t *testing.T) {
	// JWT `aud` may be a single string or an array; both must parse.
	str := makeJWT(t, map[string]any{"aud": "https://api.wxyc.org"})
	arr := makeJWT(t, map[string]any{"aud": []string{"https://api.wxyc.org", "other"}})
	for _, tok := range []string{str, arr} {
		if _, err := ParseClaims(tok); err != nil {
			t.Errorf("ParseClaims() error = %v", err)
		}
	}
}

func TestParseClaims_Malformed(t *testing.T) {
	for _, tok := range []string{"", "onlyonesegment", "two.segments", "a.!!notbase64!!.c"} {
		if _, err := ParseClaims(tok); err == nil {
			t.Errorf("ParseClaims(%q) = nil error, want error", tok)
		}
	}
}

func TestClaims_Expired(t *testing.T) {
	now := time.Unix(1000, 0)
	tests := []struct {
		name string
		exp  int64
		skew time.Duration
		want bool
	}{
		{name: "future beyond skew is valid", exp: 2000, skew: 30 * time.Second, want: false},
		{name: "already past is expired", exp: 900, skew: 0, want: true},
		{name: "within skew window is treated as expired", exp: 1010, skew: 30 * time.Second, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Claims{Exp: tt.exp}
			if got := c.Expired(now, tt.skew); got != tt.want {
				t.Errorf("Expired() = %v, want %v", got, tt.want)
			}
		})
	}
}
