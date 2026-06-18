---
title: "st"
description: "A command line for Steam."
heroTitle: "Steam, from the command line"
heroLead: "A command line for Steam. One pure-Go binary, no API key, output that pipes into the rest of your tools, and a resource-URI driver other programs can address."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

`st` reads public Steam data over plain HTTPS, shapes it into clean records, and
gets out of your way. There is no API key and no login: every surface it reads is
public.

```bash
st app 620                # one store app in full
st app 620 -o json        # as JSON, ready for jq
st reviews 620 -n 50      # an app's user reviews
st serve --addr :7777     # the same operations over HTTP
```

There is nothing to sign up for and nothing to run alongside it. Output adapts to
where it goes: an aligned table on your terminal, JSONL the moment you pipe it
somewhere.

## One keyless plane, three hosts

The numeric appid is the universal key, so every record links into one graph.

- The **store** (`store.steampowered.com`) answers app details, store search,
  user reviews, package details, and the featured lists. `st app` also folds in
  the two structured islands the store page carries that the JSON omits: the
  schema.org rating and the user tag list.
- The keyless subset of the **web API** (`api.steampowered.com`) answers a game's
  news, its live concurrent player count, and its global achievement unlock
  rates.
- The **community site** (`steamcommunity.com`) serves public profiles as XML and
  the community market as JSON.

## Two ways to use it

- **As a command** for reading Steam by hand or in a script. Start with the
  [quick start](/getting-started/quick-start/).
- **As a resource-URI driver** so a host like
  [ant](https://github.com/tamnd/ant) can address Steam as `steam://` URIs and
  follow links across sites. See [resource URIs](/guides/resource-uris/).

Both are the same code: one operation, declared once, is a CLI command, an HTTP
route, an MCP tool, and a URI dereference.

## Where to go next

- New here? Read the [introduction](/getting-started/introduction/), then the
  [quick start](/getting-started/quick-start/).
- Installing? See [installation](/getting-started/installation/).
- Doing a specific job? The [guides](/guides/) are task-first.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
