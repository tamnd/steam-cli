---
title: "Quick start"
description: "Fetch your first records with st."
weight: 30
---

Once `st` is on your `PATH`, fetch a store app. The argument is an appid, a store
URL, or anything `st ref id` can classify:

```bash
st app 620
```

By default you get an aligned table. Ask for JSON when you want to pipe it:

```bash
$ st app 620 -o json
[
  {
    "appid": 620,
    "name": "Portal 2",
    "url": "https://store.steampowered.com/app/620",
    "type": "game",
    "developers": ["Valve"],
    "publishers": ["Valve"],
    ...
  }
]
```

`st app` fetches the store JSON, then folds in the two structured islands the
store page carries that the JSON omits: the schema.org rating and the user tag
list.

## Shape the output

The same flags work on every command:

```bash
st app 620 --fields appid,name,url    # keep only these columns
st app 620 --template '{{.Name}}'     # just the name
st search portal -o jsonl | jq .appid # one object per line, into jq
```

`-o` takes `table`, `markdown`, `json`, `jsonl`, `csv`, `tsv`, `url`, or `raw`.
Left to `auto`, it prints a table to a terminal and JSONL into a pipe, so the same
command reads well by hand and parses cleanly downstream. See
[output formats](/reference/output/) for the full contract.

## Search the store

`search` takes a free-text query and returns matching apps, each addressable by
its appid:

```bash
st search portal                      # matching apps
st search portal -n 5 -o url          # the first five, as URLs
st search portal -o url | head -3 | sed 's#.*/app/##' | xargs -n1 st app
```

## Walk one app's graph

Everything below addresses the same appid, so an app and its data line up:

```bash
st reviews 620 -n 50                  # user reviews (cursor-paged)
st news 620                           # news and announcements
st players 620                        # the live concurrent player count
st achievements 620                   # global achievement unlock rates
st package 7877                       # a package (sub) with the apps it bundles
st featured                           # the featured store categories' apps
```

## Profiles and the market

```bash
st profile gabelogannewell            # a public community profile
st resolve gabelogannewell            # a vanity name to every SteamID form
st market "AK-47 | Redline"           # search the community market
st price 730 "AK-47 | Redline (Field-Tested)"  # one item's lowest, median, volume
```

## Resolve references offline

The `ref` commands touch no network. They classify any Steam reference, build the
URL for a kind and id, and convert a SteamID between its forms:

```bash
st ref id https://store.steampowered.com/app/620/Portal_2/   # -> (app, 620)
st ref url app 620                                            # -> the store URL
st ref steamid STEAM_1:0:11101                               # all SteamID forms
```

## Serve it instead

The same operations are available over HTTP and to agents over MCP:

```bash
st serve --addr :7777 &
curl -s 'localhost:7777/v1/app/620'   # NDJSON, one record per line
st mcp                                # MCP over stdio
```

From here, the [guides](/guides/) cover the common jobs and the
[CLI reference](/reference/cli/) lists every command and flag.
