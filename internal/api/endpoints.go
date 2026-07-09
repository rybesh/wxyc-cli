package api

import (
	"context"
	"strconv"
)

// LibrarySearch queries the catalog. Accepted params mirror the backend, e.g.
// artist_name, album_title, n (page size).
func (c *Client) LibrarySearch(ctx context.Context, params map[string]string) ([]Album, error) {
	var albums []Album
	if err := c.get(ctx, "/library/", params, &albums); err != nil {
		return nil, err
	}
	return albums, nil
}

// Flowsheet returns the most recent on-air log entries, newest last.
func (c *Client) Flowsheet(ctx context.Context, limit int) ([]FlowsheetEntry, error) {
	q := map[string]string{}
	if limit > 0 {
		q["limit"] = strconv.Itoa(limit)
	}
	var resp flowsheetResponse
	if err := c.get(ctx, "/flowsheet", q, &resp); err != nil {
		return nil, err
	}
	return resp.Entries, nil
}
