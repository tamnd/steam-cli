---
title: "Troubleshooting"
description: "The handful of things that trip people up, and how to fix each one."
weight: 40
---

Most of these come down to network reality or how Steam serves its data, not a
bug. st reads public data only: there is no login, no cookie, and no token to set.

## Exit codes

st maps each outcome to a stable exit code, so a script can branch on it:

| Code | Meaning |
|---|---|
| 0 | Success |
| 2 | Usage error (a bad argument or reference) |
| 3 | No results |
| 4 | A walled surface (an anti-bot block or an interstitial) |
| 5 | Rate limited |
| 6 | Not found |
| 8 | Network error |

## Requests start failing or returning 429

Steam rate-limits like any public site. st already paces requests and retries the
transient failures, but a hard limit still means backing off. Raise the delay
between requests with `--rate` (for example `--rate 1s`) and retry later. A burst
of 429 responses is the site asking you to slow down, not a defect, and it exits
with code 5.

## A surface is walled (exit 4)

Some requests come back as an anti-bot block or a challenge interstitial rather
than data. st detects that and exits 4 rather than pretending it succeeded. It
does not solve the challenge, forge a clearance cookie, or rotate proxies. Try
again later from a different network, or accept that the surface is not readable
without a session st does not hold.

## Community surfaces time out (exit 8)

`profile`, `resolve`, `market`, and `price` read `steamcommunity.com`. Some
networks and datacenter IP ranges cannot reach that host even when
`store.steampowered.com` and `api.steampowered.com` are fine, and the request
times out as a network error (exit 8). The store and web-API commands still work
from the same machine. If the community commands time out everywhere, check
whether `steamcommunity.com` is reachable from your network at all.

## Nothing is found for something you expected

The public surface is not the whole site. Some data sits behind a login, a region,
or a page that only renders with JavaScript, and that part is not reachable
anonymously. Check that the input is spelled the way Steam uses it, try `--cc` and
`--lang` for a different storefront, and see whether the same thing is visible in a
private browser window before assuming it is missing. A genuinely empty result
exits with code 3.

## The binary is not on your PATH

`go install` puts the binary in `$(go env GOPATH)/bin` (usually `~/go/bin`), and a
release archive leaves it wherever you unpacked it. If your shell cannot find
`st`, add that directory to your `PATH`. See
[installation](/getting-started/installation/).

## Seeing what st actually did

When something behaves unexpectedly, `-v` adds per-request detail so you can see
the URLs it hit and the responses it got. That is usually enough to tell a rate
limit apart from a walled surface apart from a genuinely empty result.
