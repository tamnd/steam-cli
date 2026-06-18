package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// featured.go reads store.steampowered.com/api/featuredcategories and flattens the
// promoted categories (specials, top sellers, new releases, coming soon) into
// lightweight App records, de-duplicated by appid. The top-level object mixes
// category objects with scalar status fields, so each value is tried as a category
// and the ones that do not parse are skipped.

type featuredCategory struct {
	Items []featuredItem `json:"items"`
}

type featuredItem struct {
	ID               int    `json:"id"`
	Type             int    `json:"type"`
	Name             string `json:"name"`
	DiscountPercent  int    `json:"discount_percent"`
	OriginalPrice    int    `json:"original_price"`
	FinalPrice       int    `json:"final_price"`
	Currency         string `json:"currency"`
	HeaderImage      string `json:"header_image"`
	LargeCapsule     string `json:"large_capsule_image"`
	WindowsAvailable bool   `json:"windows_available"`
	MacAvailable     bool   `json:"mac_available"`
	LinuxAvailable   bool   `json:"linux_available"`
}

// Featured returns up to limit promoted apps, de-duplicated by appid.
func (c *Client) Featured(ctx context.Context, limit int) ([]*App, error) {
	u := fmt.Sprintf("%s/api/featuredcategories/?cc=%s&l=%s", c.cfg.StoreURL, c.cfg.CC, c.cfg.Lang)
	var raw map[string]json.RawMessage
	if err := c.getJSON(ctx, u, &raw); err != nil {
		return nil, err
	}
	var out []*App
	seen := map[string]bool{}
	for _, v := range raw {
		var cat featuredCategory
		if err := json.Unmarshal(v, &cat); err != nil || len(cat.Items) == 0 {
			continue
		}
		for i := range cat.Items {
			it := &cat.Items[i]
			if it.ID == 0 {
				continue
			}
			id := strconv.Itoa(it.ID)
			if seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, featuredItemToApp(it))
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func featuredItemToApp(it *featuredItem) *App {
	id := strconv.Itoa(it.ID)
	app := &App{
		ID:          id,
		Name:        it.Name,
		HeaderImage: firstNonEmpty(it.HeaderImage, it.LargeCapsule),
		Platforms:   &Platforms{Windows: it.WindowsAvailable, Mac: it.MacAvailable, Linux: it.LinuxAvailable},
		URL:         StoreURL + "/app/" + id,
		ReviewsRef:  id,
		NewsRef:     id,
	}
	if it.Currency != "" || it.FinalPrice != 0 {
		app.Price = &Price{
			Currency:    it.Currency,
			Initial:     it.OriginalPrice,
			Final:       it.FinalPrice,
			DiscountPct: it.DiscountPercent,
		}
	}
	app.IsFree = it.FinalPrice == 0 && it.OriginalPrice == 0 && it.Currency == ""
	return app
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
