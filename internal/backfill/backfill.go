package backfill

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/alyraffauf/asterism/internal/index"
	"github.com/alyraffauf/asterism/internal/store"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/ipfs/go-cid"

	indigorepo "github.com/bluesky-social/indigo/repo"
)

const hostPacingDelay = 2 * time.Second

type Backfill struct {
	Client    *xrpc.Client
	Directory identity.Directory
	Store     *store.Store
	Logger    *slog.Logger

	lastRequest map[string]time.Time
}

// don't DDoS the PDS
func (b *Backfill) waitForHost(ctx context.Context, host string) error {
	if b.lastRequest == nil {
		b.lastRequest = make(map[string]time.Time)
	}

	if last, ok := b.lastRequest[host]; ok {
		if wait := hostPacingDelay - time.Since(last); wait > 0 {
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	b.lastRequest[host] = time.Now()
	return nil
}

func (b *Backfill) Run(ctx context.Context, collections []string) error {
	dids := make(map[string]map[string]struct{})

	for _, collection := range collections {
		if err := b.listRepos(ctx, collection, dids); err != nil {
			return fmt.Errorf("list repos for %s: %w", collection, err)
		}
	}

	for did, wantedCollections := range dids {
		if err := b.Repo(ctx, did, wantedCollections); err != nil {
			b.Logger.Error("could not backfill repo", "did", did, "err", err)
			continue
		}
	}

	return nil
}

func (b *Backfill) listRepos(ctx context.Context, collection string, dids map[string]map[string]struct{}) error {
	cursor := ""
	for {
		page, err := atproto.SyncListReposByCollection(ctx, b.Client, collection, cursor, 1000)
		if err != nil {
			return fmt.Errorf("list repos: %w", err)
		}

		for _, repo := range page.Repos {
			if dids[repo.Did] == nil {
				dids[repo.Did] = make(map[string]struct{})
			}
			dids[repo.Did][collection] = struct{}{}
		}

		if page.Cursor == nil || *page.Cursor == "" {
			return nil
		}
		cursor = *page.Cursor
	}
}

func (b *Backfill) Repo(ctx context.Context, did string, wantedCollections map[string]struct{}) error {
	resolveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	identity, err := b.Directory.LookupDID(resolveCtx, syntax.DID(did))
	cancel()
	if err != nil {
		return fmt.Errorf("resolve did: %w", err)
	}

	pdsClient := &xrpc.Client{
		Host:   identity.PDSEndpoint(),
		Client: &http.Client{Timeout: 5 * time.Minute},
	}

	if err := b.waitForHost(ctx, pdsClient.Host); err != nil {
		return fmt.Errorf("wait for host: %w", err)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	carBytes, err := atproto.SyncGetRepo(fetchCtx, pdsClient, did, "")
	if err != nil {
		return fmt.Errorf("get repo: %w", err)
	}

	repo, err := indigorepo.ReadRepoFromCar(ctx, bytes.NewReader(carBytes))
	if err != nil {
		return fmt.Errorf("read repo car: %w", err)
	}

	return repo.ForEach(ctx, "", func(k string, v cid.Cid) error {
		collection, recordKey, ok := strings.Cut(k, "/")
		if !ok {
			return fmt.Errorf("bad path: %s", k)
		}

		if len(wantedCollections) > 0 {
			if _, wanted := wantedCollections[collection]; !wanted {
				return nil
			}
		}

		recordCid, recordBytes, err := repo.GetRecordBytes(ctx, k)
		if err != nil {
			return fmt.Errorf("read record: %w", err)
		}

		return index.Record(ctx, b.Store, did, collection, recordKey, recordCid.String(), "", *recordBytes)
	})
}
