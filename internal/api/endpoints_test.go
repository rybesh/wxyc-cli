package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLibrarySearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("artist_name"); got != "aphex" {
			t.Errorf("artist_name = %q, want aphex", got)
		}
		w.Write([]byte(`[
		  {"id":53951,"code_letters":"AP","code_artist_number":2,"code_number":5,
		   "artist_name":"Aphex Twin","album_title":"Classics","format":"vinyl"}
		]`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	albums, _, err := c.LibrarySearch(context.Background(), map[string]string{"artist_name": "aphex"})
	if err != nil {
		t.Fatal(err)
	}
	if len(albums) != 1 {
		t.Fatalf("got %d albums, want 1", len(albums))
	}
	a := albums[0]
	if a.ArtistName != "Aphex Twin" || a.AlbumTitle != "Classics" || a.ID != 53951 {
		t.Errorf("unexpected album %+v", a)
	}
}

func TestFlowsheet_UnwrapsEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}
		w.Write([]byte(`{"entries":[
		  {"id":5284309,"entry_type":"track","artist_name":"Metropolitan Blues All Stars",
		   "album_title":"life of the party","track_title":"Five Long Years","record_label":"June Appal"},
		  {"id":5284310,"entry_type":"show_end","dj_name":"dj fozzie"}
		]}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	entries, _, err := c.Flowsheet(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].TrackTitle != "Five Long Years" || entries[0].ArtistName != "Metropolitan Blues All Stars" {
		t.Errorf("track row mismatch: %+v", entries[0])
	}
	if entries[1].EntryType != "show_end" || entries[1].DJName != "dj fozzie" {
		t.Errorf("marker row mismatch: %+v", entries[1])
	}
}
