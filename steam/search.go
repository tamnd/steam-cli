package steam

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// search.go reads store.steampowered.com/api/storesearch and maps each hit to a
// lightweight App (id, name, type, price, platforms, header, metacritic, url). The
// record shape is the same as `st app`, so every hit is an addressable steam app a
// reader or a host can follow to the full record.

type storeSearchResponse struct {
	Total int               `json:"total"`
	Items []storeSearchItem `json:"items"`
}

type storeSearchItem struct {
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	ID        int            `json:"id"`
	TinyImage string         `json:"tiny_image"`
	Metascore string         `json:"metascore"`
	Platforms *platformsWire `json:"platforms"`
	Price     *struct {
		Currency string `json:"currency"`
		Initial  int    `json:"initial"`
		Final    int    `json:"final"`
	} `json:"price"`
}

// Search returns up to limit store hits for term as lightweight App records.
func (c *Client) Search(ctx context.Context, term string, limit int) ([]*App, error) {
	u := fmt.Sprintf("%s/api/storesearch/?term=%s&cc=%s&l=%s",
		c.cfg.StoreURL, url.QueryEscape(term), url.QueryEscape(c.cfg.CC), url.QueryEscape(c.cfg.Lang))
	var resp storeSearchResponse
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return nil, err
	}
	out := make([]*App, 0, len(resp.Items))
	for i := range resp.Items {
		out = append(out, searchItemToApp(&resp.Items[i]))
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func searchItemToApp(it *storeSearchItem) *App {
	id := strconv.Itoa(it.ID)
	app := &App{
		ID:          id,
		Name:        it.Name,
		Type:        it.Type,
		HeaderImage: it.TinyImage,
		URL:         StoreURL + "/app/" + id,
		ReviewsRef:  id,
		NewsRef:     id,
	}
	if it.Metascore != "" {
		if n, err := strconv.Atoi(it.Metascore); err == nil {
			app.Metacritic = n
		}
	}
	if it.Platforms != nil {
		app.Platforms = &Platforms{Windows: it.Platforms.Windows, Mac: it.Platforms.Mac, Linux: it.Platforms.Linux}
	}
	if it.Price != nil {
		app.Price = &Price{Currency: it.Price.Currency, Initial: it.Price.Initial, Final: it.Price.Final}
		app.IsFree = it.Price.Final == 0 && it.Price.Initial == 0
	}
	return app
}
