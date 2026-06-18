package steam

import (
	"context"
	"fmt"
	"strings"
)

// achievements.go reads the keyless
// api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp
// endpoint and maps each entry to an Achievement with its global unlock rate. The
// endpoint gives only the api name and the percent, so the record carries no
// display name or icon st cannot read keylessly.

type globalAchievementsResponse struct {
	AchievementPercentages struct {
		Achievements []struct {
			Name    string    `json:"name"`
			Percent flexFloat `json:"percent"` // Steam quotes this number
		} `json:"achievements"`
	} `json:"achievementpercentages"`
}

// Achievements returns up to limit global achievement rates for appid, ordered as
// the endpoint returns them (most unlocked first).
func (c *Client) Achievements(ctx context.Context, appid string, limit int) ([]*Achievement, error) {
	appid = strings.TrimSpace(appid)
	if !numRE.MatchString(appid) {
		if r := Classify(appid); r.Kind == "app" {
			appid = r.ID
		} else {
			return nil, fmt.Errorf("%w: not an appid: %q", ErrUsage, appid)
		}
	}
	u := fmt.Sprintf("%s/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/?gameid=%s", c.cfg.APIURL, appid)
	var resp globalAchievementsResponse
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return nil, err
	}
	items := resp.AchievementPercentages.Achievements
	out := make([]*Achievement, 0, len(items))
	for _, a := range items {
		out = append(out, &Achievement{ID: a.Name, App: appid, Percent: float64(a.Percent)})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
