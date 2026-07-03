package firehose

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/api/atproto"
)

func (c *Consumer) HandleAccount(ctx context.Context, event *atproto.SyncSubscribeRepos_Account) error {
	if err := c.Store.SaveCursor(ctx, event.Seq); err != nil {
		fmt.Println("could not save cursor:", err)
	}

	if event.Status != nil && *event.Status == "deleted" {
		return c.Store.DeleteAllLinks(ctx, event.Did)
	}

	return nil
}
