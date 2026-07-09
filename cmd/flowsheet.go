package cmd

import (
	"github.com/rybesh/wxyc-cli/internal/api"
	"github.com/spf13/cobra"
)

func newFlowsheetCmd(app *App) *cobra.Command {
	fs := &cobra.Command{Use: "flowsheet", Short: "Read the on-air flowsheet log"}
	fs.AddCommand(newFlowsheetTailCmd(app))
	return fs
}

func newFlowsheetTailCmd(app *App) *cobra.Command {
	var n int
	cmd := &cobra.Command{
		Use:   "tail",
		Short: "Show the most recent flowsheet entries",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries, err := app.client.Flowsheet(cmd.Context(), n)
			if err != nil {
				return err
			}
			return app.render.Emit(entries, []string{"TYPE", "ARTIST", "TRACK", "ALBUM"}, entryRows(entries))
		},
	}
	cmd.Flags().IntVarP(&n, "limit", "n", 10, "number of entries")
	return cmd
}

func entryRows(entries []api.FlowsheetEntry) [][]string {
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		artist, track := e.ArtistName, e.TrackTitle
		if e.EntryType != "track" {
			// Marker rows (show_start/end, dj_join/leave) carry the DJ name.
			artist = e.DJName
		}
		rows = append(rows, []string{e.EntryType, artist, track, e.AlbumTitle})
	}
	return rows
}
