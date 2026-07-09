// Package config resolves runtime configuration from environment variables,
// with sensible production defaults.
package config

import "strings"

// Config holds resolved endpoints and the active profile.
type Config struct {
	APIBase  string // backend REST base, e.g. https://api.wxyc.org
	AuthBase string // better-auth base, e.g. https://api.wxyc.org/auth
	Profile  string // credential profile name
}

const (
	defaultAPIBase = "https://api.wxyc.org"
	defaultProfile = "default"
)

// Resolve builds a Config from the given environment lookup (os.Getenv in
// production; a map in tests). WXYC_AUTH_URL defaults to WXYC_API_URL + "/auth".
func Resolve(getenv func(string) string) Config {
	api := firstNonEmpty(getenv("WXYC_API_URL"), defaultAPIBase)
	api = strings.TrimRight(api, "/")

	auth := getenv("WXYC_AUTH_URL")
	if auth == "" {
		auth = api + "/auth"
	}
	auth = strings.TrimRight(auth, "/")

	return Config{
		APIBase:  api,
		AuthBase: auth,
		Profile:  firstNonEmpty(getenv("WXYC_PROFILE"), defaultProfile),
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
