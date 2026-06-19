package steam

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes steam as a kit Domain: a driver a multi-domain host enables
// with a single blank import,
//
//	import _ "github.com/tamnd/steam-cli/steam"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// steam:// URIs by routing to the operations Register installs. The same Domain
// also builds the standalone st binary (see cli.NewApp), so the binary and a host
// share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the steam driver. It carries no state; the per-run client is built by
// the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme:   "steam",
		Hosts:    []string{StoreHost, CommunityHost, APIHost},
		Identity: Identity(),
	}
}

// Identity is the fixed description of the Steam CLI, shared by the domain and the
// standalone composition root so help and version read the same.
func Identity() kit.Identity {
	return kit.Identity{
		Binary: "st",
		Short:  "Read public Steam apps, packages, reviews, news, players, market, and profiles into structured records",
		Long: `st reads public Steam data over plain HTTPS with no API key and no login.
It reads one keyless plane across three hosts: store.steampowered.com for app
details, search, reviews, packages, and the featured lists; the keyless subset
of api.steampowered.com for news, the live player count, and global achievement
rates; and steamcommunity.com for public profiles and the community market. The
numeric appid is the universal key, so an app, its
reviews, its news, its player count, and its achievements all address the same
record, and the records link into one graph. st returns records as a table,
JSON, JSONL, CSV, TSV, or URLs, and serves the same operations over HTTP and
MCP.

st is an independent tool and is not affiliated with Valve or Steam.`,
		Site: StoreURL,
		Repo: "https://github.com/tamnd/steam-cli",
	}
}

// Register installs the client factory and every operation onto app. The store
// group reads the storefront and the keyless web API; the player group reads
// public community profiles; the market group reads the community market; the ref
// group is offline and never touches the network.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)
	app.CommandGroup("store", "Read public Steam store and catalog data")
	app.CommandGroup("player", "Read public Steam community profiles")
	app.CommandGroup("market", "Read the Steam community market")
	app.CommandGroup("ref", "Resolve references to ids and URLs (offline)")

	kit.Handle(app, kit.OpMeta{
		Name: "app", Group: "store", Single: true,
		Summary: "Show one store app in full (details, then the store-page island)",
		URIType: "app", Resolver: true,
		Args: []kit.Arg{{Name: "ref", Help: "an appid or a store URL"}},
	}, getApp)

	kit.Handle(app, kit.OpMeta{
		Name: "search", Group: "store", List: true,
		Summary: "Search the store for apps",
		URIType: "search",
		Args:    []kit.Arg{{Name: "query", Help: "a search term"}},
	}, search)

	kit.Handle(app, kit.OpMeta{
		Name: "browse", Group: "store", List: true,
		Summary: "Page through the whole store catalog (the discovery seed)",
		URIType: "browse",
	}, browse)

	kit.Handle(app, kit.OpMeta{
		Name: "crawl", Group: "store", List: true,
		Summary: "Walk the public graph breadth-first from a seed app, package, or profile",
		Args:    []kit.Arg{{Name: "ref", Help: "a seed app, package, or profile (an id or a URL)"}},
	}, crawl)

	kit.Handle(app, kit.OpMeta{
		Name: "reviews", Group: "store", List: true,
		Summary: "List an app's user reviews (cursor-paged)",
		URIType: "reviews",
		Args:    []kit.Arg{{Name: "ref", Help: "an appid or a store URL"}},
	}, reviews)

	kit.Handle(app, kit.OpMeta{
		Name: "package", Group: "store", Single: true,
		Summary: "Show one store package (a sub) with the apps it bundles",
		URIType: "package", Resolver: true,
		Args: []kit.Arg{{Name: "id", Help: "a packageid or a /sub/<id> URL"}},
	}, getPackage)

	kit.Handle(app, kit.OpMeta{
		Name: "featured", Group: "store", List: true,
		Summary: "List the featured store categories' apps",
		URIType: "featured",
	}, featured)

	kit.Handle(app, kit.OpMeta{
		Name: "top-sellers", Group: "store", List: true,
		Summary: "List the current top-selling apps",
		URIType: "featured",
	}, featuredSlice("top_sellers"))

	kit.Handle(app, kit.OpMeta{
		Name: "new-releases", Group: "store", List: true,
		Summary: "List the new-release apps",
		URIType: "featured",
	}, featuredSlice("new_releases"))

	kit.Handle(app, kit.OpMeta{
		Name: "specials", Group: "store", List: true,
		Summary: "List the apps currently on sale",
		URIType: "featured",
	}, featuredSlice("specials"))

	kit.Handle(app, kit.OpMeta{
		Name: "coming-soon", Group: "store", List: true,
		Summary: "List the upcoming apps",
		URIType: "featured",
	}, featuredSlice("coming_soon"))

	kit.Handle(app, kit.OpMeta{
		Name: "news", Group: "store", List: true,
		Summary: "List an app's news and announcements",
		URIType: "news",
		Args:    []kit.Arg{{Name: "ref", Help: "an appid or a store URL"}},
	}, news)

	kit.Handle(app, kit.OpMeta{
		Name: "players", Group: "store", Single: true,
		Summary: "Show an app's live concurrent player count",
		URIType: "players",
		Args:    []kit.Arg{{Name: "ref", Help: "an appid or a store URL"}},
	}, players)

	kit.Handle(app, kit.OpMeta{
		Name: "achievements", Group: "store", List: true,
		Summary: "List an app's global achievement unlock rates",
		URIType: "achievements",
		Args:    []kit.Arg{{Name: "ref", Help: "an appid or a store URL"}},
	}, achievements)

	kit.Handle(app, kit.OpMeta{
		Name: "profile", Group: "player", Single: true,
		Summary: "Show one public community profile",
		URIType: "profile", Resolver: true,
		Args: []kit.Arg{{Name: "ref", Help: "a SteamID64, a vanity name, or a community URL"}},
	}, getProfile)

	kit.Handle(app, kit.OpMeta{
		Name: "resolve", Group: "player", Single: true,
		Summary: "Resolve a vanity name into every SteamID form",
		Args:    []kit.Arg{{Name: "ref", Help: "a vanity name or a community URL"}},
	}, resolve)

	kit.Handle(app, kit.OpMeta{
		Name: "market", Group: "market", List: true,
		Summary: "Search the community market for items",
		URIType: "market",
		Args:    []kit.Arg{{Name: "query", Help: "a search term"}},
	}, market)

	kit.Handle(app, kit.OpMeta{
		Name: "price", Group: "market", Single: true,
		Summary: "Show the lowest, median, and volume for one market item",
		Args: []kit.Arg{
			{Name: "appid", Help: "an appid"},
			{Name: "name", Help: "a market hash name"},
		},
	}, price)

	// Reference tools (offline).
	kit.Handle(app, kit.OpMeta{
		Name: "id", Parent: "ref", Single: true,
		Summary: "Classify a reference into its (kind, id)",
		Args:    []kit.Arg{{Name: "ref", Help: "any Steam URL, SteamID, vanity, appid, or packageid"}},
	}, classifyRef)

	kit.Handle(app, kit.OpMeta{
		Name: "url", Parent: "ref", Single: true,
		Summary: "Build the addressable URL for a (kind, id)",
		Args: []kit.Arg{
			{Name: "kind", Help: "app, package, profile, reviews, news, search, featured, or market"},
			{Name: "id", Help: "the id for that kind"},
		},
	}, buildURL)

	kit.Handle(app, kit.OpMeta{
		Name: "steamid", Parent: "ref", Single: true,
		Summary: "Convert a SteamID between its 64-bit, [U:1:N], and STEAM_X:Y:Z forms",
		Args:    []kit.Arg{{Name: "ref", Help: "a SteamID64, [U:1:N], or STEAM_X:Y:Z"}},
	}, convertSteamID)
}

// newClient builds the client from the host-resolved config, so a host and the
// standalone binary pace and identify themselves the same way.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	return ClientFromConfig(cfg), nil
}

// ClientFromConfig maps the framework config onto a steam.Config and returns a
// client. There are no credentials to read; the only domain-specific knobs are the
// storefront locale and the review filters.
func ClientFromConfig(cfg kit.Config) *Client {
	sc := DefaultConfig()
	if cfg.Rate > 0 {
		sc.Delay = cfg.Rate
	}
	if cfg.Retries >= 0 {
		sc.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		sc.Timeout = cfg.Timeout
	}
	if ua := cfg.Extra["user-agent"]; ua != "" {
		sc.UserAgent = ua
	} else if cfg.UserAgent != "" {
		sc.UserAgent = cfg.UserAgent
	}
	if v := cfg.Extra["cc"]; v != "" {
		sc.CC = v
	}
	if v := cfg.Extra["lang"]; v != "" {
		sc.Lang = v
	}
	if v := cfg.Extra["currency"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			sc.Currency = n
		}
	}
	if v := cfg.Extra["review-filter"]; v != "" {
		sc.ReviewFilter = v
	}
	if v := cfg.Extra["review-language"]; v != "" {
		sc.ReviewLanguage = v
	}
	if v := cfg.Extra["purchase-type"]; v != "" {
		sc.PurchaseType = v
	}
	sc.CacheDir = cfg.CacheDir
	sc.NoCache = cfg.NoCache
	if ttl := cfg.Extra["cache-ttl"]; ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			sc.CacheTTL = d
		}
	}
	sc.Refresh = cfg.Extra["refresh"] == "true"
	return NewClient(sc)
}

// Defaults seeds the framework baseline with steam's own values, so an unset
// --rate or --timeout uses the steam default rather than the generic kit one.
func Defaults(c *kit.Config) {
	def := DefaultConfig()
	c.Rate = def.Delay
	c.Retries = def.Retries
	c.Timeout = def.Timeout
	c.UserAgent = def.UserAgent
}

// Classify turns any accepted input into the canonical (type, id), so a host's
// resolve and url touch no network.
func (Domain) Classify(input string) (uriType, id string, err error) {
	r := Classify(input)
	if r.Kind == "unknown" {
		return "", "", errs.Usage("unrecognized steam reference: %q", input)
	}
	return r.Kind, r.ID, nil
}

// Locate is the inverse: the addressable URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	u := URLFor(uriType, id)
	if u == "" {
		return "", errs.Usage("steam has no resource type %q", uriType)
	}
	return u, nil
}

// mapErr translates a library error into a kit error so the exit code matches the
// rest of the fleet: a missing entity reads as not found (exit 6), a throttle as
// rate limited (exit 5), the anti-bot wall as need-auth (exit 4), a caught bad
// argument as usage (exit 2), and a transport failure that survives every retry as
// a network error (exit 8). There is no need-key case, because st reads only
// keyless surfaces.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return errs.NotFound("%s", err.Error())
	case errors.Is(err, ErrRateLimited):
		return errs.RateLimited("%s", err.Error())
	case errors.Is(err, ErrBlocked):
		return errs.NeedAuth("%s", err.Error())
	case errors.Is(err, ErrUsage):
		return errs.Usage("%s", err.Error())
	case errors.Is(err, ErrNetwork):
		return errs.Network("%s", err.Error())
	default:
		return err
	}
}
