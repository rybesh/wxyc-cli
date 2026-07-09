package auth

import (
	"context"
	"net/http"
)

// Transport is an http.RoundTripper that authenticates every request with a
// bearer JWT from Token, and transparently refreshes once on a 401. Installing
// it on the http.Client used by the API layer means no individual command can
// forget to authenticate or mishandle an expired token.
//
// Token and Refresh match TokenProvider.Token / TokenProvider.Refresh, so a
// provider wires straight in.
type Transport struct {
	Base    http.RoundTripper
	Token   func(ctx context.Context) (string, error)
	Refresh func(ctx context.Context) (string, error)
}

func (t *Transport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.Token(req.Context())
	if err != nil {
		return nil, err
	}

	res, err := t.base().RoundTrip(cloneWithAuth(req, tok))
	if err != nil || res.StatusCode != http.StatusUnauthorized {
		return res, err
	}

	// The JWT was rejected — likely expired between our expiry check and the
	// request landing. Refresh once and retry. Bodies are replayed via GetBody.
	res.Body.Close()
	tok, err = t.Refresh(req.Context())
	if err != nil {
		return nil, err
	}
	retry, err := rewind(req)
	if err != nil {
		return nil, err
	}
	return t.base().RoundTrip(cloneWithAuth(retry, tok))
}

func cloneWithAuth(req *http.Request, token string) *http.Request {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}

// rewind returns a request whose body is reset to the start, using GetBody when
// the original carried a replayable body.
func rewind(req *http.Request) (*http.Request, error) {
	if req.Body == nil || req.GetBody == nil {
		return req, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	r := req.Clone(req.Context())
	r.Body = body
	return r, nil
}
