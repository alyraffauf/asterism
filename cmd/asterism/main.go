package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/alyraffauf/asterism/internal/firehose"
)

const relayURL = "wss://relay1.us-east.bsky.network/xrpc/com.atproto.sync.subscribeRepos"

func parseCollections(raw string) map[string]struct{} {
	if raw == "" {
		return nil
	}

	wanted := make(map[string]struct{})
	for collection := range strings.SplitSeq(raw, ",") {
		collection = strings.TrimSpace(collection)
		if collection == "" {
			continue
		}
		wanted[collection] = struct{}{}

	}
	return wanted
}

func main() {
	collectionsFlag := flag.String("collections", "", "comma-separated list of collection NSIDs to filter on (empty means all)")
	flag.Parse()

	ctx := context.Background()
	logger := slog.Default()

	conn, _, err := websocket.DefaultDialer.Dial(relayURL, http.Header{})
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	consumer := &firehose.Consumer{
		WantedCollections: parseCollections(*collectionsFlag),
	}

	if err := consumer.Run(ctx, conn, logger); err != nil {
		panic(err)
	}
}
