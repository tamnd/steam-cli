---
title: "Output formats"
description: "The output contract every command shares: formats, fields, and templates."
weight: 30
---

Every command renders through one formatter, so the same flags work everywhere.
Pick a format with `-o`, or let st choose: a table when writing to a terminal,
JSONL when piped.

## Formats

```bash
st app 620 -o table     # a rounded, color-aware grid for reading
st app 620 -o markdown  # a GitHub pipe table to paste into docs (alias: md)
st app 620 -o jsonl     # one JSON object per line, for piping
st app 620 -o json      # a single JSON array
st app 620 -o csv       # spreadsheet friendly
st app 620 -o tsv       # tab-separated
st app 620 -o url       # just the URL column
st app 620 -o raw       # the underlying bytes, unformatted
```

| Format | Best for |
|---|---|
| `table` | Reading on a terminal: a rounded border with an accented header |
| `markdown` | Pasting into a README, issue, or PR (alias `md`) |
| `jsonl` | Piping into another tool, one object at a time |
| `json` | Loading a whole result as an array |
| `csv` / `tsv` | Spreadsheets and quick column math |
| `url` | Feeding URLs into other commands |
| `raw` | The unformatted bytes (response bodies) |

## Color

On an interactive terminal the `table` and `json`/`jsonl` formats are colored: the
table draws a dim border with an accented header, and JSON keys, strings, numbers,
and literals are highlighted. Color is suppressed the moment output is not a
terminal, so a pipe always gets plain, parseable bytes. Force the choice with
`--color always|never` (or set `NO_COLOR`). `markdown`, `csv`, `tsv`, `url`, and
`raw` are never colored, so they stay safe to redirect into a file.

## Narrowing columns

Keep only the fields you want:

```bash
st app 620 --fields appid,name,url
```

`--no-header` drops the header row in `table` and `csv` output, which helps when a
downstream tool expects bare rows.

## Templating rows

For full control over each line, apply a Go text/template. Fields are the JSON
keys, capitalized:

```bash
st search portal --template '{{.AppID}} {{.Name}}'
```

## Why auto-detection helps

Because the default adapts to the destination, the same command reads well by hand
and parses cleanly in a pipe:

```bash
st search portal            # a table, because this is a terminal
st search portal | wc -l    # JSONL, because this is a pipe
```

You only reach for `-o` when you want something other than that default.
