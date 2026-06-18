// Package cli assembles the st command tree from the steam domain on top of the
// any-cli/kit framework.
package cli

import (
	"strconv"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/steam-cli/steam"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// builder holds the domain-global flags while the app is assembled, then folds
// them onto the resolved config in finalize, using the exact keys ClientFromConfig
// reads. There is no token or key flag: st reads only keyless Steam surfaces.
type builder struct {
	userAgent      string
	cc             string
	lang           string
	currency       int
	reviewFilter   string
	reviewLanguage string
	purchaseType   string
	cacheTTL       string
	refresh        bool
}

// NewApp assembles the kit application from the steam domain. The domain's
// Register installs the client factory and every operation, so the binary and a
// host (which blank-imports the package) share one source of truth. This package
// adds the domain-global flags and the version command; kit.Run turns the App into
// the CLI, plus the serve and mcp surfaces and the typed-error-to-exit-code
// mapping.
//
// To add a command, declare it in steam/domain.go with kit.Handle and it appears
// here automatically. Reach for app.AddCommand only for a verb that does not fit
// the emit-records shape, the way version does below.
func NewApp() *kit.App {
	b := &builder{}
	id := steam.Identity()
	id.Version = Version

	app := kit.New(id, kit.WithDefaults(steam.Defaults))
	app.GlobalFlags(b.globals)
	app.Finalize(b.finalize)

	steam.Domain{}.Register(app)
	app.AddCommand(newVersionCmd())
	return app
}

func (b *builder) globals(f *kit.FlagSet) {
	def := steam.DefaultConfig()
	f.StringVar(&b.userAgent, "user-agent", steam.DefaultUserAgent, "User-Agent sent with each request")
	f.StringVar(&b.cc, "cc", def.CC, "storefront country code, e.g. us; sets price currency and availability")
	f.StringVar(&b.lang, "lang", def.Lang, "storefront language, e.g. english; sets description and name language")
	f.IntVar(&b.currency, "currency", def.Currency, "market currency code, e.g. 1 for USD")
	f.StringVar(&b.reviewFilter, "review-filter", def.ReviewFilter, "review order: recent, updated, or all")
	f.StringVar(&b.reviewLanguage, "review-language", def.ReviewLanguage, "review language: all, english, ...")
	f.StringVar(&b.purchaseType, "purchase-type", def.PurchaseType, "review purchase type: all, steam, or non_steam_purchase")
	f.StringVar(&b.cacheTTL, "cache-ttl", steam.DefaultCacheTTL.String(), "how long a cached response stays fresh")
	f.BoolVar(&b.refresh, "refresh", false, "fetch fresh copies and rewrite the cache, ignoring any hit")
}

func (b *builder) finalize(c *kit.Config) {
	if c.Extra == nil {
		c.Extra = map[string]string{}
	}
	set := func(k, v string) {
		if v != "" {
			c.Extra[k] = v
		}
	}
	set("user-agent", b.userAgent)
	set("cc", b.cc)
	set("lang", b.lang)
	if b.currency != 0 {
		c.Extra["currency"] = strconv.Itoa(b.currency)
	}
	set("review-filter", b.reviewFilter)
	set("review-language", b.reviewLanguage)
	set("purchase-type", b.purchaseType)
	set("cache-ttl", b.cacheTTL)
	if b.refresh {
		c.Extra["refresh"] = "true"
	}
}
