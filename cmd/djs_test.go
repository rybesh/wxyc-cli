package cmd

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// jwtWithSub builds an unsigned JWT whose payload carries sub, enough for the
// CLI's claim-reading path (it never verifies the signature).
func jwtWithSub(sub string) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + sub + `"}`))
	return "x." + payload + ".y"
}

func TestPlaylists_DefaultsToOwnDJID(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	t.Setenv("WXYC_API_URL", srv.URL)
	t.Setenv("WXYC_JWT", jwtWithSub("me123"))

	if _, err := runCLI(t, "djs", "playlists"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "dj_id=me123") {
		t.Errorf("query = %q, want dj_id=me123 from token", gotQuery)
	}
}

func TestPlaylists_TableProjection(t *testing.T) {
	body := `[{"show":42,"date":"2026-05-15T16:03:07.479Z",` +
		`"djs":[{"dj_id":"abc","dj_name":"Ryan Shaw"}],` +
		`"preview":[` +
		`{"entry_type":"show_start","artist_name":"marker"},` +
		`{"entry_type":"track","artist_name":"3BallMTY","track_title":"Beso Al Aire"}` +
		`]}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()
	t.Setenv("WXYC_API_URL", srv.URL)
	t.Setenv("WXYC_JWT", "unused")

	out, err := runCLI(t, "djs", "playlists", "abc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "2026-05-15") || !strings.Contains(out, "Ryan Shaw") {
		t.Errorf("table missing date/dj:\n%s", out)
	}
	// Only the track row appears in the preview; the marker row is dropped.
	if !strings.Contains(out, "3BallMTY – Beso Al Aire") {
		t.Errorf("preview missing track:\n%s", out)
	}
	if strings.Contains(out, "marker") {
		t.Errorf("preview should skip non-track rows:\n%s", out)
	}
}
