# Asterism

Asterism is an [AT Protocol](https://atproto.com) link index that consumes events from across the network. At its core, Asterism is a drop-in replacement for [Constellation](https://constellation.microcosm.blue/), and implements a compatible API. It is intended for app developers that want to own their own stack without rewriting their apps.

Constellation is vital community infrastructure, and many ATProto apps have been built on its back, including my own, [atbbs.xyz](https://atbbs.xyz/). It can be run on a Raspberry Pi with modest storage requirements, thanks in part to its dependency on Jetstream, which provides a nice, reasonable JSON stream for events.

Asterism, meanwhile, consumes cryptographically verifiable events directly from the Firehose, and filters them by the collection of your choice. There's no Jetstream in the middle, meaning fewer moving parts. And while Asterism has significant bandwidth requirements, the filtered index is significantly smaller and scales with your application, not with the network.

> **Early stage.** Functional but *very* incomplete. APIs may change, backfill is rudimentary, and several features are not yet implemented. See [Roadmap](#roadmap).

## What it does

Asterism connects directly to the relay Firehose (`com.atproto.sync.subscribeRepos`), decodes each repo commit's CAR-framed CBOR blocks itself, and recursively walks each record for link references (strong refs, AT-URIs, DIDs, URLs). Links are stored keyed by target, source collection, and field path. On startup it backfills existing repos for your configured collections so the index is useful immediately.

This matters for two reasons:

**Sovereignty** — One fewer dependency and one fewer hop. You're reading straight from the relay, not downstream of someone else's stream processor.

**Verifiability** — Firehose commits carry signed MST proofs; Jetstream strips them and re-serializes as plain JSON. Asterism verifies each record against its repo's signed commit instead of trusting an upstream re-encoding.

```
Relay ──► Jetstream ──► Constellation     (preprocessed events)
Relay ──► Asterism                        (raw commits, filter locally)
```

## Quick start

**Requirements:** Go 1.26+

The typical deployment indexes only the collections your app queries:

```bash
go run ./cmd/asterism/ -collections sh.tangled.graph.follow,sh.tangled.repo.issue,sh.tangled.feed.comment
```

This connects to the relay Firehose, backfills existing repos for those collections in the background, stores links in an sqlite database at `asterism.db`, and serves the query API on `:8081`.

To index all collections (Constellation-equivalent scope, not recommended for sovereign deployments):

```bash
go run ./cmd/asterism/
```

## API

Asterism implements all five current endpoints from the [microcosm links XRPC namespace](https://constellation.microcosm.blue/) (the older `/links/*` REST endpoints are deprecated upstream in favor of these and aren't implemented here):

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

| Parameter | Description |
|---|---|
| `subject` | Target AT-URI, DID, or URL (required) |
| `source` | Collection and field path, e.g. `app.bsky.feed.like:subject.uri` (required) |
| `limit` | Page size, 1–1000 (default 100) |
| `cursor` | Pagination cursor from previous response |

Response: `{"total": 42, "linking_dids": ["did:plc:..."], "cursor": "..."}`

### `GET /xrpc/blue.microcosm.links.getBacklinks`

List source records linking to a subject. Paginated.

| Parameter | Description |
|---|---|
| `subject` | Target AT-URI, DID, or URL (required) |
| `source` | Collection and field path (required) |
| `did` | Filter to specific actor DIDs (repeatable) |
| `limit` | Page size, 1–1000 (default 100) |
| `reverse` | Return links in ascending order (default false) |
| `cursor` | Pagination cursor from previous response |

Response: `{"total": 42, "records": [{"did": "...", "collection": "...", "rkey": "..."}], "cursor": "..."}`

Records identify the linking record by DID, collection, and rkey. Clients must hydrate display data separately (via AppView, PDS, etc.).

### `GET /xrpc/blue.microcosm.links.getManyToMany`

Join records linking to a subject with a second field path on those same records — a one-hop join in a single query. For example, `app.bsky.graph.listitem` records have both a `list` field and a `subject` field; joining them resolves list membership directly instead of requiring a `getBacklinks` call followed by N individual record lookups.

| Parameter | Description |
|---|---|
| `subject` | Target AT-URI, DID, or URL (required) |
| `source` | Collection and field path (required) |
| `pathToOther` | Second field path on the same source record (required) |
| `linkDid` | Filter to specific linking-record DIDs (repeatable) |
| `otherSubject` | Filter to specific secondary link targets (repeatable) |
| `limit` | Page size, 1–1000 (default 100) |
| `cursor` | Pagination cursor from previous response |

Response: `{"total": 42, "items": [{"linkRecord": {"did": "...", "collection": "...", "rkey": "..."}, "otherSubject": "..."}], "cursor": "..."}`

### `GET /xrpc/blue.microcosm.links.getManyToManyCounts`

Like `getManyToMany`, but grouped: counts of linking records per distinct secondary target instead of the individual records themselves. Useful when you only need aggregate counts, e.g. "how many people on each of these lists also follow me" without paginating every membership record.

| Parameter | Description |
|---|---|
| `subject` | Target AT-URI, DID, or URL (required) |
| `source` | Collection and field path (required) |
| `pathToOther` | Second field path on the same source record (required) |
| `did` | Filter to specific linking-record DIDs (repeatable) |
| `otherSubject` | Filter to specific secondary link targets (repeatable) |
| `limit` | Page size, 1–1000 (default 100) |
| `cursor` | Pagination cursor from previous response |

Response: `{"counts_by_other_subject": [{"subject": "...", "total": 42, "distinct": 12}], "cursor": "..."}`

Note the DID filter parameter is `did` here, not `linkDid` like `getManyToMany` — a real inconsistency in the upstream Constellation API (their own source flags it as a known TODO), preserved here for compatibility rather than "fixed."

## Roadmap

**Near term**

- [x] Full Constellation API parity (`getBacklinksCount`, `getBacklinkDids`, `getBacklinks`, `getManyToMany`, `getManyToManyCounts`)
- [ ] Configurable listen address, database path, and relay URL
- [ ] Account deletion and deactivation handling
- [x] Graceful shutdown and Firehose reconnect

**Medium term**

- [ ] Robust automatic backfill with checkpoint/resume (survive restarts mid-backfill)
- [ ] Exponential backoff for getRepo requests
- [ ] Backfill progress reporting
- [ ] Metrics and health endpoints

**Longer term**

- [ ] Deployment guides (Docker, single-binary production setup)
- [ ] Pluggable storage backends for larger indexes
- [ ] Horizontal read scaling

## Related projects

- [Constellation](https://constellation.microcosm.blue/) — The reference backlink index from [microcosm.blue](https://www.microcosm.blue/)
- [Spacedust](https://www.microcosm.blue/) — Real-time link stream filtered by target
- [indigo](https://github.com/bluesky-social/indigo) — Go ATProto library used for Firehose, repo, and identity handling
