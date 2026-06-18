# st

A command line for Steam.

`st` is a single pure-Go binary. It reads public Steam data over plain HTTPS,
shapes it into clean records, and prints output that pipes into the rest of your
tools. There is no API key and no login: every surface it reads is public.

It reads one keyless plane across three hosts, all sharing the numeric appid as
the universal key:

- The **store** (`store.steampowered.com`) answers app details, store search,
  user reviews, package details, and the featured lists as JSON. `st app` also
  folds in the two structured islands the store page carries that the JSON omits:
  the schema.org rating and the user tag list.
- The keyless subset of the **web API** (`api.steampowered.com`) answers a game's
  news, its live concurrent player count, and its global achievement unlock
  rates.
- The **community site** (`steamcommunity.com`) serves public profiles as XML and
  the community market as JSON.

Because the appid keys the store entry, its reviews, its news, its player count,
and its achievements, every record links into one graph: a search hit points at
an app, an app points at its DLC, packages, reviews, and news, a review points at
its author's profile, and a profile points at its most-played apps. No record is
a dead leaf.

The same package is also a [resource-URI driver](#use-it-as-a-resource-uri-driver),
so a host program like [ant](https://github.com/tamnd/ant) can address Steam as
`steam://` URIs.

## Install

```bash
go install github.com/tamnd/steam-cli/cmd/st@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/steam-cli/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/st:latest --help
```

## Usage

```bash
st app 620                           # one store app in full (by appid or store URL)
st app 620 -o json                   # as JSON, ready for jq
st search portal                     # search the store for apps
st reviews 620 -n 50                 # an app's user reviews (cursor-paged)
st package 7877                      # a package (sub) with the apps it bundles
st featured                          # the featured store categories' apps
st news 620                          # an app's news and announcements
st players 620                       # the live concurrent player count
st achievements 620                  # global achievement unlock rates
st profile gabelogannewell           # a public community profile
st resolve gabelogannewell           # a vanity name to every SteamID form
st market "AK-47 | Redline"          # search the community market
st price 730 "AK-47 | Redline (Field-Tested)"   # one item's lowest, median, volume
st ref id <url>                      # resolve any Steam URL to its (kind, id)
st ref steamid STEAM_1:0:11101       # convert a SteamID between its forms
st --help                            # the whole command tree
```

Every command shares one output contract:
`-o table|markdown|json|jsonl|csv|tsv|url|raw`, `--fields` to pick columns,
`--template` for a custom line, and `-n` to limit. The default adapts to where
output goes (a color-aware table on a terminal, JSONL in a pipe), so the same
command reads well by hand and parses cleanly downstream.

The storefront locale is set with `--cc`, `--lang`, and `--currency`; the review
filters with `--review-filter`, `--review-language`, and `--purchase-type`. The
`ref` commands are offline and resolve URLs, ids, and SteamID forms with no
network at all.

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents,
with no extra code:

```bash
st serve --addr :7777    # GET /v1/app/620  returns NDJSON
st mcp                   # speak MCP over stdio
```

## Use it as a resource-URI driver

`st` registers a `steam` domain the way a program registers a database driver
with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/steam-cli/steam"
```

Then [ant](https://github.com/tamnd/ant) (or any program that links the package)
dereferences `steam://` URIs without knowing anything about Steam:

```bash
ant get steam://app/620    # fetch the record
ant cat steam://app/620    # just the detailed description
ant ls  steam://app/620    # the edges (dlc, packages, reviews, news)
ant url steam://app/620    # the addressable URL
```

## Attribution

`st` reads public, read-only data only. It does not log in, store credentials, or
solve anti-bot challenges, and it is honest about a walled surface rather than
working around it. Every record keeps its `url`, so a downstream view can link
back to the source on Steam.

## Development

```
cmd/st/        thin main: hands cli.NewApp to kit.Run
cli/           assembles the kit App from the steam domain
steam/         the library: HTTP client, keyless readers, data models, and domain.go (the driver)
pkg/steamid/   offline SteamID conversion (64-bit, [U:1:N], STEAM_X:Y:Z)
docs/          tago documentation site
```

```bash
make build      # ./bin/st
make test       # go test ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the archives,
Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a cosign
signature:

```bash
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

`st` is an independent tool and is not affiliated with Valve or Steam. Apache-2.0.
See [LICENSE](LICENSE).
