package api

import "context"

// Bin returns the albums in the authenticated DJ's bin, plus the raw JSON for
// --json passthrough.
func (c *Client) Bin(ctx context.Context) ([]BinItem, []byte, error) {
	var items []BinItem
	raw, err := c.getInto(ctx, "/djs/bin", nil, &items)
	return items, raw, err
}

// BinAdd saves an album to the DJ's bin. Mutating: gated behind --write.
func (c *Client) BinAdd(ctx context.Context, albumID int) error {
	return c.post(ctx, "/djs/bin", map[string]int{"album_id": albumID}, nil)
}
