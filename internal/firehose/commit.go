package firehose

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alyraffauf/asterism/internal/index"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
	indigorepo "github.com/bluesky-social/indigo/repo"
	"github.com/ipfs/go-cid"
)

func (c *Consumer) HandleCommit(ctx context.Context, event *atproto.SyncSubscribeRepos_Commit) error {
	if event.TooBig {
		go func() {
			if err := c.Backfill.Repo(ctx, event.Repo, c.WantedCollections); err != nil {
				c.Logger.Error("could not backfill too-big repo", "repo", event.Repo, "err", err)
			}
		}()
		return nil
	}

	repo, err := indigorepo.ReadRepoFromCar(ctx, bytes.NewReader(event.Blocks))
	if err != nil {
		return err
	}

	if !c.hasWantedOps(event.Ops) {
		return nil
	}

	if err := c.verifyCommit(ctx, repo); err != nil {
		c.Logger.Error("commit verification failed", "repo", event.Repo, "err", err)
		return nil
	}

	for _, operation := range event.Ops {
		if err := c.handleOperation(ctx, event, repo, operation); err != nil {
			c.Logger.Error("could not handle operation", "err", err)
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

	return index.Record(ctx, c.Store, event.Repo, collection, recordKey, recordCid.String(), event.Rev, *recordBytes)
}

func (c *Consumer) hasWantedOps(ops []*atproto.SyncSubscribeRepos_RepoOp) bool {
	for _, op := range ops {
		collection, _, ok := strings.Cut(op.Path, "/")
		if ok && c.wants(collection) {
			return true
		}
	}
	return false
}

func (c *Consumer) verifyCommit(ctx context.Context, repo *indigorepo.Repo) error {
	sc := repo.SignedCommit()

	resolveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	identity, err := c.Directory.LookupDID(resolveCtx, syntax.DID(sc.Did))
	cancel()
	if err != nil {
		return fmt.Errorf("resolve did: %w", err)
	}

	pubKey, err := identity.PublicKey()
	if err != nil {
		return fmt.Errorf("get public key: %w", err)
	}

	unsignedBytes, err := sc.Unsigned().BytesForSigning()
	if err != nil {
		return fmt.Errorf("marshal unsigned commit: %w", err)
	}

	return pubKey.HashAndVerify(unsignedBytes, sc.Sig)
}
