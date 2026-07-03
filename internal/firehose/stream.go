package firehose

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/sequential"
	"github.com/gorilla/websocket"
)

const (
	minBackoff = 1 * time.Second
	maxBackoff = 30 * time.Second
)

func (c *Consumer) Run(ctx context.Context, relayURL string, logger *slog.Logger) error {
	backoff := minBackoff

	for {
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, relayURL, http.Header{})
		if err != nil {
			logger.Warn("dial failed", "err", err, "retry in", backoff)
		} else {
			backoff = minBackoff
			c.stream(ctx, conn, logger)
			conn.Close()
		}

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}

		backoff = min(backoff*2, maxBackoff)
	}

}

func (c *Consumer) stream(ctx context.Context, conn *websocket.Conn, logger *slog.Logger) {
	callbacks := &events.RepoStreamCallbacks{
		RepoCommit: func(event *atproto.SyncSubscribeRepos_Commit) error {
			return c.HandleCommit(ctx, event)
		},
	}

	scheduler := sequential.NewScheduler("asterism", callbacks.EventHandler)

	if err := events.HandleRepoStream(ctx, conn, scheduler, logger); err != nil {
		logger.Warn("stream ended", "err", err)
	}
}
