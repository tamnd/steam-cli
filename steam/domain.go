package steam

import (
	"context"

	"github.com/tamnd/any-cli/kit"
)

func init() { kit.Register(Domain{}) }

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
}

func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClientWithConfig(DefaultConfig())
	if cfg.UserAgent != "" {
		c.cfg.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.cfg.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.cfg.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.cfg.Timeout = cfg.Timeout
		c.http.Timeout = cfg.Timeout
	}
	return c, nil
}

type listIn struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

func topSellers(ctx context.Context, in listIn, emit func(*Game) error) error {
	games, err := in.Client.TopSellers(ctx, in.Limit)
	if err != nil {
		return err
	}
	for _, g := range games {
		if err := emit(g); err != nil {
			return err
		}
	}
	return nil
}
