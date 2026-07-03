package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/gorilla/websocket"

	"github.com/alyraffauf/asterism/internal/api"
	"github.com/alyraffauf/asterism/internal/backfill"
	"github.com/alyraffauf/asterism/internal/firehose"
	"github.com/alyraffauf/asterism/internal/store"
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

	linkStore, err := store.Open("asterism.db")
	if err != nil {
		panic(err)
	}

	server := &api.Server{Store: linkStore}
	go func() {
		if err := server.Run(":8081"); err != nil {
			panic(err)
		}
	}()

	wantedCollections := parseCollections(*collectionsFlag)

	var collections []string
	for collection := range wantedCollections {
		collections = append(collections, collection)
	}

	if len(collections) > 0 {
		bf := &backfill.Backfill{
			Client:    &xrpc.Client{Host: "https://relay1.us-east.bsky.network"},
			Directory: identity.DefaultDirectory(),
			Store:     linkStore,
		}
		go func() {
			if err := bf.Run(ctx, collections); err != nil {
				fmt.Println("backfill error:", err)
			}
		}()
	}

	consumer := &firehose.Consumer{
		WantedCollections: wantedCollections,
		Store:             linkStore,
	}

	if err := consumer.Run(ctx, conn, logger); err != nil {
		panic(err)
	}
}
