package firehose

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/asterism/internal/backlink"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/atdata"
	indigorepo "github.com/bluesky-social/indigo/repo"
	"github.com/ipfs/go-cid"
)

func (c *Consumer) HandleCommit(ctx context.Context, event *atproto.SyncSubscribeRepos_Commit) error {
	// fmt.Println("repo:", event.Repo, "commit:", event.Rev)

	if event.TooBig {
		fmt.Println("too big, queue for backfill later")
		return nil
	}

	repo, err := indigorepo.ReadRepoFromCar(ctx, bytes.NewReader(event.Blocks))
	if err != nil {
		return err
	}

	for _, operation := range event.Ops {
		if err := c.handleOperation(ctx, event, repo, operation); err != nil {
			fmt.Println("could not handle operation:", err)
			continue
		}
	}

	return nil
}

func (c *Consumer) handleOperation(ctx context.Context, event *atproto.SyncSubscribeRepos_Commit, repo *indigorepo.Repo, operation *atproto.SyncSubscribeRepos_RepoOp) error {
	collection, recordKey, ok := strings.Cut(operation.Path, "/")

	if !ok {
		return fmt.Errorf("bad path: %s", operation.Path)
	}

	if !c.wants(collection) {
		return nil
	}

	fmt.Println("op:", operation.Action, "collection:", collection, "rkey:", recordKey)

	if operation.Action == "delete" {
		return c.Store.DeleteLinks(ctx, event.Repo, collection, recordKey)
	}

	recordCid, recordBytes, err := repo.GetRecordBytes(ctx, operation.Path)
	if err != nil {
		return fmt.Errorf("read record: %w", err)
	}

	if operation.Cid == nil {
		return fmt.Errorf("missing operation cid: %s", operation.Path)
	}

	operationCid := cid.Cid(*operation.Cid)
	if !recordCid.Equals(operationCid) {
		return fmt.Errorf("cid mismatch: %s operation=%s record=%s", operation.Path, operationCid, recordCid)
	}

	record, err := atdata.UnmarshalCBOR(*recordBytes)
	if err != nil {
		return fmt.Errorf("decode record: %w", err)
	}

	base := backlink.Link{
		ActorDid:   event.Repo,
		Collection: collection,
		RecordKey:  recordKey,
		RecordCid:  recordCid.String(),
		Rev:        event.Rev,
	}

	links := backlink.Extract(record, base)

	return c.Store.SaveLinks(ctx, event.Repo, collection, recordKey, links)
}
