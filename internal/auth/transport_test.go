package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestTransport_InjectsBearer(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := &Transport{
		Token:   func(context.Context) (string, error) { return "jwt-1", nil },
		Refresh: func(context.Context) (string, error) { return "jwt-1", nil },
	}
	client := &http.Client{Transport: tr}
	res, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if seen != "Bearer jwt-1" {
		t.Errorf("Authorization = %q, want %q", seen, "Bearer jwt-1")
	}
}

func TestTransport_RefreshesOn401AndRetries(t *testing.T) {
	var refreshes int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only the refreshed token is accepted.
		if r.Header.Get("Authorization") == "Bearer jwt-new" {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, "ok")
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tr := &Transport{
		Token: func(context.Context) (string, error) { return "jwt-stale", nil },
		Refresh: func(context.Context) (string, error) {
			atomic.AddInt32(&refreshes, 1)
			return "jwt-new", nil
		},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("final status = %d, want 200", res.StatusCode)
	}
	if refreshes != 1 {
		t.Errorf("refreshes = %d, want 1", refreshes)
	}
}

func TestTransport_GivesUpAfterOneRetry(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusUnauthorized) // always reject
	}))
	defer srv.Close()

	tr := &Transport{
		Token:   func(context.Context) (string, error) { return "a", nil },
		Refresh: func(context.Context) (string, error) { return "b", nil },
	}
	client := &http.Client{Transport: tr}
	res, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", res.StatusCode)
	}
	if hits != 2 { // original + one retry, then give up
		t.Errorf("upstream hits = %d, want 2", hits)
	}
}

// Body must be replayable on retry.
func TestTransport_ReplaysBodyOnRetry(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if r.Header.Get("Authorization") == "Bearer new" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tr := &Transport{
		Token:   func(context.Context) (string, error) { return "old", nil },
		Refresh: func(context.Context) (string, error) { return "new", nil },
	}
	client := &http.Client{Transport: tr}
	res, err := client.Post(srv.URL, "text/plain", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if len(bodies) != 2 || bodies[0] != "payload" || bodies[1] != "payload" {
		t.Errorf("bodies = %v, want both %q", bodies, "payload")
	}
}
