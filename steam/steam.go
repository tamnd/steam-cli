// Package steam is the library behind the steam command: the HTTP client,
// request shaping, and typed data models for the Steam game store.
//
// Two base APIs:
//   - https://store.steampowered.com/api — store catalog (no key required)
//   - appreviews endpoint on store.steampowered.com (no key required)
package steam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultAPIBase = "https://store.steampowered.com/api"

// DefaultUserAgent identifies the client to Steam.
const DefaultUserAgent = "steam/dev (+https://github.com/tamnd/steam-cli)"

// ErrNotFound is returned when an app is unavailable or the API returns no data.
var ErrNotFound = errors.New("not found")

// Client talks to the Steam store APIs.
type Client struct {
	httpClient *http.Client
	userAgent  string
	rate       time.Duration
	retries    int
	baseURL    string // e.g. https://store.steampowered.com/api
	storeRoot  string // e.g. https://store.steampowered.com
	mu         sync.Mutex
	last       time.Time
}

// Config holds constructor parameters.
type Config struct {
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
	// BaseURL overrides the Steam store API base for testing.
	// Defaults to "https://store.steampowered.com/api".
	BaseURL string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
		BaseURL:   defaultAPIBase,
	}
}

// NewClient returns a Client configured from cfg.
func NewClient(cfg Config) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = defaultAPIBase
	}
	// Derive storeRoot: strip trailing "/api" suffix if present.
	root := strings.TrimSuffix(base, "/api")
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
		baseURL:    base,
		storeRoot:  root,
	}
}

// get fetches a URL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
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
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
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
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
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

// getJSON fetches and JSON-decodes into v. Returns ErrNotFound when the body is null.
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "null" {
		return ErrNotFound
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// ─── featured categories ─────────────────────────────────────────────────────

func (c *Client) fetchFeatured(ctx context.Context) (featuredCategoriesResp, error) {
	rawURL := c.baseURL + "/featuredcategories/?cc=us&l=en"
	var resp featuredCategoriesResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return featuredCategoriesResp{}, err
	}
	return resp, nil
}

// TopSellers returns the top-selling games on the store.
func (c *Client) TopSellers(ctx context.Context, limit int) ([]Game, error) {
	fc, err := c.fetchFeatured(ctx)
	if err != nil {
		return nil, err
	}
	items := fc.TopSellers.Items
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	out := make([]Game, len(items))
	for i, item := range items {
		out[i] = featuredItemToGame(item, i+1)
	}
	return out, nil
}

// NewReleases returns the newest released games on the store.
func (c *Client) NewReleases(ctx context.Context, limit int) ([]Game, error) {
	fc, err := c.fetchFeatured(ctx)
	if err != nil {
		return nil, err
	}
	items := fc.NewReleases.Items
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	out := make([]Game, len(items))
	for i, item := range items {
		out[i] = featuredItemToGame(item, i+1)
	}
	return out, nil
}

// Specials returns games currently on sale (discount_percent > 0).
func (c *Client) Specials(ctx context.Context, limit int) ([]Game, error) {
	fc, err := c.fetchFeatured(ctx)
	if err != nil {
		return nil, err
	}
	var out []Game
	rank := 1
	for _, item := range fc.Specials.Items {
		if item.DiscountPercent <= 0 {
			continue
		}
		out = append(out, featuredItemToGame(item, rank))
		rank++
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// ─── search ──────────────────────────────────────────────────────────────────

// Search searches the Steam store for query and returns up to limit results.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Game, error) {
	params := url.Values{}
	params.Set("term", query)
	params.Set("l", "en")
	params.Set("cc", "us")
	rawURL := c.baseURL + "/storesearch/?" + params.Encode()

	var resp storeSearchResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}

	items := resp.Items
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	out := make([]Game, len(items))
	for i, item := range items {
		out[i] = searchItemToGame(item, i+1)
	}
	return out, nil
}

// ─── game details ─────────────────────────────────────────────────────────────

// GameDetails returns full details for a single app by appid.
func (c *Client) GameDetails(ctx context.Context, appid int) (GameDetail, error) {
	params := url.Values{}
	params.Set("appids", strconv.Itoa(appid))
	params.Set("cc", "us")
	params.Set("l", "en")
	rawURL := c.baseURL + "/appdetails/?" + params.Encode()

	var outer appDetailsOuter
	if err := c.getJSON(ctx, rawURL, &outer); err != nil {
		return GameDetail{}, err
	}

	key := strconv.Itoa(appid)
	entry, ok := outer[key]
	if !ok || !entry.Success {
		return GameDetail{}, ErrNotFound
	}
	return appDataToDetail(entry.Data), nil
}

// ─── reviews ─────────────────────────────────────────────────────────────────

// Reviews returns user reviews for appid. It paginates until limit is reached.
func (c *Client) Reviews(ctx context.Context, appid, limit int) ([]Review, error) {
	cursor := "*"
	var out []Review
	for {
		params := url.Values{}
		params.Set("json", "1")
		params.Set("filter", "all")
		params.Set("language", "english")
		params.Set("cursor", cursor)
		params.Set("review_type", "all")
		params.Set("purchase_type", "all")
		params.Set("num_per_page", "20")
		rawURL := fmt.Sprintf("%s/appreviews/%d?%s", c.storeRoot, appid, params.Encode())

		var resp reviewsResp
		if err := c.getJSON(ctx, rawURL, &resp); err != nil {
			return out, err
		}
		if resp.Success != 1 {
			return out, fmt.Errorf("reviews api returned success=%d", resp.Success)
		}

		for _, rw := range resp.Reviews {
			out = append(out, reviewWireToReview(rw, len(out)+1))
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}

		if len(resp.Reviews) == 0 || resp.Cursor == "" || resp.Cursor == cursor {
			break
		}
		cursor = resp.Cursor
	}
	return out, nil
}

// ─── ParseAppID ──────────────────────────────────────────────────────────────

// ParseAppID parses a bare integer or a Steam store URL into an appid.
// Accepted forms:
//
//	570
//	https://store.steampowered.com/app/570/
//	https://store.steampowered.com/app/570/Dota_2/
func ParseAppID(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty app id")
	}
	// Fast path: bare integer.
	if id, err := strconv.Atoi(s); err == nil {
		if id <= 0 {
			return 0, fmt.Errorf("invalid app id %q", s)
		}
		return id, nil
	}
	// URL path: extract segment after /app/
	const marker = "/app/"
	idx := strings.Index(s, marker)
	if idx < 0 {
		return 0, fmt.Errorf("cannot parse app id from %q", s)
	}
	rest := s[idx+len(marker):]
	// Take everything up to the next slash or end.
	if slash := strings.Index(rest, "/"); slash >= 0 {
		rest = rest[:slash]
	}
	id, err := strconv.Atoi(rest)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("cannot parse app id from %q", s)
	}
	return id, nil
}
