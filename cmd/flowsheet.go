package cmd

import (
	"fmt"
	"strconv"

	"github.com/rybesh/wxyc-cli/internal/api"
	"github.com/spf13/cobra"
)

func newFlowsheetCmd(app *App) *cobra.Command {
	fs := &cobra.Command{Use: "flowsheet", Short: "Read and manage the on-air flowsheet log"}
	fs.AddCommand(
		newFlowsheetTailCmd(app),
		newFlowsheetStartCmd(app),
		newFlowsheetAddCmd(app),
		newFlowsheetMarkerCmd(app, "talkset", "Log a talkset", "Talkset", "talkset"),
		newFlowsheetMarkerCmd(app, "breakpoint", "Log a breakpoint (top of hour)", "Breakpoint", "breakpoint"),
		newFlowsheetEndCmd(app),
	)
	return fs
}

func newFlowsheetStartCmd(app *App) *cobra.Command {
	var name, as string
	var specialty int
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start your show (or join the active show as co-host)",
		Args:  cobra.NoArgs,
		// Mutating: the root PersistentPreRunE blocks this unless writes are unlocked.
		Annotations: map[string]string{annMutates: "true", annOp: "flowsheet:start"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			// dj_id must match the authenticated user; read it from the token.
			djID, err := resolveDJID(cmd, app, nil)
			if err != nil {
				return err
			}
			req := api.StartShowRequest{DJID: djID, ShowName: name, DJNameOverride: as}
			if specialty > 0 {
				req.SpecialtyID = &specialty
			}
			s, raw, err := app.client.FlowsheetStart(cmd.Context(), req)
			if err != nil {
				return err
			}
			return app.render.EmitRaw(raw, showHeaders, showRows(s, "started", "joined"))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "show name")
	cmd.Flags().StringVar(&as, "as", "", "per-show DJ display name override")
	cmd.Flags().IntVar(&specialty, "specialty", 0, "specialty show id")
	return cmd
}

func newFlowsheetAddCmd(app *App) *cobra.Command {
	var track, artist, album, label string
	var albumID, rotationID int
	var segue, request bool
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a played track to the flowsheet",
		Args:  cobra.NoArgs,
		// Mutating: the root PersistentPreRunE blocks this unless writes are unlocked.
		Annotations: map[string]string{annMutates: "true", annOp: "flowsheet:add"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if track == "" {
				return fmt.Errorf("--track is required")
			}
			// With --album-id the server backfills artist/album/label; without
			// it those fields are required, so fail fast rather than 400.
			if albumID == 0 && (artist == "" || album == "") {
				return fmt.Errorf("--artist and --album are required unless --album-id is given")
			}
			t := api.FlowsheetTrack{
				TrackTitle:  track,
				ArtistName:  artist,
				AlbumTitle:  album,
				RecordLabel: label,
				Segue:       segue,
				RequestFlag: request,
			}
			if albumID > 0 {
				t.AlbumID = &albumID
			}
			if rotationID > 0 {
				t.RotationID = &rotationID
			}
			e, raw, err := app.client.FlowsheetAddTrack(cmd.Context(), t)
			if err != nil {
				return err
			}
			return app.render.EmitRaw(raw, entryHeaders, entryResultRows(e))
		},
	}
	cmd.Flags().StringVar(&track, "track", "", "track title (required)")
	cmd.Flags().StringVar(&artist, "artist", "", "artist name")
	cmd.Flags().StringVar(&album, "album", "", "album title")
	cmd.Flags().StringVar(&label, "label", "", "record label")
	cmd.Flags().IntVar(&albumID, "album-id", 0, "library album id (backfills artist/album/label)")
	cmd.Flags().IntVar(&rotationID, "rotation-id", 0, "rotation id")
	cmd.Flags().BoolVar(&segue, "segue", false, "mark as a segue into the next track")
	cmd.Flags().BoolVar(&request, "request", false, "mark as a listener request")
	return cmd
}

// newFlowsheetMarkerCmd builds a talkset/breakpoint command that appends a
// fixed non-track entry, tagging it with the given entry_type explicitly.
func newFlowsheetMarkerCmd(app *App, use, short, message, entryType string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		// Mutating: the root PersistentPreRunE blocks this unless writes are unlocked.
		Annotations: map[string]string{annMutates: "true", annOp: "flowsheet:" + entryType},
		RunE: func(cmd *cobra.Command, _ []string) error {
			e, raw, err := app.client.FlowsheetAddMarker(cmd.Context(), message, entryType)
			if err != nil {
				return err
			}
			return app.render.EmitRaw(raw, entryHeaders, entryResultRows(e))
		},
	}
}

func newFlowsheetEndCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "end",
		Short: "End your show (or leave the active show as co-host)",
		Args:  cobra.NoArgs,
		// Mutating: the root PersistentPreRunE blocks this unless writes are unlocked.
		Annotations: map[string]string{annMutates: "true", annOp: "flowsheet:end"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			djID, err := resolveDJID(cmd, app, nil)
			if err != nil {
				return err
			}
			s, raw, err := app.client.FlowsheetEnd(cmd.Context(), djID)
			if err != nil {
				return err
			}
			return app.render.EmitRaw(raw, showHeaders, showRows(s, "ended", "left"))
		},
	}
}

var (
	showHeaders  = []string{"SHOW", "NAME", "PRIMARY_DJ", "STATUS"}
	entryHeaders = []string{"ID", "TYPE", "ARTIST", "TITLE"}
)

// showRows renders a start/end confirmation. A show row (ID set) uses the solo
// label; a co-host show_djs row (ShowID set) uses the cohost label.
func showRows(s api.ShowSession, solo, cohost string) [][]string {
	status := solo
	if s.ID == 0 {
		status = cohost
	}
	return [][]string{{strconv.Itoa(s.EffectiveShowID()), s.ShowName, s.PrimaryDJID, status}}
}

// entryResultRows renders the echoed entry. Marker rows carry no artist/track,
// so their message goes in the title column.
func entryResultRows(e api.FlowsheetResult) [][]string {
	artist, title := e.ArtistName, e.TrackTitle
	if e.EntryType != "track" {
		artist, title = "", e.Message
	}
	return [][]string{{strconv.Itoa(e.ID), e.EntryType, artist, title}}
}

func newFlowsheetTailCmd(app *App) *cobra.Command {
	var n int
	cmd := &cobra.Command{
		Use:   "tail",
		Short: "Show the most recent flowsheet entries",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries, raw, err := app.client.Flowsheet(cmd.Context(), n)
			if err != nil {
				return err
			}
			return app.render.EmitRaw(raw, []string{"TYPE", "ARTIST", "TRACK", "ALBUM"}, entryRows(entries))
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
