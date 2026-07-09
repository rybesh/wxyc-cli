package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// deviceGrantType is the RFC 8628 device-code grant identifier.
const deviceGrantType = "urn:ietf:params:oauth:grant-type:device_code"

// DeviceStrategy implements RFC 8628 device authorization: it requests a
// user_code, shows it for the DJ to approve from the iOS app, and polls until
// the browser-equivalent session is issued. The password never touches the
// CLI. The returned access_token is a normal session token, so it plugs into
// TokenProvider exactly like the password flow's session token.
type DeviceStrategy struct {
	AuthBase string
	HTTP     *http.Client
	ClientID string
	Out      io.Writer     // where approval instructions are written
	Sleep    func(sec int) // injectable for tests; defaults to time.Sleep
}

type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type deviceTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
}

func (s DeviceStrategy) sleep(sec int) {
	if s.Sleep != nil {
		s.Sleep(sec)
		return
	}
	time.Sleep(time.Duration(sec) * time.Second)
}

func (s DeviceStrategy) Login(ctx context.Context) (string, error) {
	code, err := s.requestCode(ctx)
	if err != nil {
		return "", err
	}
	s.display(code)

	interval := code.Interval
	if interval <= 0 {
		interval = 5
	}
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		resp, err := s.poll(ctx, code.DeviceCode)
		if err != nil {
			return "", err
		}
		switch {
		case resp.AccessToken != "":
			fmt.Fprintln(s.Out, "approved.")
			return resp.AccessToken, nil
		case resp.Error == "authorization_pending":
			s.sleep(interval)
		case resp.Error == "slow_down":
			// RFC 8628 §3.5: widen the interval by 5s and keep polling.
			interval += 5
			s.sleep(interval)
		case resp.Error == "access_denied":
			return "", fmt.Errorf("sign-in was denied from the approving device")
		case resp.Error == "expired_token":
			return "", fmt.Errorf("the code expired before it was approved; run login again")
		default:
			return "", fmt.Errorf("device authorization failed: %s", resp.Error)
		}
	}
}

func (s DeviceStrategy) requestCode(ctx context.Context) (deviceCodeResponse, error) {
	body, _ := json.Marshal(map[string]string{"client_id": s.ClientID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.AuthBase+"/device/code", bytes.NewReader(body))
	if err != nil {
		return deviceCodeResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.HTTP.Do(req)
	if err != nil {
		return deviceCodeResponse{}, fmt.Errorf("requesting device code: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return deviceCodeResponse{}, fmt.Errorf("device code request failed: HTTP %d: %s", res.StatusCode, b)
	}
	var code deviceCodeResponse
	if err := json.NewDecoder(res.Body).Decode(&code); err != nil {
		return deviceCodeResponse{}, fmt.Errorf("decoding device code: %w", err)
	}
	return code, nil
}

func (s DeviceStrategy) poll(ctx context.Context, deviceCode string) (deviceTokenResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"grant_type":  deviceGrantType,
		"device_code": deviceCode,
		"client_id":   s.ClientID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.AuthBase+"/device/token", bytes.NewReader(body))
	if err != nil {
		return deviceTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.HTTP.Do(req)
	if err != nil {
		return deviceTokenResponse{}, fmt.Errorf("polling for approval: %w", err)
	}
	defer res.Body.Close()

	// Both success (200) and the pending/slow_down/denied/expired errors (400)
	// carry a JSON body; decode it either way.
	var out deviceTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return deviceTokenResponse{}, fmt.Errorf("decoding poll response (HTTP %d): %w", res.StatusCode, err)
	}
	return out, nil
}

func (s DeviceStrategy) display(code deviceCodeResponse) {
	target := code.VerificationURIComplete
	if target == "" {
		target = code.VerificationURI
	}
	fmt.Fprintf(s.Out, "\nTo sign in, approve this request from the WXYC iOS app.\n")
	fmt.Fprintf(s.Out, "  code: %s\n", code.UserCode)
	fmt.Fprintf(s.Out, "  url:  %s\n", target)
	renderQR(s.Out, target)
	fmt.Fprintln(s.Out, "\nWaiting for approval…")
}
