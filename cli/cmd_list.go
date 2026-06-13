package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) topSellersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "top-sellers",
		Short: "Top selling games on the Steam store",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(10)
			a.progressf("fetching top sellers...")
			games, err := a.client.TopSellers(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(games, len(games))
		},
	}
}

func (a *App) newReleasesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new-releases",
		Short: "Recently released games on the Steam store",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(10)
			a.progressf("fetching new releases...")
			games, err := a.client.NewReleases(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(games, len(games))
		},
	}
}

func (a *App) specialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "specials",
		Short: "Games currently on sale on the Steam store",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(10)
			a.progressf("fetching specials...")
			games, err := a.client.Specials(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(games, len(games))
		},
	}
}
