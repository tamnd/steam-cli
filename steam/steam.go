package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const Host = "store.steampowered.com"
const baseURL = "https://store.steampowered.com"
const DefaultUserAgent = "Mozilla/5.0 (compatible; steam-cli/0.1; +https://github.com/tamnd/steam-cli)"

type Config struct {
	BaseURL   string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
	UserAgent string
}

func DefaultConfig() Config {
	return Config{
		BaseURL:   baseURL,
		Rate:      time.Second,
		Retries:   3,
		Timeout:   30 * time.Second,
		UserAgent: DefaultUserAgent,
	}
}

type Client struct {
	cfg  Config
	http *http.Client
	last time.Time
}

func NewClient() *Client { return NewClientWithConfig(DefaultConfig()) }

func NewClientWithConfig(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// wireTopSellers is the raw API response structure.
type wireTopSellers struct {
	TopSellers struct {
		Items []wireItem `json:"items"`
	} `json:"top_sellers"`
}

type wireItem struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	DiscountPercent int    `json:"discount_percent"`
	FinalPrice      int    `json:"final_price"`
	Currency        string `json:"currency"`
}

// Game is a Steam game entry.
type Game struct {
	ID       string `json:"id"       kit:"id" table:"id"`
	Name     string `json:"name"              table:"name"`
	Price    string `json:"price"             table:"price"`
	Discount string `json:"discount"          table:"discount"`
	URL      string `json:"url"               table:"url,url"`
}

// TopSellers returns the Steam top selling games.
func (c *Client) TopSellers(ctx context.Context, limit int) ([]*Game, error) {
	base := c.cfg.BaseURL
	if base == "" {
		base = baseURL
	}
	body, err := c.get(ctx, base+"/api/featuredcategories/?cc=us&l=en")
	if err != nil {
		return nil, err
	}
	var wire wireTopSellers
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	items := wire.TopSellers.Items
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	games := make([]*Game, 0, len(items))
	for _, item := range items {
		g := &Game{
			ID:   fmt.Sprintf("%d", item.ID),
			Name: item.Name,
			URL:  fmt.Sprintf("https://store.steampowered.com/app/%d", item.ID),
		}
		if item.FinalPrice == 0 {
			g.Price = "Free"
		} else {
			g.Price = fmt.Sprintf("$%.2f", float64(item.FinalPrice)/100)
		}
		if item.DiscountPercent > 0 {
			g.Discount = fmt.Sprintf("-%d%%", item.DiscountPercent)
		}
		games = append(games, g)
	}
	return games, nil
}
