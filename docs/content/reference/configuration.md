---
title: "Configuration"
description: "Defaults, the data directory, and the environment variables st reads."
weight: 20
---

st needs no configuration: it runs anonymously against public data out of the box.
There is no API key and no login. The settings below let you tune the locale,
politeness, and storage.

## Defaults

| Setting | Default | Flag |
|---|---|---|
| Requests | paced and retried on 429/5xx | `--rate`, `--retries` |
| Cache freshness | 6h | `--cache-ttl`, `--no-cache`, `--refresh` |
| Storefront country | `us` | `--cc` |
| Storefront language | `english` | `--lang` |
| Market currency | `1` (USD) | `--currency` |
| Review order | `recent` | `--review-filter` |
| Review language | `all` | `--review-language` |
| Review purchase type | `all` | `--purchase-type` |

## Locale

`--cc`, `--lang`, and `--currency` set the storefront the store and market answer
from. The country code drives price and availability, the language drives the name
and description, and the currency code (an integer, 1 is USD) drives market
prices:

```bash
st app 620 --cc de --lang german          # the German storefront
st price 730 "AK-47 | Redline (Field-Tested)" --currency 3   # EUR
```

## Review filters

`reviews` takes three filters that map straight to the public review endpoint:

```bash
st reviews 620 --review-filter all --review-language english --purchase-type steam
```

## The data directory

Caches and any record store live under one data directory, chosen in this order:

1. `--data-dir`
2. `ST_DATA_DIR`
3. `$XDG_DATA_HOME/st`
4. the platform default (for example `~/.local/share/st`)

## Environment variables

st reads a small, fixed set of environment variables:

| Variable | Effect |
|---|---|
| `ST_DATA_DIR` | Where caches and any record store live |
| `ST_CONFIG_DIR` | Where a config file is looked up |
| `XDG_DATA_HOME` | Falls back to `$XDG_DATA_HOME/st` for the data directory |
| `XDG_CONFIG_HOME` | Falls back to `$XDG_CONFIG_HOME/st` for the config directory |
| `NO_COLOR` | Disables colored output, same as `--color never` |
| `COLUMNS` | Width hint for the table formatter |

Everything else is a flag. Flags win over the config file, which wins over the
built-in defaults.

## Sending records to a store

`--db` tees every emitted record into a store as a side effect of reading, so a
session fills a local database without a separate import step:

```bash
st reviews 620 -n 200 --db out.db        # SQLite file
st reviews 620 -n 200 --db 'postgres://...'
```
