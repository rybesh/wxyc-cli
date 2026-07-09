package cmd

import (
	"strconv"

	"github.com/rybesh/wxyc-cli/internal/api"
	"github.com/spf13/cobra"
)

func newLibraryCmd(app *App) *cobra.Command {
	lib := &cobra.Command{Use: "library", Short: "Query the music library catalog"}
	lib.AddCommand(newLibrarySearchCmd(app))
	return lib
}

func newLibrarySearchCmd(app *App) *cobra.Command {
	var artist, album string
	var n int
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search the catalog by artist and/or album",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			params := map[string]string{}
			if artist != "" {
				params["artist_name"] = artist
			}
			if album != "" {
				params["album_title"] = album
			}
			if n > 0 {
				params["n"] = strconv.Itoa(n)
			}
			albums, err := app.client.LibrarySearch(cmd.Context(), params)
			if err != nil {
				return err
			}
			return app.render.Emit(albums, []string{"ID", "ARTIST", "ALBUM", "FORMAT"}, albumRows(albums))
		},
	}
	cmd.Flags().StringVar(&artist, "artist", "", "artist name")
	cmd.Flags().StringVar(&album, "album", "", "album title")
	cmd.Flags().IntVarP(&n, "limit", "n", 0, "max results")
	return cmd
}

func albumRows(albums []api.Album) [][]string {
	rows := make([][]string, 0, len(albums))
	for _, a := range albums {
		rows = append(rows, []string{strconv.Itoa(a.ID), a.ArtistName, a.AlbumTitle, a.Format})
	}
	return rows
}
