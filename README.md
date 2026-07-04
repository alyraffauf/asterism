# Asterism

> **asterism** (n.) — a group of stars that forms a pattern in the night sky, but may or may not be an official constellation.
> — [Cambridge Dictionary](https://dictionary.cambridge.org/dictionary/english/asterism)

Asterism is an [AT Protocol](https://atproto.com) link index that consumes events from across the network. At its core, Asterism is a drop-in replacement for [Constellation](https://constellation.microcosm.blue/), and implements a compatible API. It is intended for app developers that want to own their own stack without rewriting their apps.

Constellation is vital community infrastructure, and many ATProto apps have been built on its back, including my own, [atbbs.xyz](https://atbbs.xyz/). It can be run on a Raspberry Pi with modest storage requirements, thanks in part to its dependency on Jetstream, which provides a nice, reasonable JSON stream for events.

Asterism, meanwhile, consumes cryptographically verifiable events directly from the Firehose, and filters them by the collection of your choice. There's no Jetstream in the middle, meaning fewer moving parts. And while Asterism has significant bandwidth requirements, the filtered index is significantly smaller and scales with your application, not with the network.

> **Early stage.** Functional but incomplete. APIs may change, backfill is rudimentary, and several features are not yet implemented. See [Roadmap](#roadmap).

## What it does

Asterism connects directly to the relay Firehose (`com.atproto.sync.subscribeRepos`), decodes each repo commit's CAR-framed CBOR blocks itself, and recursively walks each record for link references (strong refs, AT-URIs, DIDs, URLs). Links are stored keyed by target, source collection, and field path. It can optionally backfill existing repos for your configured collections on startup so the index is useful immediately.

This matters for three reasons:

**Sovereignty** — No middlemen. Asterism reads straight from the relay Firehose, and doesn't rely on secondary processors like Jetstream.

**Latency** — Fewer hops also means fresher data faster. Asterism reduces Constellation's Relay → Jetstream → Constellation to a single hop, Relay -> Asterism.

**Verifiability** — Firehose commits carry signed MST proofs; Jetstream strips them and re-serializes as plain JSON. Asterism verifies each record against its repo's signed commit instead of trusting an upstream re-encoding.

```
Relay ──► Jetstream ──► Constellation     (preprocessed events)
Relay ──► Asterism                        (raw commits, filter locally)
```

## Quick start

**Requirements:** Go 1.26+

The typical deployment indexes only the collections your app queries:

```bash
go run ./cmd/asterism/ --collections sh.tangled.graph.follow,sh.tangled.repo.issue,sh.tangled.feed.comment
```

This connects to the relay Firehose, stores links in an sqlite database at `asterism.db`, and serves the query API on `:8081`. Firehose events that are too large to include inline are fetched through the repo API automatically.

To also backfill existing repos for your configured collections on startup:

```bash
go run ./cmd/asterism/ --backfill --collections sh.tangled.graph.follow,sh.tangled.repo.issue,sh.tangled.feed.comment
```

To live-index all collections (Constellation-equivalent scope, not recommended for sovereign deployments):

```bash
go run ./cmd/asterism/
```

### Configuration

Every flag can also be set with an environment variable.

| Flag            | Environment variable   | Default                       | Description                                                                                 |
| --------------- | ---------------------- | ----------------------------- | ------------------------------------------------------------------------------------------- |
| `--collections` | `ASTERISM_COLLECTIONS` | empty                         | Comma-separated collection NSIDs to index. Empty means all collections.                     |
| `--backfill`    | `ASTERISM_BACKFILL`    | false                         | Backfill existing repos for configured collections on startup.                              |
| `--database`    | `ASTERISM_DATABASE`    | `asterism.db`                 | SQLite database path.                                                                       |
| `--listen`      | `ASTERISM_LISTEN`      | `:8081`                       | HTTP API listen address.                                                                    |
| `--relay`       | `ASTERISM_RELAY`       | `relay1.us-east.bsky.network` | Relay host. Asterism derives the Firehose websocket and relay HTTP API URLs from this host. |

For example:

```bash
ASTERISM_DATABASE=/var/lib/asterism/asterism.db \
ASTERISM_LISTEN=:8081 \
ASTERISM_COLLECTIONS=sh.tangled.graph.follow,sh.tangled.repo.issue \
ASTERISM_BACKFILL=true \
go run ./cmd/asterism/
```

### Docker

A container image is published to `ghcr.io/alyraffauf/asterism` on every push to `master` and on tagged releases. The image is built `FROM gcr.io/distroless/static-debian12` — Asterism has no C dependencies (its SQLite driver is pure Go), so the final image is a single static binary plus CA certificates, nothing else.

```bash
docker run -d \
  -p 8081:8081 \
  -v asterism-data:/data \
  -e ASTERISM_DATABASE=/data/asterism.db \
  -e ASTERISM_COLLECTIONS=sh.tangled.graph.follow,sh.tangled.repo.issue \
  -e ASTERISM_BACKFILL=true \
  ghcr.io/alyraffauf/asterism:latest
```

To build locally instead: `docker build -t asterism .`

## API

Asterism implements all five current endpoints from the [microcosm links XRPC namespace](https://constellation.microcosm.blue/) (the older `/links/*` REST endpoints are deprecated upstream in favor of these and aren't implemented here), plus an identity endpoint borrowed from [Slingshot](https://slingshot.microcosm.blue/) and the standard `com.atproto.identity.resolveHandle`:

### `GET /xrpc/blue.microcosm.links.getBacklinksCount`

Count records linking to a subject from a specific collection and field path.

```bash
curl 'http://localhost:8081/xrpc/blue.microcosm.links.getBacklinksCount\
?subject=at%3A%2F%2Fdid%3Aplc%3Aexample%2Fapp.bsky.feed.post%2F3juxgle5hpk2z\
&source=app.bsky.feed.like%3Asubject.uri'
```

Response: `{"total": 42}`

### `GET /xrpc/blue.microcosm.links.getBacklinkDids`

List distinct DIDs that have records linking to a subject. Paginated.

| Parameter | Description                                                                 |
| --------- | --------------------------------------------------------------------------- |
| `subject` | Target AT-URI, DID, or URL (required)                                       |
| `source`  | Collection and field path, e.g. `app.bsky.feed.like:subject.uri` (required) |
| `limit`   | Page size, 1–1000 (default 100)                                             |
| `cursor`  | Pagination cursor from previous response                                    |

Response: `{"total": 42, "linking_dids": ["did:plc:..."], "cursor": "..."}`

### `GET /xrpc/blue.microcosm.links.getBacklinks`

List source records linking to a subject. Paginated.

| Parameter | Description                                     |
| --------- | ----------------------------------------------- |
| `subject` | Target AT-URI, DID, or URL (required)           |
| `source`  | Collection and field path (required)            |
| `did`     | Filter to specific actor DIDs (repeatable)      |
| `limit`   | Page size, 1–1000 (default 100)                 |
| `reverse` | Return links in ascending order (default false) |
| `cursor`  | Pagination cursor from previous response        |

Response: `{"total": 42, "records": [{"did": "...", "collection": "...", "rkey": "..."}], "cursor": "..."}`

Records identify the linking record by DID, collection, and rkey. Clients must hydrate display data separately (via AppView, PDS, etc.).

### `GET /xrpc/blue.microcosm.links.getManyToMany`

Join records linking to a subject with a second field path on those same records — a one-hop join in a single query. For example, `app.bsky.graph.listitem` records have both a `list` field and a `subject` field; joining them resolves list membership directly instead of requiring a `getBacklinks` call followed by N individual record lookups.

| Parameter      | Description                                            |
| -------------- | ------------------------------------------------------ |
| `subject`      | Target AT-URI, DID, or URL (required)                  |
| `source`       | Collection and field path (required)                   |
| `pathToOther`  | Second field path on the same source record (required) |
| `linkDid`      | Filter to specific linking-record DIDs (repeatable)    |
| `otherSubject` | Filter to specific secondary link targets (repeatable) |
| `limit`        | Page size, 1–1000 (default 100)                        |
| `cursor`       | Pagination cursor from previous response               |

Response: `{"total": 42, "items": [{"linkRecord": {"did": "...", "collection": "...", "rkey": "..."}, "otherSubject": "..."}], "cursor": "..."}`

### `GET /xrpc/blue.microcosm.links.getManyToManyCounts`

Like `getManyToMany`, but grouped: counts of linking records per distinct secondary target instead of the individual records themselves. Useful when you only need aggregate counts, e.g. "how many people on each of these lists also follow me" without paginating every membership record.

| Parameter      | Description                                            |
| -------------- | ------------------------------------------------------ |
| `subject`      | Target AT-URI, DID, or URL (required)                  |
| `source`       | Collection and field path (required)                   |
| `pathToOther`  | Second field path on the same source record (required) |
| `did`          | Filter to specific linking-record DIDs (repeatable)    |
| `otherSubject` | Filter to specific secondary link targets (repeatable) |
| `limit`        | Page size, 1–1000 (default 100)                        |
| `cursor`       | Pagination cursor from previous response               |

Response: `{"counts_by_other_subject": [{"subject": "...", "total": 42, "distinct": 12}], "cursor": "..."}`

Note the DID filter parameter is `did` here, not `linkDid` like `getManyToMany` — a real inconsistency in the upstream Constellation API (their own source flags it as a known TODO), preserved here for compatibility rather than "fixed."

### `GET /xrpc/blue.microcosm.identity.resolveMiniDoc`

Resolve a handle or DID to its identity. Asterism already resolves DIDs to verify commit signatures against the repo's signing key, so this endpoint is essentially free.

| Parameter    | Description                          |
| ------------ | ------------------------------------- |
| `identifier` | Handle or DID to resolve (required)  |

Response: `{"did": "...", "handle": "...", "pds": "...", "signing_key": "..."}`

### `GET /xrpc/com.atproto.identity.resolveHandle`

Resolve a handle to a DID. This is a standard atproto lexicon (not microcosm-specific), included for compatibility with generic atproto tooling that expects any resolver/PDS to expose it.

| Parameter | Description                  |
| --------- | ----------------------------- |
| `handle`  | Handle to resolve (required) |

Response: `{"did": "..."}`

## Roadmap

**Near term**

- [x] Full Constellation API parity (`getBacklinksCount`, `getBacklinkDids`, `getBacklinks`, `getManyToMany`, `getManyToManyCounts`)
- [x] Configurable listen address, database path, relay host, and startup backfill
- [x] Account deletion handling
- [ ] Account deactivation handling
- [x] Graceful shutdown and Firehose reconnect
- [x] CI + Dockerfile
- [x] Add health endpoint
- [x] Verify commits against the repo's signing key (not just CID/hash consistency)

**Medium term**

- [ ] Robust automatic backfill with checkpoint/resume (survive restarts mid-backfill)
- [ ] Exponential backoff for getRepo requests
- [ ] Backfill progress reporting
- [ ] Prometheus metrics endpoint

**Longer term**

- [ ] Deployment guides + Docker Compose + Helm Chart
- [ ] Pluggable storage backends for larger indexes
- [ ] Horizontal read scaling

## Related projects

- [Constellation](https://constellation.microcosm.blue/) — The reference backlink index from [microcosm.blue](https://www.microcosm.blue/)
- [Spacedust](https://www.microcosm.blue/) — Real-time link stream filtered by target
- [indigo](https://github.com/bluesky-social/indigo) — Go ATProto library used for Firehose, repo, and identity handling
