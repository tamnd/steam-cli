package steam

import (
	"context"
	"regexp"

	"github.com/tamnd/any-cli/kit/errs"
	"github.com/tamnd/steam-cli/pkg/steamid"
)

// ops.go holds the handler for every operation declared in domain.go. kit reflects
// each input struct into CLI flags, HTTP query params, and MCP tool arguments:
// kit:"arg" is a positional, kit:"flag,inherit" binds the shared --limit, and
// kit:"inject" receives the client newClient builds. The locale and review flags
// are domain-global and reach the client through its config, so they do not repeat
// on every input struct. The reference ops (id, url, steamid) take no client; they
// run offline.

// --- store group ---

type appIn struct {
	Ref    string  `kit:"arg" help:"an appid or a store URL"`
	Client *Client `kit:"inject"`
}

func getApp(ctx context.Context, in appIn, emit func(*App) error) error {
	a, err := in.Client.App(ctx, in.Ref)
	if err != nil {
		return mapErr(err)
	}
	return emit(a)
}

type searchIn struct {
	Query  string  `kit:"arg" help:"a search term"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func search(ctx context.Context, in searchIn, emit func(*App) error) error {
	items, err := in.Client.Search(ctx, in.Query, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

type reviewsIn struct {
	Ref    string  `kit:"arg" help:"an appid or a store URL"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func reviews(ctx context.Context, in reviewsIn, emit func(*Review) error) error {
	items, err := in.Client.Reviews(ctx, in.Ref, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

type packageIn struct {
	ID     string  `kit:"arg" help:"a packageid or a /sub/<id> URL"`
	Client *Client `kit:"inject"`
}

func getPackage(ctx context.Context, in packageIn, emit func(*Package) error) error {
	p, err := in.Client.Package(ctx, in.ID)
	if err != nil {
		return mapErr(err)
	}
	return emit(p)
}

type featuredIn struct {
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func featured(ctx context.Context, in featuredIn, emit func(*App) error) error {
	items, err := in.Client.Featured(ctx, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

// featuredSlice handles the named featured categories (top-sellers, new-releases,
// specials, coming-soon), each one a single key of the featuredcategories
// endpoint.
func featuredSlice(key string) func(context.Context, featuredIn, func(*App) error) error {
	return func(ctx context.Context, in featuredIn, emit func(*App) error) error {
		items, err := in.Client.FeaturedCategory(ctx, key, limitOr(in.Limit, defaultLimit))
		if err != nil {
			return mapErr(err)
		}
		return emitAll(items, emit)
	}
}

type browseIn struct {
	Query    string  `kit:"flag,name=query" help:"a free-text term, omitted for the whole catalog"`
	Sort     string  `kit:"flag,name=sort" help:"sort order: Released_DESC, Reviews_DESC, Price_ASC, Name_ASC"`
	MaxPrice string  `kit:"flag,name=maxprice" help:"price ceiling: free, 5, 10, ..."`
	Start    int     `kit:"flag,name=start" help:"the first result offset"`
	Limit    int     `kit:"flag,inherit"`
	Client   *Client `kit:"inject"`
}

func browse(ctx context.Context, in browseIn, emit func(*App) error) error {
	items, err := in.Client.Browse(ctx, BrowseOpts{
		Query:    in.Query,
		Sort:     in.Sort,
		MaxPrice: in.MaxPrice,
		Start:    in.Start,
		Limit:    limitOr(in.Limit, defaultLimit),
	})
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

type crawlIn struct {
	Ref    string  `kit:"arg" help:"a seed app, package, or profile (an id or a URL)"`
	Depth  int     `kit:"flag,name=depth" help:"how far from the seed to walk" default:"2"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func crawl(ctx context.Context, in crawlIn, emit func(*CrawlNode) error) error {
	return in.Client.Crawl(ctx, in.Ref, in.Depth, limitOr(in.Limit, defaultLimit), emit)
}

type newsIn struct {
	Ref    string  `kit:"arg" help:"an appid or a store URL"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func news(ctx context.Context, in newsIn, emit func(*NewsItem) error) error {
	items, err := in.Client.News(ctx, in.Ref, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

type playersIn struct {
	Ref    string  `kit:"arg" help:"an appid or a store URL"`
	Client *Client `kit:"inject"`
}

func players(ctx context.Context, in playersIn, emit func(*PlayerCount) error) error {
	pc, err := in.Client.Players(ctx, in.Ref)
	if err != nil {
		return mapErr(err)
	}
	return emit(pc)
}

type achievementsIn struct {
	Ref    string  `kit:"arg" help:"an appid or a store URL"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func achievements(ctx context.Context, in achievementsIn, emit func(*Achievement) error) error {
	items, err := in.Client.Achievements(ctx, in.Ref, limitOr(in.Limit, 0))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

// --- player group ---

type profileIn struct {
	Ref    string  `kit:"arg" help:"a SteamID64, a vanity name, or a community URL"`
	Client *Client `kit:"inject"`
}

func getProfile(ctx context.Context, in profileIn, emit func(*Profile) error) error {
	p, err := in.Client.Profile(ctx, in.Ref)
	if err != nil {
		return mapErr(err)
	}
	return emit(p)
}

type resolveIn struct {
	Ref    string  `kit:"arg" help:"a vanity name or a community URL"`
	Client *Client `kit:"inject"`
}

func resolve(ctx context.Context, in resolveIn, emit func(*SteamID) error) error {
	id, err := in.Client.Resolve(ctx, in.Ref)
	if err != nil {
		return mapErr(err)
	}
	return emit(id)
}

// --- market group ---

type marketIn struct {
	Query  string  `kit:"arg" help:"a search term"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func market(ctx context.Context, in marketIn, emit func(*MarketItem) error) error {
	items, err := in.Client.MarketSearch(ctx, in.Query, limitOr(in.Limit, defaultLimit))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

type priceIn struct {
	AppID  string  `kit:"arg" help:"an appid"`
	Name   string  `kit:"arg" help:"a market hash name"`
	Client *Client `kit:"inject"`
}

func price(ctx context.Context, in priceIn, emit func(*MarketPrice) error) error {
	p, err := in.Client.MarketPrice(ctx, in.AppID, in.Name)
	if err != nil {
		return mapErr(err)
	}
	return emit(p)
}

// --- reference tools (offline) ---

type refIn struct {
	Ref string `kit:"arg" help:"any Steam URL, SteamID, vanity, appid, or packageid"`
}

func classifyRef(_ context.Context, in refIn, emit func(*Ref) error) error {
	r := Classify(in.Ref)
	if r.Kind == "unknown" {
		return errs.Usage("unrecognized steam reference: %q", in.Ref)
	}
	return emit(&r)
}

type urlIn struct {
	Kind string `kit:"arg" help:"app, package, profile, reviews, news, search, featured, or market"`
	ID   string `kit:"arg" help:"the id for that kind"`
}

func buildURL(_ context.Context, in urlIn, emit func(*Ref) error) error {
	u := URLFor(in.Kind, in.ID)
	if u == "" {
		return errs.Usage("steam cannot build a URL for %q/%q", in.Kind, in.ID)
	}
	return emit(&Ref{Input: in.Kind + "/" + in.ID, Kind: in.Kind, ID: in.ID, URL: u})
}

type steamIDIn struct {
	Ref string `kit:"arg" help:"a SteamID64, [U:1:N], or STEAM_X:Y:Z"`
}

func convertSteamID(_ context.Context, in steamIDIn, emit func(*SteamID) error) error {
	id, err := steamid.Parse(in.Ref)
	if err == steamid.ErrVanity {
		return emit(&SteamID{
			Input:  in.Ref,
			Kind:   "vanity",
			Vanity: in.Ref,
			URL:    CommunityURL + "/id/" + in.Ref,
		})
	}
	if err != nil {
		return errs.Usage("not a SteamID: %q (resolve a vanity with st resolve)", in.Ref)
	}
	rec := steamIDRecord(in.Ref, classifySteamForm(in.Ref), id)
	return emit(rec)
}

// classifySteamForm names which SteamID form the input was written in.
func classifySteamForm(s string) string {
	switch {
	case formID3RE.MatchString(s):
		return "steamid3"
	case formSteam2RE.MatchString(s):
		return "steam2"
	default:
		return "steamid64"
	}
}

// formID3RE and formSteam2RE label which SteamID form an input used. They only
// need to recognize the shape, since steamid.Parse has already validated it.
var (
	formID3RE    = regexp.MustCompile(`^\[?U:1:\d+\]?$`)
	formSteam2RE = regexp.MustCompile(`^STEAM_[0-5]:[01]:\d+$`)
)

// --- helpers ---

// emitAll streams a slice of records through emit.
func emitAll[T any](items []*T, emit func(*T) error) error {
	for _, it := range items {
		if err := emit(it); err != nil {
			return err
		}
	}
	return nil
}

// limitOr returns the operator's --limit when set, else the command's own default
// fetch count.
func limitOr(limit, def int) int {
	if limit > 0 {
		return limit
	}
	return def
}
