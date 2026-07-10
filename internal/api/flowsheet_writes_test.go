package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// captureServer records the method, path, and decoded body of the single
// request it serves, then replies with resp.
func captureServer(t *testing.T, resp string) (*httptest.Server, *struct {
	Method string
	Path   string
	Body   map[string]any
}) {
	t.Helper()
	got := &struct {
		Method string
		Path   string
		Body   map[string]any
	}{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.Method, got.Path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(resp))
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func newTestClient(url string) *Client {
	return &Client{BaseURL: url, HTTP: http.DefaultClient}
}

func TestFlowsheetStart_PostsJoinWithDJID(t *testing.T) {
	srv, got := captureServer(t, `{"id":77,"show_name":"Freeform","primary_dj_id":"dj-42","end_time":null}`)
	c := newTestClient(srv.URL)

	s, _, err := c.FlowsheetStart(context.Background(), StartShowRequest{DJID: "dj-42", ShowName: "Freeform"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Method != http.MethodPost || got.Path != "/flowsheet/join" {
		t.Errorf("request = %s %s, want POST /flowsheet/join", got.Method, got.Path)
	}
	if got.Body["dj_id"] != "dj-42" || got.Body["show_name"] != "Freeform" {
		t.Errorf("body = %v, want dj_id/show_name set", got.Body)
	}
	if s.EffectiveShowID() != 77 {
		t.Errorf("show id = %d, want 77", s.EffectiveShowID())
	}
}

func TestFlowsheetStart_OmitsEmptyOptionals(t *testing.T) {
	srv, got := captureServer(t, `{"id":1}`)
	c := newTestClient(srv.URL)

	if _, _, err := c.FlowsheetStart(context.Background(), StartShowRequest{DJID: "dj-42"}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"show_name", "specialty_id", "dj_name_override"} {
		if _, ok := got.Body[k]; ok {
			t.Errorf("body carried empty optional %q: %v", k, got.Body)
		}
	}
}

func TestFlowsheetStart_CohostUsesShowID(t *testing.T) {
	// Joining an active show echoes a show_djs row (show_id, no id).
	srv, _ := captureServer(t, `{"show_id":88,"dj_id":"dj-9"}`)
	c := newTestClient(srv.URL)

	s, _, err := c.FlowsheetStart(context.Background(), StartShowRequest{DJID: "dj-9"})
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != 0 || s.EffectiveShowID() != 88 {
		t.Errorf("cohost session = %+v, want EffectiveShowID 88 with ID 0", s)
	}
}

func TestFlowsheetEnd_PostsEndWithDJID(t *testing.T) {
	srv, got := captureServer(t, `{"id":77,"end_time":"2026-07-10T05:00:00Z"}`)
	c := newTestClient(srv.URL)

	if _, _, err := c.FlowsheetEnd(context.Background(), "dj-42"); err != nil {
		t.Fatal(err)
	}
	if got.Method != http.MethodPost || got.Path != "/flowsheet/end" {
		t.Errorf("request = %s %s, want POST /flowsheet/end", got.Method, got.Path)
	}
	if got.Body["dj_id"] != "dj-42" {
		t.Errorf("body = %v, want dj_id dj-42", got.Body)
	}
}

func TestFlowsheetAddTrack_PostsTrackFields(t *testing.T) {
	srv, got := captureServer(t, `{"id":501,"entry_type":"track","artist_name":"Boards","track_title":"Roygbiv"}`)
	c := newTestClient(srv.URL)

	e, _, err := c.FlowsheetAddTrack(context.Background(), FlowsheetTrack{
		TrackTitle: "Roygbiv", ArtistName: "Boards", AlbumTitle: "Music Has...", Segue: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/flowsheet" {
		t.Errorf("path = %q, want /flowsheet", got.Path)
	}
	if got.Body["track_title"] != "Roygbiv" || got.Body["artist_name"] != "Boards" || got.Body["segue"] != true {
		t.Errorf("body = %v, want track fields + segue", got.Body)
	}
	// A message key must NOT be present — this is a track, not a marker.
	if _, ok := got.Body["message"]; ok {
		t.Errorf("track body leaked a message key: %v", got.Body)
	}
	if e.ID != 501 || e.EntryType != "track" {
		t.Errorf("result = %+v, want id 501 track", e)
	}
}

func TestFlowsheetAddTrack_WithAlbumIDOmitsBlankFields(t *testing.T) {
	srv, got := captureServer(t, `{"id":1,"entry_type":"track"}`)
	c := newTestClient(srv.URL)

	albumID := 45029
	if _, _, err := c.FlowsheetAddTrack(context.Background(), FlowsheetTrack{
		TrackTitle: "T", AlbumID: &albumID,
	}); err != nil {
		t.Fatal(err)
	}
	if got.Body["album_id"] != float64(45029) {
		t.Errorf("album_id = %v, want 45029", got.Body["album_id"])
	}
	// Blank artist/album must be omitted so the server takes the backfill path.
	for _, k := range []string{"artist_name", "album_title", "record_label"} {
		if _, ok := got.Body[k]; ok {
			t.Errorf("blank %q not omitted: %v", k, got.Body)
		}
	}
}

func TestFlowsheetAddMarker_PostsMessageAndType(t *testing.T) {
	srv, got := captureServer(t, `{"id":9,"entry_type":"talkset","message":"Talkset"}`)
	c := newTestClient(srv.URL)

	e, _, err := c.FlowsheetAddMarker(context.Background(), "Talkset", "talkset")
	if err != nil {
		t.Fatal(err)
	}
	if got.Body["message"] != "Talkset" || got.Body["entry_type"] != "talkset" {
		t.Errorf("body = %v, want message+entry_type", got.Body)
	}
	if e.EntryType != "talkset" {
		t.Errorf("result entry_type = %q, want talkset", e.EntryType)
	}
}

func TestFlowsheetWrite_StatusErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Bad Request, There are no active shows"))
	}))
	defer srv.Close()
	c := newTestClient(srv.URL)

	_, _, err := c.FlowsheetAddTrack(context.Background(), FlowsheetTrack{TrackTitle: "T"})
	var se *StatusError
	if !errors.As(err, &se) || se.Code != http.StatusBadRequest {
		t.Fatalf("err = %v, want StatusError 400", err)
	}
}
