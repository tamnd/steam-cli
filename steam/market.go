package steam

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// market.go reads the community market: search/render lists items for a term, and
// priceoverview gives the lowest, median, and volume for one item. Each market
// record carries the appid the item belongs to, so a crawl walks a listing back to
// its app.

type marketSearchResponse struct {
	Success    bool               `json:"success"`
	TotalCount int                `json:"total_count"`
	Results    []marketResultWire `json:"results"`
}

type marketResultWire struct {
	Name             string `json:"name"`
	HashName         string `json:"hash_name"`
	SellListings     int    `json:"sell_listings"`
	SellPrice        int    `json:"sell_price"`
	SellPriceText    string `json:"sell_price_text"`
	AppIcon          string `json:"app_icon"`
	AppName          string `json:"app_name"`
	AssetDescription struct {
		AppID          int    `json:"appid"`
		Type           string `json:"type"`
		MarketHashName string `json:"market_hash_name"`
	} `json:"asset_description"`
}

// MarketSearch returns up to limit community market listings for term.
func (c *Client) MarketSearch(ctx context.Context, term string, limit int) ([]*MarketItem, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	u := fmt.Sprintf("%s/market/search/render/?query=%s&start=0&count=%d&norender=1",
		c.cfg.CommunityURL, url.QueryEscape(term), limit)
	var resp marketSearchResponse
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return nil, err
	}
	out := make([]*MarketItem, 0, len(resp.Results))
	for i := range resp.Results {
		out = append(out, marketResultToItem(&resp.Results[i]))
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func marketResultToItem(r *marketResultWire) *MarketItem {
	hash := r.HashName
	if hash == "" {
		hash = r.AssetDescription.MarketHashName
	}
	appid := strconv.Itoa(r.AssetDescription.AppID)
	return &MarketItem{
		ID:            hash,
		Name:          r.Name,
		App:           appid,
		AppName:       r.AppName,
		SellListings:  r.SellListings,
		SellPrice:     r.SellPrice,
		SellPriceText: r.SellPriceText,
		Type:          r.AssetDescription.Type,
		Icon:          r.AppIcon,
		URL:           CommunityURL + "/market/listings/" + appid + "/" + url.PathEscape(hash),
	}
}

type priceOverviewResponse struct {
	Success     bool   `json:"success"`
	LowestPrice string `json:"lowest_price"`
	MedianPrice string `json:"median_price"`
	Volume      string `json:"volume"`
}

// MarketPrice returns the price overview for one market item, addressed by its app
// and its market hash name.
func (c *Client) MarketPrice(ctx context.Context, appid, hashName string) (*MarketPrice, error) {
	if appid == "" || hashName == "" {
		return nil, fmt.Errorf("%w: price needs an appid and a market hash name", ErrUsage)
	}
	if !numRE.MatchString(appid) {
		if r := Classify(appid); r.Kind == "app" {
			appid = r.ID
		} else {
			return nil, fmt.Errorf("%w: not an appid: %q", ErrUsage, appid)
		}
	}
	u := fmt.Sprintf("%s/market/priceoverview/?appid=%s&currency=%d&market_hash_name=%s",
		c.cfg.CommunityURL, appid, c.cfg.Currency, url.QueryEscape(hashName))
	var resp priceOverviewResponse
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, ErrNotFound
	}
	return &MarketPrice{
		ID:          hashName,
		App:         appid,
		Currency:    c.cfg.Currency,
		LowestPrice: resp.LowestPrice,
		MedianPrice: resp.MedianPrice,
		Volume:      resp.Volume,
		URL:         CommunityURL + "/market/listings/" + appid + "/" + url.PathEscape(hashName),
	}, nil
}
