package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderer_EmitRaw_JSONPassesThroughAllFields(t *testing.T) {
	var buf bytes.Buffer
	r := Renderer{JSON: true, Out: &buf}
	raw := []byte(`[{"id":1,"artist":"A","nested":{"deep":true},"extra":"kept"}]`)
	if err := r.EmitRaw(raw, []string{"ID"}, [][]string{{"1"}}); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	// The full server payload survives, including fields the table never shows.
	for _, want := range []string{`"nested"`, `"deep"`, `"extra"`, `"kept"`} {
		if !strings.Contains(got, want) {
			t.Errorf("raw JSON output dropped %s:\n%s", want, got)
		}
	}
}

func TestRenderer_EmitRaw_TableUsesProjection(t *testing.T) {
	var buf bytes.Buffer
	r := Renderer{JSON: false, Out: &buf}
	raw := []byte(`[{"id":1,"artist":"A"}]`)
	if err := r.EmitRaw(raw, []string{"ARTIST"}, [][]string{{"Aphex Twin"}}); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Contains(got, "{") {
		t.Errorf("table mode should not emit JSON:\n%s", got)
	}
	if !strings.Contains(got, "Aphex Twin") {
		t.Errorf("table missing projected row:\n%s", got)
	}
}
