---
title: "Resource URIs"
description: "Use st as a database/sql-style driver so a host program can address Steam as steam:// URIs."
weight: 20
---

`st` is a command line, but the `steam` Go package is also a small driver that
makes Steam addressable as a resource URI. A host program registers it the way a
program registers a database driver with `database/sql`, then dereferences
`steam://` URIs without knowing anything about how Steam is fetched.

The host that does this today is [ant](https://github.com/tamnd/ant), a single
binary that puts one URI namespace over a family of site tools. The examples
below use `ant`; any program that links the package gets the same behavior.

## Mounting the driver

A host enables the driver with one blank import, exactly like
`import _ "github.com/lib/pq"`:

```go
import _ "github.com/tamnd/steam-cli/steam"
```

The package's `init` registers a domain with the scheme `steam` for the hosts
`store.steampowered.com`, `steamcommunity.com`, and `api.steampowered.com`. The
standalone `st` binary does not change.

## Addressing records

A URI is `scheme://authority/id`. The resolver types map to the commands you
already know:

| URI | What it is |
| --- | --- |
| `steam://app/<appid>` | one store app, by appid |
| `steam://package/<id>` | one store package (a sub) |
| `steam://profile/<id>` | one public community profile, by SteamID64 or vanity |

```bash
ant get steam://app/620    # the app record
ant cat steam://app/620    # just the detailed description
ant url steam://app/620    # the live https URL
ant resolve https://store.steampowered.com/app/620/Portal_2/ # a pasted link, back to its URI
```

`st ref id` is the same classifier the resolver uses, so anything it accepts on
the command line also dereferences as a URI.

## Walking the graph

`ls` lists the members of a collection, and every member is itself an addressable
URI, so a host can follow the graph and write it to disk:

```bash
ant ls     steam://app/620             # the app's reviews, news, DLC, and packages
ant export steam://app/620 --follow 1 --to ./data
```

Records carry their edges as `kit:"link"` tags. An app points at its DLC,
packages, reviews, and news; a review points at its author's profile; a profile
points at its most-played apps. `ant export --follow` and `ant graph` walk those
edges, and across tools when a link points at another site's scheme.

## Why this is the same code

The driver and the binary share one definition per operation. A resolver op
answers both `st app` on the command line and `ant get steam://app/...` through a
host, from the same handler and the same client. There is no second
implementation to keep in step.
