package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/steam-cli/steam"
)

func (a *App) reviewsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reviews <appid|URL>",
		Short: "Show user reviews for a Steam game",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := steam.ParseAppID(args[0])
			if err != nil {
				return codeError(exitUsage, err)
			}
			n := a.effectiveLimit(20)
			a.progressf("fetching reviews for game %d...", id)
			reviews, err := a.client.Reviews(cmd.Context(), id, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(reviews, len(reviews))
		},
	}
}
