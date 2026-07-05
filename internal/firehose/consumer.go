package firehose

import (
	"log/slog"

	"github.com/alyraffauf/asterism/internal/backfill"
	"github.com/alyraffauf/asterism/internal/store"
	"github.com/bluesky-social/indigo/atproto/identity"
)

type Consumer struct {
	WantedCollections map[string]struct{}
	Store             *store.Store
	Directory         identity.Directory
	Backfill          *backfill.Backfill
	Logger            *slog.Logger
}

func (c *Consumer) wants(collection string) bool {
	if len(c.WantedCollections) == 0 {
		return true
	}
	_, ok := c.WantedCollections[collection]
	return ok
}
