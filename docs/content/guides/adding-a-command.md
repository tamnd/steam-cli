---
title: "Add a command"
description: "Model a new Steam record and expose it as a command, a route, and a tool at once."
weight: 10
---

A new surface is two pieces of work: model the record in the `steam` library, then
declare its operation in `steam/domain.go`. Every surface updates itself from that
one declaration.

## 1. Model the record

In the `steam` package, add a struct for the thing you are fetching and a client
method that returns it. The `kit` struct tags decide how a host addresses the
record:

```go
type Bundle struct {
    ID    string   `json:"id"    kit:"id"`                    // the URI id
    Name  string   `json:"name"`
    Body  string   `json:"body"  kit:"body"`                  // what cat and Markdown print
    Apps  []int    `json:"apps"  kit:"link,kind=steam/app"`   // edges to other records
    URL   string   `json:"url"`
}

func (c *Client) GetBundle(ctx context.Context, id string) (*Bundle, error) {
    body, err := c.get(ctx, c.cfg.StoreURL+"/bundle/"+id)
    if err != nil {
        return nil, err
    }
    // decode body into a Bundle ...
    return b, nil
}
```

- `kit:"id"` marks the field that becomes the URI id.
- `kit:"body"` marks the prose that `cat` and the Markdown export render.
- `kit:"link,kind=<scheme>/<type>"` marks an outbound edge. It can point at
  another Steam type or at another site entirely, which is what lets a host walk
  the graph across tools.

Return the package's own error sentinels (`ErrNotFound`, `ErrRateLimited`,
`ErrBlocked`, `ErrUsage`, `ErrNetwork`) so the next step can map them.

## 2. Declare the operation

In `steam/domain.go`, add an input struct and a handler, then register it in
`Register`:

```go
type bundleIn struct {
    Ref    string  `kit:"arg"`
    Client *Client `kit:"inject"`
}

func getBundle(ctx context.Context, in bundleIn, emit func(*Bundle) error) error {
    b, err := in.Client.GetBundle(ctx, in.Ref)
    if err != nil {
        return mapErr(err)
    }
    return emit(b)
}

// inside Register(app):
kit.Handle(app, kit.OpMeta{
    Name: "bundle", Group: "store", Single: true,
    Summary: "Show one store bundle with the apps it bundles",
    URIType: "bundle", Resolver: true,
    Args: []kit.Arg{{Name: "ref", Help: "a bundle id or URL"}},
}, getBundle)
```

That is the whole change. `kit.Handle` reflects the input for flags and the output
for the record shape, so the operation immediately becomes:

```bash
st bundle <id>                       # the command, under STORE COMMANDS
curl 'localhost:7777/v1/bundle/<id>' # the route, under serve
ant get steam://bundle/<id>          # the URI dereference, via a host
```

The `Group` puts the command under a heading in `st --help`; the existing groups
are `store`, `player`, `market`, and `ref`.

## Resolver ops and list ops

Two flags shape how a host treats an operation:

- **`Single: true`** with **`Resolver: true`** marks the canonical one-record
  fetch for a `URIType`. It answers `ant get`. `app`, `package`, and `profile`
  are the resolvers st ships.
- **`List: true`** marks a member-lister for a parent resource. It answers
  `ant ls`. A list op emits records that are themselves addressable, so every
  member is a URI a host can follow. `search`, `reviews`, `news`, and the rest are
  list ops.

## Map errors to exit codes

`mapErr` turns the library's sentinels into the `errs` kinds, so every surface
reports the same outcome with the same exit code:

```go
case errors.Is(err, ErrNotFound):
    return errs.NotFound("%s", err.Error())    // exit 6
case errors.Is(err, ErrRateLimited):
    return errs.RateLimited("%s", err.Error()) // exit 5
case errors.Is(err, ErrBlocked):
    return errs.NeedAuth("%s", err.Error())    // exit 4
case errors.Is(err, ErrUsage):
    return errs.Usage("%s", err.Error())       // exit 2
case errors.Is(err, ErrNetwork):
    return errs.Network("%s", err.Error())     // exit 8
```

See [output formats](/reference/output/) for how records render,
[troubleshooting](/reference/troubleshooting/) for the exit-code taxonomy, and
[resource URIs](/guides/resource-uris/) for how a host addresses them.
