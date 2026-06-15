package steam

import (
	"context"
	"fmt"

	"github.com/tamnd/any-cli/kit"
)

func init() { kit.Register(Domain{}) }

// Domain is the steam kit driver. Register it with a blank import:
//
//	import _ "github.com/tamnd/steam-cli/steam"
type Domain struct{}

func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "steam",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "steam",
			Short:  "A command line for Steam.",
			Long: `A command line for Steam.

Browse the Steam store. No API key required.`,
			Site: "https://" + Host,
			Repo: "https://github.com/tamnd/steam-cli",
		},
	}
}

func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "top-sellers", Group: "store", List: true,
		URIType: "game", Summary: "Steam top selling games"}, topSellers)

	kit.Handle(app, kit.OpMeta{Name: "new-releases", Group: "store", List: true,
		URIType: "game", Summary: "Steam new releases"}, newReleases)

	kit.Handle(app, kit.OpMeta{Name: "specials", Group: "store", List: true,
		URIType: "game", Summary: "Steam games currently on sale"}, specials)

	kit.Handle(app, kit.OpMeta{Name: "search", Group: "store", List: true,
		URIType: "game", Summary: "Search the Steam store",
		Args: []kit.Arg{{Name: "query", Help: "search terms"}}}, searchGames)

	kit.Handle(app, kit.OpMeta{Name: "game", Group: "store", Single: true,
		Resolver: true, URIType: "game", Summary: "Steam game details",
		Args: []kit.Arg{{Name: "ref", Help: "app id or store URL"}}}, getGame)
}

func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

type listIn struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type searchIn struct {
	Query  string  `kit:"arg" help:"search terms"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type gameIn struct {
	Ref    string  `kit:"arg" help:"app id or store URL"`
	Client *Client `kit:"inject"`
}

func topSellers(ctx context.Context, in listIn, emit func(*Game) error) error {
	games, err := in.Client.TopSellers(ctx, in.Limit)
	if err != nil {
		return err
	}
	for i := range games {
		if err := emit(&games[i]); err != nil {
			return err
		}
	}
	return nil
}

func newReleases(ctx context.Context, in listIn, emit func(*Game) error) error {
	games, err := in.Client.NewReleases(ctx, in.Limit)
	if err != nil {
		return err
	}
	for i := range games {
		if err := emit(&games[i]); err != nil {
			return err
		}
	}
	return nil
}

func specials(ctx context.Context, in listIn, emit func(*Game) error) error {
	games, err := in.Client.Specials(ctx, in.Limit)
	if err != nil {
		return err
	}
	for i := range games {
		if err := emit(&games[i]); err != nil {
			return err
		}
	}
	return nil
}

func searchGames(ctx context.Context, in searchIn, emit func(*Game) error) error {
	games, err := in.Client.Search(ctx, in.Query, in.Limit)
	if err != nil {
		return err
	}
	for i := range games {
		if err := emit(&games[i]); err != nil {
			return err
		}
	}
	return nil
}

func getGame(ctx context.Context, in gameIn, emit func(*GameDetail) error) error {
	appid, err := ParseAppID(in.Ref)
	if err != nil {
		return fmt.Errorf("invalid ref %q: %w", in.Ref, err)
	}
	detail, err := in.Client.GameDetails(ctx, appid)
	if err != nil {
		return err
	}
	return emit(&detail)
}
