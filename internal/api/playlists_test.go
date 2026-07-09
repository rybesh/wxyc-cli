package api

import (
	"context"
	"strings"
	"testing"
)

func TestPlaylists(t *testing.T) {
	// dj_name on a track preview row is null; show_start carries the dj_name and
	// an unmodeled field (metadata_status) that must still survive in raw.
	body := `[{"show":1947075,"show_name":"","date":"2026-05-15T16:03:07.479Z",` +
		`"djs":[{"dj_id":"abc","dj_name":"Ryan Shaw"}],"specialty_show":"",` +
		`"preview":[` +
		`{"id":1,"entry_type":"show_start","artist_name":"El Vaquero","dj_name":"El Vaquero","metadata_status":"pending"},` +
		`{"id":2,"entry_type":"track","artist_name":"3BallMTY","track_title":"Beso Al Aire","record_label":"Latin Power"}` +
		`]}]`

	var q string
	srv := serve(t, "/djs/playlists", body, &q)
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	p, raw, err := c.Playlists(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "dj_id=abc") {
		t.Errorf("query = %q, want dj_id=abc", q)
	}
	if len(p) != 1 || p[0].Show != 1947075 {
		t.Fatalf("decoded %+v", p)
	}
	if len(p[0].DJs) != 1 || p[0].DJs[0].DJName != "Ryan Shaw" {
		t.Errorf("djs = %+v", p[0].DJs)
	}
	if len(p[0].Preview) != 2 || p[0].Preview[1].TrackTitle != "Beso Al Aire" {
		t.Errorf("preview = %+v", p[0].Preview)
	}
	// Fidelity guarantee: unmodeled preview field survives for --json.
	if !strings.Contains(string(raw), "metadata_status") {
		t.Errorf("raw dropped unmodeled field: %s", raw)
	}
}
