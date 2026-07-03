package index

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atdata"

	"github.com/alyraffauf/asterism/internal/backlink"
	"github.com/alyraffauf/asterism/internal/store"
)

func Record(ctx context.Context, s *store.Store, actorDid, collection, recordKey, recordCid, rev string, recordBytes []byte) error {
	record, err := atdata.UnmarshalCBOR(recordBytes)
	if err != nil {
		return fmt.Errorf("decode record: %w", err)
	}

	base := backlink.Link{
		ActorDid:   actorDid,
		Collection: collection,
		RecordKey:  recordKey,
		RecordCid:  recordCid,
		Rev:        rev,
	}

	links := backlink.Extract(record, base)

	return s.SaveLinks(ctx, actorDid, collection, recordKey, links)
}
