---
title: "CLI"
description: "Every command and subcommand, with the flags that matter."
weight: 10
---

```
st <command> [arguments] [flags]
```

Run `st <command> --help` for the full flag list on any command. This page is the
map.

## Store commands

| Command | What it does |
|---|---|
| `app <ref>` | Show one store app in full (details, then the store-page island) |
| `search <query>` | Search the store for apps |
| `browse` | Page through the whole store catalog (the discovery seed) |
| `crawl <ref>` | Walk the public graph breadth-first from a seed app, package, or profile |
| `reviews <ref>` | List an app's user reviews (cursor-paged) |
| `package <id>` | Show one store package (a sub) with the apps it bundles |
| `featured` | List the featured store categories' apps |
| `top-sellers` | List the current top-selling apps |
| `new-releases` | List the new-release apps |
| `specials` | List the apps currently on sale |
| `coming-soon` | List the upcoming apps |
| `news <ref>` | List an app's news and announcements |
| `players <ref>` | Show an app's live concurrent player count |
| `achievements <ref>` | List an app's global achievement unlock rates |

A `<ref>` is an appid, a store URL, or anything `st ref id` can classify.

### Discovery: browse and crawl

`browse` reads the same paginated catalog the store page calls as it scrolls. It
reports a total of around 260,000 apps and returns them in pages, so it is the
seed a walk or a bulk export starts from. Each row is a full app reference, so a
hit pipes straight into `st app`, `st reviews`, or `st crawl`.

| Flag | Meaning |
|---|---|
| `--query` | A free-text term, omitted for the whole catalog |
| `--sort` | Sort order: `Released_DESC`, `Reviews_DESC`, `Price_ASC`, `Name_ASC` |
| `--maxprice` | Price ceiling: `free`, `5`, `10`, and so on |
| `--start` | The first result offset, for resuming a walk |
| `-n, --limit` | How many apps to return in total |

`crawl` follows the typed edges the records already carry. From an app it reaches
the DLC, demos, base game, and packages; from a package it reaches the apps it
bundles; from a profile it reaches the most-played apps. It emits one node per
visited record, never revisits a node, and is best-effort past the seed, so an
unreachable node is skipped rather than fatal. Each node carries its `kind:id`
edges, so the output is a graph ready to load elsewhere.

| Flag | Meaning |
|---|---|
| `--depth` | How far from the seed to walk (default `2`, `0` visits only the seed) |
| `-n, --limit` | The total number of nodes to emit |

## Player commands

| Command | What it does |
|---|---|
| `profile <ref>` | Show one public community profile |
| `resolve <ref>` | Resolve a vanity name into every SteamID form |

A `<ref>` here is a SteamID64, a SteamID in `[U:1:N]` or `STEAM_X:Y:Z` form, a
vanity name, or a profile URL.

## Market commands

| Command | What it does |
|---|---|
| `market <query>` | Search the community market for items |
| `price <appid> <name>` | Show the lowest, median, and volume for one market item |

## Reference commands (offline)

These touch no network.

| Command | What it does |
|---|---|
| `ref id <ref>` | Classify a reference into its `(kind, id)` |
| `ref url <kind> <id>` | Build the addressable URL for a `(kind, id)` |
| `ref steamid <ref>` | Convert a SteamID between its 64-bit, `[U:1:N]`, and `STEAM_X:Y:Z` forms |

## Service commands

| Command | What it does |
|---|---|
| `serve [--addr]` | Serve the operations over HTTP as NDJSON |
| `mcp` | Run as an MCP server over stdio |
| `version` | Print the version and exit |

## Global flags

These are shared by every operation, so they work the same on every command.

| Flag | Meaning |
|---|---|
| `-o, --output` | Output format: `auto`, `table`, `markdown`, `json`, `jsonl`, `csv`, `tsv`, `url`, `raw` |
| `--fields` | Comma-separated columns to keep |
| `--template` | Go text/template applied per record |
| `--no-header` | Omit the header row in `table` and `csv` |
| `-n, --limit` | Stop after N records (0 means no limit) |
| `--rate` | Minimum delay between requests |
| `--retries` | Retry attempts on rate limit or 5xx |
| `--timeout` | Per-request timeout |
| `--cache-ttl` | How long a cached response stays fresh |
| `--no-cache` | Bypass on-disk caches |
| `--refresh` | Fetch fresh copies and rewrite the cache, ignoring any hit |
| `--data-dir` | Override the data directory |
| `--db` | Tee every record into a store (e.g. `out.db`, `postgres://...`) |
| `-v, --verbose` | Increase verbosity (repeatable) |
| `-q, --quiet` | Suppress progress output |
| `--color` | `auto`, `always`, or `never` |

## Steam-specific flags

These set the storefront locale and the review and market filters.

| Flag | Meaning | Default |
|---|---|---|
| `--cc` | Storefront country code, sets price currency and availability | `us` |
| `--lang` | Storefront language, sets description and name language | `english` |
| `--currency` | Market currency code (1 is USD) | `1` |
| `--review-filter` | Review order: `recent`, `updated`, or `all` | `recent` |
| `--review-language` | Review language: `all`, `english`, and so on | `all` |
| `--purchase-type` | Review purchase type: `all`, `steam`, or `non_steam_purchase` | `all` |
| `--user-agent` | User-Agent sent with each request | a desktop Chrome string |

See [output formats](/reference/output/) for what `-o`, `--fields`, and
`--template` produce, and [configuration](/reference/configuration/) for
environment variables and defaults.
