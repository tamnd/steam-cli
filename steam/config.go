package steam

import "time"

// config.go holds the resolved settings a Client reads. domain.go's
// ClientFromConfig maps the framework's kit.Config onto this, so the standalone
// st binary and a host pace and identify themselves the same way.
//
// There are no credentials. st reads only public, keyless Steam surfaces, so the
// config carries no token, no key, and no plane selection (see spec 8031
// section 10). The only knobs are the storefront locale (cc, lang, currency), the
// review filters, and the usual pacing, timeout, and cache settings.

const (
	// StoreHost is the storefront, the host the URI driver primarily claims.
	StoreHost = "store.steampowered.com"
	// CommunityHost serves public profiles (XML) and the community market.
	CommunityHost = "steamcommunity.com"
	// APIHost serves the keyless api.steampowered.com endpoints (app list, news,
	// player count, global achievement percentages).
	APIHost = "api.steampowered.com"

	// StoreURL, CommunityURL, and APIURL are the roots each request is built from.
	StoreURL     = "https://" + StoreHost
	CommunityURL = "https://" + CommunityHost
	APIURL       = "https://" + APIHost

	// DefaultCacheTTL is how long a cached response stays fresh by default. Store
	// data changes on a daily cadence, so a few hours is plenty.
	DefaultCacheTTL = 6 * time.Hour

	defaultLimit = 20 // a bare list command's fetch count
)

// DefaultUserAgent identifies the client. It names a current browser, because the
// storefront and the community site serve a leaner response to an obvious script
// and a few endpoints reject an empty agent. It is honest in that st does not
// forge a crawler identity it is reverse-DNS-checked against.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// Config is the resolved settings a Client reads.
type Config struct {
	UserAgent string
	Delay     time.Duration // minimum gap between requests
	Retries   int           // retries on 429/5xx
	Timeout   time.Duration // per-request timeout

	// Storefront locale, shared by the store commands.
	CC       string // country code, e.g. "us"; sets the price currency and availability
	Lang     string // language, e.g. "english"; sets the description and name language
	Currency int    // market currency code, e.g. 1 for USD

	// Review filters, used by the reviews command.
	ReviewFilter   string // recent, updated, all
	ReviewLanguage string // all, english, ...
	PurchaseType   string // all, steam, non_steam_purchase

	// Overridable for tests.
	StoreURL     string
	CommunityURL string
	APIURL       string

	CacheDir string
	NoCache  bool
	CacheTTL time.Duration
	Refresh  bool // refetch and rewrite the cache, ignoring any hit
}

// DefaultConfig returns the baseline settings. It reads no environment variable:
// st has no key and no per-flag environment fallback.
func DefaultConfig() Config {
	return Config{
		UserAgent:      DefaultUserAgent,
		Delay:          1 * time.Second,
		Retries:        3,
		Timeout:        30 * time.Second,
		CC:             "us",
		Lang:           "english",
		Currency:       1,
		ReviewFilter:   "recent",
		ReviewLanguage: "all",
		PurchaseType:   "all",
		StoreURL:       StoreURL,
		CommunityURL:   CommunityURL,
		APIURL:         APIURL,
		CacheTTL:       DefaultCacheTTL,
	}
}
