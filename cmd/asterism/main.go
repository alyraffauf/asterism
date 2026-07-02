package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/sequential"
	indigorepo "github.com/bluesky-social/indigo/repo"
	"github.com/gorilla/websocket"
	"github.com/ipfs/go-cid"
)

type Edge struct {
	Kind       string
	ActorDid   string
	Collection string
	RecordKey  string
	RecordPath string
	RecordCid  string
	Target     string
	TargetCid  string
	Rev        string
}

func main() {
	ctx := context.Background()

	conn, _, err := websocket.DefaultDialer.Dial(
		"wss://relay1.us-east.bsky.network/xrpc/com.atproto.sync.subscribeRepos",
		http.Header{},
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	callbacks := &events.RepoStreamCallbacks{
		RepoCommit: func(event *atproto.SyncSubscribeRepos_Commit) error {
			return handleRepoCommit(ctx, event)
		},
	}

	scheduler := sequential.NewScheduler("asterism", callbacks.EventHandler)
	logger := slog.Default()

	if err := events.HandleRepoStream(ctx, conn, scheduler, logger); err != nil {
		panic(err)
	}
}

func handleRepoCommit(ctx context.Context, event *atproto.SyncSubscribeRepos_Commit) error {
	fmt.Println("repo:", event.Repo, "commit:", event.Rev)

	if event.TooBig {
		fmt.Println("too big, queue for backfill later")
		return nil
	}

	repo, err := indigorepo.ReadRepoFromCar(ctx, bytes.NewReader(event.Blocks))
	if err != nil {
		return err
	}

	for _, operation := range event.Ops {
		if err := handleRepoOperation(ctx, event, repo, operation); err != nil {
			fmt.Println("could not handle operation:", err)
			continue
		}
	}

	return nil
}

func handleRepoOperation(ctx context.Context, event *atproto.SyncSubscribeRepos_Commit, repo *indigorepo.Repo, operation *atproto.SyncSubscribeRepos_RepoOp) error {
	collection, recordKey, ok := strings.Cut(operation.Path, "/")
	if !ok {
		return fmt.Errorf("bad path: %s", operation.Path)
	}

	fmt.Println("op:", operation.Action, "collection:", collection, "rkey:", recordKey)

	if operation.Action == "delete" {
		return nil
	}

	recordCid, record, err := repo.GetRecord(ctx, operation.Path)
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

	printRecord(event, operation, collection, recordKey, recordCid, record)

	return nil
}

func printRecord(
	event *atproto.SyncSubscribeRepos_Commit,
	operation *atproto.SyncSubscribeRepos_RepoOp,
	collection string,
	recordKey string,
	recordCid cid.Cid,
	record any,
) {
	switch record := record.(type) {
	case *bsky.FeedRepost:
		edge := Edge{
			Kind:       "repost",
			ActorDid:   event.Repo,
			Collection: collection,
			RecordKey:  recordKey,
			RecordPath: operation.Path,
			RecordCid:  recordCid.String(),
			Target:     record.Subject.Uri,
			TargetCid:  record.Subject.Cid,
			Rev:        event.Rev,
		}
		fmt.Printf("edge: %+v\n", edge)

	case *bsky.GraphFollow:
		edge := Edge{
			Kind:       "follow",
			ActorDid:   event.Repo,
			Collection: collection,
			RecordKey:  recordKey,
			RecordPath: operation.Path,
			RecordCid:  recordCid.String(),
			Target:     record.Subject,
			Rev:        event.Rev,
		}
		fmt.Printf("edge: %+v\n", edge)

	default:
		fmt.Printf("other record type: %T\n", record)
	}
}
