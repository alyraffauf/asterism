package firehose

import (
	"context"

	"github.com/bluesky-social/indigo/api/atproto"
)

func (c *Consumer) HandleAccount(ctx context.Context, event *atproto.SyncSubscribeRepos_Account) error {
	if event.Status != nil && *event.Status == "deleted" {
		return c.Store.DeleteAllLinks(ctx, event.Did)
	}

	return nil
}
