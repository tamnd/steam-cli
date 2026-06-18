package steam

import (
	"context"
	"fmt"
	"strings"
)

// news.go reads the keyless api.steampowered.com/ISteamNews/GetNewsForApp endpoint
// and maps each item to a NewsItem. maxlength=0 keeps the full body.

type newsResponse struct {
	AppNews struct {
		AppID     int            `json:"appid"`
		NewsItems []newsItemWire `json:"newsitems"`
	} `json:"appnews"`
}

type newsItemWire struct {
	GID           string `json:"gid"`
	Title         string `json:"title"`
	URL           string `json:"url"`
	IsExternalURL bool   `json:"is_external_url"`
	Author        string `json:"author"`
	Contents      string `json:"contents"`
	FeedLabel     string `json:"feedlabel"`
	Date          int64  `json:"date"`
	FeedName      string `json:"feedname"`
}

// News returns up to limit news items for appid.
func (c *Client) News(ctx context.Context, appid string, limit int) ([]*NewsItem, error) {
	appid = strings.TrimSpace(appid)
	if !numRE.MatchString(appid) {
		if r := Classify(appid); r.Kind == "app" {
			appid = r.ID
		} else {
			return nil, fmt.Errorf("%w: not an appid: %q", ErrUsage, appid)
		}
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	u := fmt.Sprintf("%s/ISteamNews/GetNewsForApp/v2/?appid=%s&count=%d&maxlength=0",
		c.cfg.APIURL, appid, limit)
	var resp newsResponse
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return nil, err
	}
	out := make([]*NewsItem, 0, len(resp.AppNews.NewsItems))
	for i := range resp.AppNews.NewsItems {
		out = append(out, newsToRecord(&resp.AppNews.NewsItems[i], appid))
	}
	return out, nil
}

func newsToRecord(n *newsItemWire, appid string) *NewsItem {
	return &NewsItem{
		ID:        n.GID,
		App:       appid,
		Title:     n.Title,
		URL:       n.URL,
		Author:    n.Author,
		Body:      n.Contents,
		FeedLabel: n.FeedLabel,
		FeedName:  n.FeedName,
		External:  n.IsExternalURL,
		Date:      unixToRFC(n.Date),
	}
}
