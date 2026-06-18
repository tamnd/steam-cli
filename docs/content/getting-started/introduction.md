---
title: "Introduction"
description: "What st is and how it is put together."
weight: 10
---

A command line for Steam.

st is a single binary. It speaks to Steam over plain HTTPS, shapes the responses
into clean records, and gets out of your way. There is no API key and no login:
every surface it reads is public.

## One keyless plane, three hosts

st reads one keyless plane spread across three hosts, all keyed by the numeric
appid:

- The **store** (`store.steampowered.com`) answers app details, store search, user
  reviews, package details, and the featured lists as JSON. `st app` also folds in
  the two structured islands the store page carries that the JSON omits: the
  schema.org rating and the user tag list.
- The keyless subset of the **web API** (`api.steampowered.com`) answers a game's
  news, its live concurrent player count, and its global achievement unlock rates.
- The **community site** (`steamcommunity.com`) serves public profiles as XML and
  the community market as JSON.

Because the appid keys the store entry, its reviews, its news, its player count,
and its achievements, every record links into one graph. A search hit points at
an app, an app points at its DLC, packages, reviews, and news, a review points at
its author's profile, and a profile points at its most-played apps.

## How it is built

- A **library package** (`steam`) holds the HTTP client and the typed data models.
  It paces requests, sets an honest User-Agent, and retries the transient failures
  any public site throws under load.
- A **domain** (`steam/domain.go`) declares each operation once on the
  [any-cli/kit](https://github.com/tamnd/any-cli) framework. That single
  declaration becomes a CLI command, an HTTP route, an MCP tool, and a
  resource-URI dereference. It is the one place you add to the tool.
- A thin **`cmd/st`** hands the assembled app to `kit.Run`, which builds the
  command tree and the serve and mcp surfaces.

## One operation, four surfaces

Because an operation is surface-neutral, the same `app` you run on the command
line is also a route and a tool:

```bash
st app 620                      # the command
st serve --addr :7777           # GET /v1/app/620
st mcp                          # the app tool, over stdio
ant get steam://app/620         # the URI dereference (via a host)
```

You write the fetch and the record shape; the surfaces come for free.

## Scope

st is a read-only client over data Steam already serves publicly. It does not log
in, store credentials, or solve anti-bot challenges, and it is honest about a
walled surface rather than working around it. That narrow scope keeps it a single
small binary with no database, no daemon, and no setup.

Next: [install it](/getting-started/installation/), then take the
[quick start](/getting-started/quick-start/).
