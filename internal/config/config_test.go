package config

import "testing"

func TestResolve_Defaults(t *testing.T) {
	c := Resolve(func(string) string { return "" })
	if c.APIBase != "https://api.wxyc.org" {
		t.Errorf("APIBase = %q", c.APIBase)
	}
	if c.AuthBase != "https://api.wxyc.org/auth" {
		t.Errorf("AuthBase = %q, want derived from APIBase", c.AuthBase)
	}
	if c.Profile != "default" {
		t.Errorf("Profile = %q, want default", c.Profile)
	}
}

func TestResolve_EnvOverrides(t *testing.T) {
	env := map[string]string{
		"WXYC_API_URL":  "http://localhost:8080",
		"WXYC_AUTH_URL": "http://localhost:8082/auth",
		"WXYC_PROFILE":  "staging",
	}
	c := Resolve(func(k string) string { return env[k] })
	if c.APIBase != "http://localhost:8080" || c.AuthBase != "http://localhost:8082/auth" || c.Profile != "staging" {
		t.Errorf("overrides not applied: %+v", c)
	}
}

func TestResolve_AuthDerivesFromCustomAPI(t *testing.T) {
	env := map[string]string{"WXYC_API_URL": "http://localhost:8080/"}
	c := Resolve(func(k string) string { return env[k] })
	// Trailing slash trimmed, /auth appended.
	if c.AuthBase != "http://localhost:8080/auth" {
		t.Errorf("AuthBase = %q, want http://localhost:8080/auth", c.AuthBase)
	}
}
