package firehose

import (
	"github.com/alyraffauf/asterism/internal/backfill"
	"github.com/alyraffauf/asterism/internal/store"
)

type Consumer struct {
	WantedCollections map[string]struct{}
	Store             *store.Store
	Backfill          *backfill.Backfill
}

func (c *Consumer) wants(collection string) bool {
	if len(c.WantedCollections) == 0 {
		return true
	}
	_, ok := c.WantedCollections[collection]
	return ok
}
