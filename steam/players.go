package steam

import (
	"context"
	"fmt"
	"strings"
)

// players.go reads the keyless
// api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers endpoint and
// returns the live concurrent player count for an app. A result other than 1 means
// the app has no live count (a non-game or an unknown id) and is ErrNotFound.

type playerCountResponse struct {
	Response struct {
		PlayerCount int `json:"player_count"`
		Result      int `json:"result"`
	} `json:"response"`
}

// Players returns the live concurrent player count for appid.
func (c *Client) Players(ctx context.Context, appid string) (*PlayerCount, error) {
	appid = strings.TrimSpace(appid)
	if !numRE.MatchString(appid) {
		if r := Classify(appid); r.Kind == "app" {
			appid = r.ID
		} else {
			return nil, fmt.Errorf("%w: not an appid: %q", ErrUsage, appid)
		}
	}
	u := fmt.Sprintf("%s/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=%s", c.cfg.APIURL, appid)
	var resp playerCountResponse
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return nil, err
	}
	if resp.Response.Result != 1 {
		return nil, ErrNotFound
	}
	return &PlayerCount{
		ID:    appid,
		Count: resp.Response.PlayerCount,
		URL:   StoreURL + "/app/" + appid,
		App:   appid,
	}, nil
}
