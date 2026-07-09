package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderer_JSON(t *testing.T) {
	var buf bytes.Buffer
	r := Renderer{JSON: true, Out: &buf}
	data := []map[string]any{{"artist": "Aphex Twin", "album": "Classics"}}
	if err := r.Emit(data, []string{"Artist"}, [][]string{{"Aphex Twin"}}); err != nil {
		t.Fatal(err)
	}
	// Output must be valid JSON echoing the structured value, not the table.
	var out []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if out[0]["artist"] != "Aphex Twin" {
		t.Errorf("json = %v", out)
	}
}

func TestRenderer_Table(t *testing.T) {
	var buf bytes.Buffer
	r := Renderer{JSON: false, Out: &buf}
	err := r.Emit(nil,
		[]string{"ARTIST", "ALBUM"},
		[][]string{
			{"Aphex Twin", "Classics"},
			{"Boards of Canada", "Music Has the Right to Children"},
		})
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "ARTIST") || !strings.Contains(got, "Boards of Canada") {
		t.Errorf("table missing content:\n%s", got)
	}
	// Header line should precede data rows.
	if strings.Index(got, "ARTIST") > strings.Index(got, "Aphex Twin") {
		t.Errorf("header should come before rows:\n%s", got)
	}
}

func TestRenderer_TableTruncatesLongCells(t *testing.T) {
	var buf bytes.Buffer
	r := Renderer{JSON: false, Out: &buf}
	long := strings.Repeat("x", 200)
	if err := r.Emit(nil, []string{"ARTIST"}, [][]string{{long}, {"short"}}); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	// No single line should be anywhere near the untruncated 200 chars.
	for _, line := range strings.Split(strings.TrimRight(got, "\n"), "\n") {
		if len([]rune(line)) > maxCellWidth+8 { // +slack for padding/other cols
			t.Errorf("line not truncated (%d runes): %q", len([]rune(line)), line)
		}
	}
	if !strings.Contains(got, "…") {
		t.Errorf("truncated cell should carry an ellipsis:\n%s", got)
	}
	if !strings.Contains(got, "short") {
		t.Errorf("short cell should be intact:\n%s", got)
	}
}

func TestRenderer_JSONNotTruncated(t *testing.T) {
	var buf bytes.Buffer
	r := Renderer{JSON: true, Out: &buf}
	long := strings.Repeat("x", 200)
	if err := r.EmitRaw([]byte(`[{"a":"`+long+`"}]`), []string{"A"}, [][]string{{long}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), long) {
		t.Errorf("--json must not truncate values")
	}
}

func TestRenderer_EmptyTable(t *testing.T) {
	var buf bytes.Buffer
	r := Renderer{JSON: false, Out: &buf}
	if err := r.Emit(nil, []string{"ARTIST"}, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "no results") {
		t.Errorf("empty table should note no results, got %q", buf.String())
	}
}
