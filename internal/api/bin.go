package api

import "context"

// Bin returns the albums in the authenticated DJ's bin.
func (c *Client) Bin(ctx context.Context) ([]BinItem, error) {
	var items []BinItem
	if err := c.get(ctx, "/djs/bin", nil, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// BinAdd saves an album to the DJ's bin. Mutating: gated behind --write.
func (c *Client) BinAdd(ctx context.Context, albumID int) error {
	return c.post(ctx, "/djs/bin", map[string]int{"album_id": albumID}, nil)
}
