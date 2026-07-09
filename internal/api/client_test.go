package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Get_DecodesJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":1,"artist_name":"Aphex Twin"}]`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	var out []struct {
		ID     int    `json:"id"`
		Artist string `json:"artist_name"`
	}
	if err := c.get(context.Background(), "/library/", nil, &out); err != nil {
		t.Fatalf("get() error = %v", err)
	}
	if len(out) != 1 || out[0].Artist != "Aphex Twin" {
		t.Fatalf("decoded %+v, want one Aphex Twin row", out)
	}
}

func TestClient_Get_SendsQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	q := map[string]string{"artist_name": "boards of canada"}
	var out []any
	if err := c.get(context.Background(), "/library/", q, &out); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "artist_name=boards+of+canada" {
		t.Errorf("query = %q, want artist_name=boards+of+canada", gotQuery)
	}
}

func TestClient_Get_MapsStatusToError(t *testing.T) {
	tests := []struct {
		code int
	}{{http.StatusUnauthorized}, {http.StatusForbidden}, {http.StatusNotFound}, {http.StatusInternalServerError}}
	for _, tt := range tests {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"nope"}`, tt.code)
		}))
		c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
		var out any
		err := c.get(context.Background(), "/x", nil, &out)
		var se *StatusError
		if !errors.As(err, &se) {
			srv.Close()
			t.Fatalf("code %d: got %v, want *StatusError", tt.code, err)
		}
		if se.Code != tt.code {
			t.Errorf("StatusError.Code = %d, want %d", se.Code, tt.code)
		}
		srv.Close()
	}
}
