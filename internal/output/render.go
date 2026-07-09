// Package output renders command results either as indented JSON (for agents
// and scripts, selected with --json) or an aligned text table (for humans).
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// Renderer emits a result in the configured format. Commands supply both a
// structured value (used for JSON) and a headers/rows projection (used for the
// table), so each view can be shaped independently.
type Renderer struct {
	JSON bool
	Out  io.Writer
}

// Emit writes the result. In JSON mode it encodes jsonValue; otherwise it draws
// a table from headers and rows.
func (r Renderer) Emit(jsonValue any, headers []string, rows [][]string) error {
	if r.JSON {
		enc := json.NewEncoder(r.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(jsonValue)
	}
	return r.table(headers, rows)
}

func (r Renderer) table(headers []string, rows [][]string) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(r.Out, "no results")
		return err
	}
	tw := tabwriter.NewWriter(r.Out, 0, 2, 2, ' ', 0)
	writeRow(tw, headers)
	for _, row := range rows {
		writeRow(tw, row)
	}
	return tw.Flush()
}

func writeRow(w io.Writer, cells []string) {
	for i, c := range cells {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, c)
	}
	fmt.Fprintln(w)
}
