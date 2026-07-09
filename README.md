# wxyc-cli

A read-only-by-default CLI for the [WXYC backend API](https://api.wxyc.org),
designed to be driven by both humans and agents. Authenticates with your
dj.wxyc.org credentials.

## Install

```sh
go build -o wxyc ./cmd/wxyc
# or: go install github.com/rybesh/wxyc-cli/cmd/wxyc@latest
```

## Quick start

```sh
wxyc login                       # prompts for email/username + password (hidden)
wxyc whoami                      # shows your identity and role

wxyc library search --artist "aphex twin"
wxyc library genres
wxyc library formats
wxyc library rotation            # --json keeps the full nested shape
wxyc flowsheet tail -n 20
wxyc labels search "june appal"
wxyc labels list
wxyc schedule
wxyc bin list
```

Add `--json` to any command for machine-readable output. The rule is uniform:
**the table is a curated projection; `--json` is the API's response verbatim**
(byte-for-byte, including fields the table doesn't show), so an agent never
misses a field the CLI didn't bother to model.

## Auth model

Your **password is never stored**. `login` exchanges it once for a long-lived
*session token* (kept in the OS keychain, or a `0600` file if no keychain is
available). Every request transparently exchanges that session token for a
short-lived signed JWT and refreshes on expiry. This mirrors the two-call
handshake the website uses: `POST /auth/sign-in/*` → `GET /auth/token`.

An agent or CI job that manages its own token can skip the keychain entirely by
exporting a JWT directly:

```sh
export WXYC_JWT="$(...)"          # used verbatim; no login/keychain needed
```

## Read-only by default

Mutating commands (`bin add`, and future flowsheet/catalog writes) are blocked
unless you explicitly unlock writes:

```sh
wxyc bin add 45029               # exits 2 (blocked)
wxyc bin add 45029 --write       # permitted
WXYC_ALLOW_WRITE=1 wxyc bin add 45029   # permitted (scripted)
```

This is a client-side guard *in addition to* the server's own role check, so an
agent can't mutate station state by accident. Your `dj` role also can't perform
catalog writes regardless — the server returns 403.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | success |
| 1 | unclassified error |
| 2 | write blocked (read-only; pass `--write`) |
| 3 | not authenticated (no session, or 401) |
| 4 | forbidden (authenticated, but role lacks permission) |
| 5 | not found |

## Configuration

| Env var | Default | Purpose |
|---------|---------|---------|
| `WXYC_API_URL` | `https://api.wxyc.org` | backend REST base |
| `WXYC_AUTH_URL` | `<API>/auth` | better-auth base |
| `WXYC_PROFILE` | `default` | credential profile (keychain namespace) |
| `WXYC_JWT` | — | supply a JWT directly, bypassing login |
| `WXYC_ALLOW_WRITE` | — | `1` unlocks mutating commands |

## Roadmap

- **v2 auth:** device-authorization (QR) sign-in — no password in the CLI at
  all; approve from the iOS app. The `SignInStrategy` interface is already in
  place for it.
- OIDC + PKCE as a registered trusted client.
- Remaining read surface (`djs playlists`, label/genre detail).

## Development

```sh
go test ./...        # red-green TDD throughout
go vet ./...
```

Layout: `cmd/` (cobra commands) → `internal/api` (typed client),
`internal/auth` (sign-in, token exchange, transport, session store),
`internal/safety` (write gate), `internal/output` (renderers),
`internal/config`.
