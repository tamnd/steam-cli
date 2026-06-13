package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/steam-cli/steam"
)

func (a *App) gameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "game <appid|URL>",
		Short: "Show details for a single Steam game",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := steam.ParseAppID(args[0])
			if err != nil {
				return codeError(exitUsage, err)
			}
			a.progressf("fetching game %d...", id)
			detail, err := a.client.GameDetails(cmd.Context(), id)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]steam.GameDetail{detail})
		},
	}
}
