package firehose

import (
	"context"
	"log/slog"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/sequential"
	"github.com/gorilla/websocket"
)

func (c *Consumer) Run(ctx context.Context, conn *websocket.Conn, logger *slog.Logger) error {
	callbacks := &events.RepoStreamCallbacks{
		RepoCommit: func(event *atproto.SyncSubscribeRepos_Commit) error {
			return c.HandleCommit(ctx, event)
		},
	}

	scheduler := sequential.NewScheduler("asterism", callbacks.EventHandler)

	return events.HandleRepoStream(ctx, conn, scheduler, logger)
}
