package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPasswordStrategy_EmailLogin(t *testing.T) {
	var path string
	var body map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &body)
		w.Header().Set("set-auth-token", "session-xyz")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"user":{"id":"u1"}}`)
	}))
	defer srv.Close()

	s := PasswordStrategy{AuthBase: srv.URL, HTTP: srv.Client(), Ident: "dj@example.com", Password: "hunter2"}
	tok, err := s.Login(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok != "session-xyz" {
		t.Errorf("token = %q, want session-xyz", tok)
	}
	if path != "/sign-in/email" {
		t.Errorf("path = %q, want /sign-in/email", path)
	}
	if body["email"] != "dj@example.com" || body["password"] != "hunter2" {
		t.Errorf("body = %v, want email+password", body)
	}
}

func TestPasswordStrategy_UsernameLogin(t *testing.T) {
	var path string
	var body map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &body)
		w.Header().Set("set-auth-token", "session-abc")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := PasswordStrategy{AuthBase: srv.URL, HTTP: srv.Client(), Ident: "rybesh", Password: "pw"}
	tok, err := s.Login(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok != "session-abc" {
		t.Errorf("token = %q", tok)
	}
	if path != "/sign-in/username" {
		t.Errorf("path = %q, want /sign-in/username", path)
	}
	if _, ok := body["username"]; !ok {
		t.Errorf("body = %v, want username field", body)
	}
}

func TestPasswordStrategy_BadCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Invalid email or password"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	s := PasswordStrategy{AuthBase: srv.URL, HTTP: srv.Client(), Ident: "x@y.z", Password: "wrong"}
	_, err := s.Login(context.Background())
	if err == nil {
		t.Fatal("Login() = nil error on 401, want error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q should mention the status", err)
	}
}

func TestPasswordStrategy_MissingSetAuthToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // 200 but no set-auth-token header
	}))
	defer srv.Close()

	s := PasswordStrategy{AuthBase: srv.URL, HTTP: srv.Client(), Ident: "x@y.z", Password: "pw"}
	if _, err := s.Login(context.Background()); err == nil {
		t.Fatal("Login() = nil error when set-auth-token absent, want error")
	}
}
