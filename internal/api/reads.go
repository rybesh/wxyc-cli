package api

import (
	"context"
	"strconv"
)

// Genres lists the genre catalog.
func (c *Client) Genres(ctx context.Context) ([]Genre, error) {
	var g []Genre
	return g, c.get(ctx, "/library/genres", nil, &g)
}

// Formats lists the media formats.
func (c *Client) Formats(ctx context.Context) ([]Format, error) {
	var f []Format
	return f, c.get(ctx, "/library/formats", nil, &f)
}

// Labels lists all record labels.
func (c *Client) Labels(ctx context.Context) ([]Label, error) {
	var l []Label
	return l, c.get(ctx, "/labels", nil, &l)
}

// LabelSearch finds labels matching q.
func (c *Client) LabelSearch(ctx context.Context, q string, limit int) ([]Label, error) {
	params := map[string]string{"q": q}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	var l []Label
	return l, c.get(ctx, "/labels/search", params, &l)
}

// Schedule returns the recurring show schedule.
func (c *Client) Schedule(ctx context.Context) ([]ScheduleEntry, error) {
	var s []ScheduleEntry
	return s, c.get(ctx, "/schedule", nil, &s)
}

// Rotation returns the current rotation as raw JSON. The shape is deep
// (nested reconciled_identity, COALESCE fallbacks), so it is passed through
// verbatim to --json; the command builds a table projection separately.
func (c *Client) Rotation(ctx context.Context) ([]byte, error) {
	return c.getRaw(ctx, "/library/rotation", nil)
}
