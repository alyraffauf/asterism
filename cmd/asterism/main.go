package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/xrpc"

	"github.com/alyraffauf/asterism/internal/api"
	"github.com/alyraffauf/asterism/internal/backfill"
	"github.com/alyraffauf/asterism/internal/firehose"
	"github.com/alyraffauf/asterism/internal/store"
)

const (
	subscribeReposPath = "/xrpc/com.atproto.sync.subscribeRepos"
)

type CLI struct {
	Collections string `help:"Comma-separated list of collection NSIDs to filter on. Empty means all." env:"ASTERISM_COLLECTIONS"`
	Backfill    bool   `help:"Backfill existing repos on startup." env:"ASTERISM_BACKFILL"`
	Database    string `help:"SQLite database path." env:"ASTERISM_DATABASE" default:"asterism.db"`
	Listen      string `help:"HTTP listen address." env:"ASTERISM_LISTEN" default:":8081"`
	Relay       string `help:"Relay host." env:"ASTERISM_RELAY" default:"relay1.us-east.bsky.network"`
}

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

func relayHTTPHost(relayHost string) string {
	return "https://" + relayHost
}

func subscribeReposURL(relayHost string) string {
	return "wss://" + relayHost + subscribeReposPath
}

func main() {
	var cli CLI
	kong.Parse(&cli,
		kong.Name("asterism"),
		kong.Description("AT Protocol link index."),
	)

	ctx := context.Background()
	logger := slog.Default()

	linkStore, err := store.Open(cli.Database)
	if err != nil {
		panic(err)
	}

	directory := identity.NewCacheDirectory(
		&identity.BaseDirectory{
			PLCURL: identity.DefaultPLCURL,
			HTTPClient: http.Client{
				Timeout: 3 * time.Second,
				Transport: &http.Transport{
					Proxy:               http.ProxyFromEnvironment,
					IdleConnTimeout:     1000 * time.Millisecond,
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 100,
				},
			},
			Resolver: net.Resolver{
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{Timeout: 3 * time.Second}
					return d.DialContext(ctx, network, address)
				},
			},
			TryAuthoritativeDNS:   true,
			SkipDNSDomainSuffixes: []string{".bsky.social", ".blacksky.app"},
			UserAgent:             "asterism/dev",
		},
		250_000, 24*time.Hour, 2*time.Minute, 5*time.Minute,
	)

	server := &api.Server{Store: linkStore, Directory: directory, Logger: logger}
	go func() {
		if err := server.Run(cli.Listen); err != nil {
			panic(err)
		}
	}()

	wantedCollections := parseCollections(cli.Collections)

	var collections []string
	for collection := range wantedCollections {
		collections = append(collections, collection)
	}

	relayURL := subscribeReposURL(cli.Relay)

	bf := &backfill.Backfill{
		Client:    &xrpc.Client{Host: relayHTTPHost(cli.Relay)},
		Directory: directory,
		Store:     linkStore,
		Logger:    logger,
	}

	if cli.Backfill {
		if len(collections) > 0 {
			go func() {
				if err := bf.Run(ctx, collections); err != nil {
					logger.Error("backfill error", "err", err)
				}
			}()
		}
	}

	consumer := &firehose.Consumer{
		WantedCollections: wantedCollections,
		Store:             linkStore,
		Directory:         directory,
		Backfill:          bf,
		Logger:            logger,
	}

	if err := consumer.Run(ctx, relayURL); err != nil {
		panic(err)
	}

}
