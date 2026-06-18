// Package steam is the library behind the st command line: the HTTP client, the
// offline reference layer, and the typed records read from public Steam surfaces.
//
// Steam has one keyless access plane that spans three hosts. The storefront
// (store.steampowered.com) answers app details, search, reviews, and package
// details as JSON. The community site (steamcommunity.com) serves public profiles
// as XML and the community market as JSON. A keyless subset of api.steampowered.com
// answers the full app catalog, a game's news, its live player count, and its
// global achievement rates. None of this needs an account or a key, so the Client
// below is a plain paced, retrying, caching GET client with no auth handshake. It
// turns a walled or rejected response into a typed error the exit-code mapping
// understands.
package steam

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client reads public Steam data over HTTP.
type Client struct {
	HTTP *http.Client
	cfg  Config

	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured from cfg, filling unset fields with their
// defaults.
func NewClient(cfg Config) *Client {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.CC == "" {
		cfg.CC = "us"
	}
	if cfg.Lang == "" {
		cfg.Lang = "english"
	}
	if cfg.Currency == 0 {
		cfg.Currency = 1
	}
	if cfg.StoreURL == "" {
		cfg.StoreURL = StoreURL
	}
	if cfg.CommunityURL == "" {
		cfg.CommunityURL = CommunityURL
	}
	if cfg.APIURL == "" {
		cfg.APIURL = APIURL
	}
	return &Client{
		HTTP: &http.Client{Timeout: cfg.Timeout},
		cfg:  cfg,
	}
}

// get fetches a URL and returns the body. It serves from cache when fresh, paces
// and retries transient failures, and classifies a walled response as ErrBlocked.
// The cache key is the URL.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	if b := c.cacheGet(rawURL); b != nil {
		return b, nil
	}
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
			c.cachePut(rawURL, body)
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	if errors.Is(lastErr, ErrRateLimited) {
		return nil, ErrRateLimited
	}
	return nil, fmt.Errorf("get %s: %w: %v", rawURL, ErrNetwork, lastErr)
}

// getJSON fetches a URL and decodes the body into v.
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json, text/xml, application/xml, text/html")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusForbidden, resp.StatusCode == http.StatusServiceUnavailable:
		return nil, false, ErrBlocked
	case resp.StatusCode == http.StatusNotFound:
		return nil, false, ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, true, ErrRateLimited
	case resp.StatusCode >= 500:
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	if isChallenge(b) {
		return nil, false, ErrBlocked
	}
	return b, false, nil
}

// pace blocks until at least Delay has passed since the previous request.
func (c *Client) pace() {
	if c.cfg.Delay <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := c.cfg.Delay - time.Since(c.last); wait > 0 {
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

// challengeMarkers are byte signatures of an anti-bot interstitial served with a
// 200 in place of the real response.
var challengeMarkers = [][]byte{
	[]byte("challenges.cloudflare.com"),
	[]byte("window._cf_chl_opt"),
	[]byte("just a moment..."),
	[]byte("enable javascript and cookies to continue"),
	[]byte("cf-browser-verification"),
}

// isChallenge reports whether a 200 body is an anti-bot challenge rather than a
// real response, by looking for a known marker in the head of the body. The store
// JSON and the api endpoints never carry these markers, so a JSON body is never a
// false positive.
func isChallenge(body []byte) bool {
	head := body
	if len(head) > 8192 {
		head = head[:8192]
	}
	lower := bytes.ToLower(head)
	for _, m := range challengeMarkers {
		if bytes.Contains(lower, m) {
			return true
		}
	}
	return false
}

// squish collapses internal whitespace and trims, for text pulled out of HTML or
// XML.
func squish(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
