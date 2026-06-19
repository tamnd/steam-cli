package steam

import (
	"context"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// browse.go reads store.steampowered.com/search/results, the paginated catalog the
// store's own search page calls as it scrolls. Unlike the storesearch quick box
// (capped at ten hits), this endpoint walks the whole catalog: it reports a
// total_count and returns a page of rendered rows for a start offset, so Browse
// pages through it until it has the requested number of apps. Each row carries an
// appid, so every hit is an addressable app a reader or a crawl can follow to the
// full record. This is the discovery seed the graph walk starts from.

// BrowseOpts are the catalog filters Browse passes through to the endpoint.
type BrowseOpts struct {
	Query    string // free-text term, empty for the whole catalog
	Sort     string // sort_by, e.g. Released_DESC, Reviews_DESC, Price_ASC, Name_ASC
	MaxPrice string // maxprice filter, e.g. "free", "5", "10"
	Start    int    // the first result offset
	Limit    int    // how many apps to return in total
}

type catalogResponse struct {
	Success     int    `json:"success"`
	ResultsHTML string `json:"results_html"`
	TotalCount  int    `json:"total_count"`
	Start       int    `json:"start"`
}

// Browse returns up to opts.Limit catalog apps, paging through the search-results
// endpoint from opts.Start in pages of up to 100 rows.
func (c *Client) Browse(ctx context.Context, opts BrowseOpts) ([]*App, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	start := opts.Start
	if start < 0 {
		start = 0
	}
	var out []*App
	seen := map[string]bool{}
	for {
		count := limit - len(out)
		if count > 100 {
			count = 100
		}
		v := url.Values{}
		v.Set("query", "")
		if opts.Query != "" {
			v.Set("term", opts.Query)
		}
		if opts.Sort != "" {
			v.Set("sort_by", opts.Sort)
		}
		if opts.MaxPrice != "" {
			v.Set("maxprice", opts.MaxPrice)
		}
		v.Set("start", strconv.Itoa(start))
		v.Set("count", strconv.Itoa(count))
		v.Set("cc", c.cfg.CC)
		v.Set("l", c.cfg.Lang)
		v.Set("infinite", "1")
		v.Set("json", "1")
		u := c.cfg.StoreURL + "/search/results/?" + v.Encode()

		var resp catalogResponse
		if err := c.getJSON(ctx, u, &resp); err != nil {
			return nil, err
		}
		rows := parseCatalogRows(resp.ResultsHTML)
		if len(rows) == 0 {
			break
		}
		for i := range rows {
			r := &rows[i]
			if seen[r.ID] {
				continue
			}
			seen[r.ID] = true
			out = append(out, r)
			if len(out) >= limit {
				return out, nil
			}
		}
		start += len(rows)
		if resp.TotalCount > 0 && start >= resp.TotalCount {
			break
		}
	}
	return out, nil
}

var (
	catalogRowRE   = regexp.MustCompile(`(?s)<a\b([^>]*?)>(.*?)</a>`)
	catalogAppidRE = regexp.MustCompile(`data-ds-appid="([\d,]+)"`)
	catalogHrefRE  = regexp.MustCompile(`href="([^"]*)"`)
	catalogTitleRE = regexp.MustCompile(`<span class="title">([^<]*)</span>`)
	catalogRelRE   = regexp.MustCompile(`class="[^"]*\bsearch_released\b[^"]*">([^<]*)</div>`)
	catalogPriceRE = regexp.MustCompile(`data-price-final="(\d+)"`)
	catalogRevRE   = regexp.MustCompile(`data-tooltip-html="([^"]*)"`)
)

// parseCatalogRows extracts one App per search_result_row anchor in the rendered
// HTML. A row carries an appid (the first when an anchor lists several for a
// bundle), the title, the release date, the price in cents, and the review tooltip.
func parseCatalogRows(htmlStr string) []App {
	var out []App
	for _, m := range catalogRowRE.FindAllStringSubmatch(htmlStr, -1) {
		attrs, inner := m[1], m[2]
		am := catalogAppidRE.FindStringSubmatch(attrs)
		if am == nil {
			continue
		}
		id := am[1]
		if i := strings.IndexByte(id, ','); i >= 0 {
			id = id[:i] // an anchor may list several ids; the first is the row's app
		}
		if id == "" || id == "0" {
			continue
		}
		app := App{
			ID:         id,
			URL:        StoreURL + "/app/" + id,
			ReviewsRef: id,
			NewsRef:    id,
		}
		if t := catalogTitleRE.FindStringSubmatch(inner); t != nil {
			app.Name = html.UnescapeString(strings.TrimSpace(t[1]))
		}
		if h := catalogHrefRE.FindStringSubmatch(attrs); h != nil {
			if u := cleanStoreURL(h[1]); u != "" {
				app.URL = u
			}
		}
		if r := catalogRelRE.FindStringSubmatch(inner); r != nil {
			app.ReleaseDate = strings.TrimSpace(r[1])
		}
		if p := catalogPriceRE.FindStringSubmatch(inner); p != nil {
			if cents, err := strconv.Atoi(p[1]); err == nil {
				app.Price = &Price{Final: cents}
				app.IsFree = cents == 0
			}
		}
		if rv := catalogRevRE.FindStringSubmatch(inner); rv != nil {
			desc := html.UnescapeString(rv[1])
			if i := strings.Index(desc, "<br>"); i >= 0 {
				desc = desc[:i]
			}
			app.ReviewScoreDesc = strings.TrimSpace(desc)
		}
		out = append(out, app)
	}
	return out
}

// cleanStoreURL strips the tracking query a search row's href carries, leaving the
// bare store page URL.
func cleanStoreURL(raw string) string {
	if i := strings.IndexByte(raw, '?'); i >= 0 {
		raw = raw[:i]
	}
	return strings.TrimSpace(raw)
}
