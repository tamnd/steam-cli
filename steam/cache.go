package steam

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"
)

// cache.go is a small on-disk response cache keyed by the request URL, so a crawl
// that revisits a page or repeats a query does not refetch it. Entries are plain
// files under CacheDir named by the hash of the URL; freshness is the file mtime
// against CacheTTL. NoCache bypasses it; Refresh ignores hits but still writes.

// cacheName is the on-disk filename for a key.
func cacheName(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:]) + ".cache"
}

// cacheGet returns the cached body for a key when caching is on, the entry exists,
// and it is within ttl. It returns nil otherwise.
func (c *Client) cacheGet(key string) []byte {
	if c.cfg.NoCache || c.cfg.Refresh || c.cfg.CacheDir == "" {
		return nil
	}
	path := filepath.Join(c.cfg.CacheDir, cacheName(key))
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	ttl := c.cfg.CacheTTL
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	if time.Since(info.ModTime()) > ttl {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

// cachePut stores a body for a key when caching is on. Write failures are ignored:
// the cache is an optimization, not a system of record.
func (c *Client) cachePut(key string, body []byte) {
	if c.cfg.NoCache || c.cfg.CacheDir == "" {
		return
	}
	if err := os.MkdirAll(c.cfg.CacheDir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(c.cfg.CacheDir, cacheName(key)), body, 0o644)
}
