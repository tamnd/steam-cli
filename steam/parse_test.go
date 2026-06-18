package steam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// routerServer dispatches a request to the handler whose path prefix matches,
// standing in for the store, api, and community hosts at once. A path with no
// match is a 404.
func routerServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for prefix, body := range routes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				_, _ = w.Write([]byte(body))
				return
			}
		}
		http.NotFound(w, r)
	}))
}

const appDetailsFixture = `{"620":{"success":true,"data":{
	"type":"game","name":"Portal 2","steam_appid":620,"required_age":"0","is_free":false,
	"dlc":[323180],"packages":[7877,204528],
	"short_description":"co-op puzzles","detailed_description":"<p>long</p>",
	"developers":["Valve"],"publishers":["Valve"],
	"price_overview":{"currency":"USD","initial":999,"final":999,"discount_percent":0,"final_formatted":"$9.99"},
	"platforms":{"windows":true,"mac":false,"linux":true},
	"categories":[{"id":2,"description":"Single-player"}],
	"genres":[{"id":"1","description":"Action"}],
	"release_date":{"coming_soon":false,"date":"21 Apr, 2011"},
	"fullgame":{"appid":"400","name":"Portal"}
}}}`

const appPageFixture = `<html><head>
<script type="application/ld+json">{"@type":"BreadcrumbList"}</script>
<script type="application/ld+json">{"@type":"VideoGame","aggregateRating":{"ratingValue":"9","reviewCount":"100"}}</script>
</head><body>
<script>InitAppTagModal( 620, [{"tagid":1,"name":"Puzzle"},{"tagid":2,"name":"Co-op"}], 'foo' );</script>
</body></html>`

func TestAppParse(t *testing.T) {
	srv := routerServer(t, map[string]string{
		"/api/appdetails": appDetailsFixture,
		"/app/620":        appPageFixture,
	})
	defer srv.Close()

	a, err := testClient(srv.URL).App(context.Background(), "620")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "620" || a.Name != "Portal 2" || a.Type != "game" {
		t.Errorf("core fields wrong: %+v", a)
	}
	if a.Price == nil || a.Price.FinalFormatted != "$9.99" {
		t.Errorf("price wrong: %+v", a.Price)
	}
	if a.Platforms == nil || !a.Platforms.Windows || a.Platforms.Mac || !a.Platforms.Linux {
		t.Errorf("platforms wrong: %+v", a.Platforms)
	}
	if len(a.Categories) != 1 || a.Categories[0].ID != "2" {
		t.Errorf("categories wrong: %+v", a.Categories)
	}
	if len(a.Genres) != 1 || a.Genres[0].ID != "1" {
		t.Errorf("genres wrong: %+v", a.Genres)
	}
	// Edges.
	if len(a.DLCRefs) != 1 || a.DLCRefs[0] != "323180" {
		t.Errorf("DLCRefs = %v", a.DLCRefs)
	}
	if a.FullgameRef != "400" {
		t.Errorf("FullgameRef = %q, want 400", a.FullgameRef)
	}
	if len(a.PackageRefs) != 2 || a.PackageRefs[0] != "7877" {
		t.Errorf("PackageRefs = %v", a.PackageRefs)
	}
	if a.ReviewsRef != "620" || a.NewsRef != "620" {
		t.Errorf("section edges wrong: reviews=%q news=%q", a.ReviewsRef, a.NewsRef)
	}
	// Store-page island enrichment.
	if a.ReviewScore != "9 from 100 reviews" {
		t.Errorf("ReviewScore = %q (ld+json island)", a.ReviewScore)
	}
	if len(a.Tags) != 2 || a.Tags[0] != "Puzzle" {
		t.Errorf("Tags = %v (tag island)", a.Tags)
	}
}

func TestAppNotFound(t *testing.T) {
	srv := routerServer(t, map[string]string{
		"/api/appdetails": `{"99":{"success":false}}`,
	})
	defer srv.Close()
	if _, err := testClient(srv.URL).App(context.Background(), "99"); err != ErrNotFound {
		t.Errorf("App(missing) = %v, want ErrNotFound", err)
	}
}

func TestSearchParse(t *testing.T) {
	const fixture = `{"total":2,"items":[
		{"type":"app","name":"Portal 2","id":620,"tiny_image":"img","metascore":"95","platforms":{"windows":true},"price":{"currency":"USD","initial":999,"final":999}},
		{"type":"app","name":"Portal","id":400,"price":{"currency":"USD","initial":0,"final":0}}
	]}`
	srv := routerServer(t, map[string]string{"/api/storesearch": fixture})
	defer srv.Close()

	items, err := testClient(srv.URL).Search(context.Background(), "portal", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].ID != "620" || items[0].Metacritic != 95 {
		t.Errorf("first item wrong: %+v", items[0])
	}
	if !items[1].IsFree {
		t.Errorf("free item not marked free: %+v", items[1])
	}
	if items[0].ReviewsRef != "620" {
		t.Errorf("search hit carries no app edge: %+v", items[0])
	}
}

func TestReviewsCursorStops(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			// First page advances the cursor.
			_, _ = w.Write([]byte(`{"success":1,"cursor":"AoJw","reviews":[
				{"recommendationid":"1","author":{"steamid":"765","playtime_forever":100,"playtime_at_review":50},"language":"english","review":"good","voted_up":true,"votes_up":3,"weighted_vote_score":"0.5","timestamp_created":1600000000}
			]}`))
			return
		}
		// Second page repeats the cursor, which must stop the loop.
		_, _ = w.Write([]byte(`{"success":1,"cursor":"AoJw","reviews":[
			{"recommendationid":"2","author":{"steamid":"766"},"language":"english","review":"ok","voted_up":false,"weighted_vote_score":"0"}
		]}`))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	revs, err := c.Reviews(context.Background(), "620", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 2 {
		t.Fatalf("got %d reviews, want 2 (one per page before the cursor repeats)", len(revs))
	}
	if hits != 2 {
		t.Errorf("server saw %d hits, want 2 (the repeated cursor stops paging)", hits)
	}
	r := revs[0]
	if r.ID != "1" || r.Author != "765" || r.AuthorRef != "765" {
		t.Errorf("review fields wrong: %+v", r)
	}
	if r.WeightedScore != 0.5 {
		t.Errorf("WeightedScore = %v, want 0.5 (quoted float)", r.WeightedScore)
	}
	if r.Created != "2020-09-13T12:26:40Z" {
		t.Errorf("Created = %q, want RFC3339 UTC", r.Created)
	}
	if r.App != "620" {
		t.Errorf("review carries no app edge: %+v", r)
	}
}

func TestReviewsNotFound(t *testing.T) {
	srv := routerServer(t, map[string]string{"/appreviews": `{"success":2}`})
	defer srv.Close()
	if _, err := testClient(srv.URL).Reviews(context.Background(), "620", 5); err != ErrNotFound {
		t.Errorf("Reviews(success!=1) = %v, want ErrNotFound", err)
	}
}

func TestPackageParse(t *testing.T) {
	const fixture = `{"7877":{"success":true,"data":{
		"name":"Portal 2","page_content":"bundle","header_image":"img",
		"price":{"currency":"USD","initial":999,"final":999,"discount_percent":0},
		"platforms":{"windows":true,"linux":true},
		"release_date":{"coming_soon":false,"date":"21 Apr, 2011"},
		"apps":[{"id":620,"name":"Portal 2"}]
	}}}`
	srv := routerServer(t, map[string]string{"/api/packagedetails": fixture})
	defer srv.Close()

	p, err := testClient(srv.URL).Package(context.Background(), "7877")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "7877" || p.Name != "Portal 2" {
		t.Errorf("package fields wrong: %+v", p)
	}
	if len(p.AppRefs) != 1 || p.AppRefs[0] != "620" {
		t.Errorf("AppRefs = %v, want [620]", p.AppRefs)
	}
}

func TestNewsParse(t *testing.T) {
	const fixture = `{"appnews":{"appid":620,"newsitems":[
		{"gid":"55","title":"Update","url":"https://x","is_external_url":false,"author":"valve","contents":"notes","feedlabel":"Community","date":1600000000,"feedname":"steam_community"}
	]}}`
	srv := routerServer(t, map[string]string{"/ISteamNews": fixture})
	defer srv.Close()

	items, err := testClient(srv.URL).News(context.Background(), "620", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "55" || items[0].App != "620" {
		t.Errorf("news wrong: %+v", items)
	}
	if items[0].Date != "2020-09-13T12:26:40Z" {
		t.Errorf("Date = %q, want RFC3339 UTC", items[0].Date)
	}
}

func TestPlayersParse(t *testing.T) {
	srv := routerServer(t, map[string]string{
		"/ISteamUserStats/GetNumberOfCurrentPlayers": `{"response":{"player_count":916,"result":1}}`,
	})
	defer srv.Close()
	pc, err := testClient(srv.URL).Players(context.Background(), "620")
	if err != nil {
		t.Fatal(err)
	}
	if pc.Count != 916 || pc.App != "620" {
		t.Errorf("player count wrong: %+v", pc)
	}
}

func TestPlayersNotFound(t *testing.T) {
	srv := routerServer(t, map[string]string{
		"/ISteamUserStats/GetNumberOfCurrentPlayers": `{"response":{"result":42}}`,
	})
	defer srv.Close()
	if _, err := testClient(srv.URL).Players(context.Background(), "620"); err != ErrNotFound {
		t.Errorf("Players(result!=1) = %v, want ErrNotFound", err)
	}
}

func TestAchievementsParse(t *testing.T) {
	srv := routerServer(t, map[string]string{
		"/ISteamUserStats/GetGlobalAchievementPercentagesForApp": `{"achievementpercentages":{"achievements":[
			{"name":"ACH.A","percent":"74.3"},
			{"name":"ACH.B","percent":"64.9"}
		]}}`,
	})
	defer srv.Close()
	items, err := testClient(srv.URL).Achievements(context.Background(), "620", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d achievements, want 2", len(items))
	}
	if items[0].ID != "ACH.A" || items[0].Percent != 74.3 || items[0].App != "620" {
		t.Errorf("achievement wrong (quoted percent?): %+v", items[0])
	}
}

func TestFeaturedParse(t *testing.T) {
	// A real payload mixes scalar status fields with category objects; the scalars
	// must be skipped and the apps de-duplicated across categories.
	const fixture = `{
		"status":1,
		"specials":{"items":[{"id":620,"name":"Portal 2","currency":"USD","final_price":499,"windows_available":true}]},
		"top_sellers":{"items":[{"id":620,"name":"Portal 2 dup"},{"id":400,"name":"Portal"}]}
	}`
	srv := routerServer(t, map[string]string{"/api/featuredcategories": fixture})
	defer srv.Close()

	items, err := testClient(srv.URL).Featured(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, it := range items {
		if ids[it.ID] {
			t.Errorf("duplicate app %q in featured output", it.ID)
		}
		ids[it.ID] = true
	}
	if !ids["620"] || !ids["400"] {
		t.Errorf("featured ids = %v, want 620 and 400", ids)
	}
}

func TestFeaturedCategoryParse(t *testing.T) {
	// A named slice returns only its own category, with no cross-category dedup.
	const fixture = `{
		"status":1,
		"specials":{"items":[{"id":620,"name":"Portal 2"}]},
		"top_sellers":{"items":[{"id":620,"name":"Portal 2"},{"id":400,"name":"Portal"}]}
	}`
	srv := routerServer(t, map[string]string{"/api/featuredcategories": fixture})
	defer srv.Close()

	c := testClient(srv.URL)
	top, err := c.FeaturedCategory(context.Background(), "top_sellers", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 2 || top[0].ID != "620" || top[1].ID != "400" {
		t.Errorf("top_sellers = %v, want [620 400] in order", top)
	}

	none, err := c.FeaturedCategory(context.Background(), "coming_soon", 0)
	if err != nil {
		t.Fatalf("missing category should not error: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("missing category = %v, want empty", none)
	}
}

const profileFixture = `<?xml version="1.0" encoding="UTF-8"?>
<profile>
	<steamID64>76561197960287930</steamID64>
	<steamID><![CDATA[Rabscuttle]]></steamID>
	<onlineState>online</onlineState>
	<privacyState>public</privacyState>
	<visibilityState>3</visibilityState>
	<customURL>gabelogannewell</customURL>
	<vacBanned>0</vacBanned>
	<memberSince>September 23, 2003</memberSince>
	<hoursPlayed2Wk>12.5</hoursPlayed2Wk>
	<mostPlayedGames>
		<mostPlayedGame>
			<gameName>Portal 2</gameName>
			<gameLink>https://steamcommunity.com/app/620</gameLink>
			<hoursOnRecord>100.0</hoursOnRecord>
		</mostPlayedGame>
	</mostPlayedGames>
</profile>`

func TestProfileParse(t *testing.T) {
	srv := routerServer(t, map[string]string{"/id/": profileFixture})
	defer srv.Close()

	p, err := testClient(srv.URL).Profile(context.Background(), "gabelogannewell")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "76561197960287930" || p.PersonaName != "Rabscuttle" {
		t.Errorf("profile fields wrong: %+v", p)
	}
	if p.Visibility != "public" {
		t.Errorf("Visibility = %q, want public", p.Visibility)
	}
	if p.HoursTwoWeeks != 12.5 {
		t.Errorf("HoursTwoWeeks = %v, want 12.5", p.HoursTwoWeeks)
	}
	if len(p.MostPlayedRefs) != 1 || p.MostPlayedRefs[0] != "620" {
		t.Errorf("MostPlayedRefs = %v, want [620]", p.MostPlayedRefs)
	}
}

func TestProfileNotFound(t *testing.T) {
	srv := routerServer(t, map[string]string{
		"/id/": `<?xml version="1.0"?><response><error>The specified profile could not be found.</error></response>`,
	})
	defer srv.Close()
	if _, err := testClient(srv.URL).Profile(context.Background(), "nobody"); err != ErrNotFound {
		t.Errorf("Profile(missing) = %v, want ErrNotFound", err)
	}
}

func TestResolveVanity(t *testing.T) {
	srv := routerServer(t, map[string]string{"/id/": profileFixture})
	defer srv.Close()
	id, err := testClient(srv.URL).Resolve(context.Background(), "gabelogannewell")
	if err != nil {
		t.Fatal(err)
	}
	if id.SteamID64 != "76561197960287930" || id.Steam2 != "STEAM_1:0:11101" {
		t.Errorf("resolve wrong: %+v", id)
	}
}

func TestMarketSearchParse(t *testing.T) {
	const fixture = `{"success":true,"total_count":1,"results":[
		{"name":"AK-47 | Redline","hash_name":"AK-47 | Redline (Field-Tested)","sell_listings":1200,"sell_price":510,"sell_price_text":"$5.10","app_name":"CS2","asset_description":{"appid":730,"type":"Rifle","market_hash_name":"AK-47 | Redline (Field-Tested)"}}
	]}`
	srv := routerServer(t, map[string]string{"/market/search/render": fixture})
	defer srv.Close()

	items, err := testClient(srv.URL).MarketSearch(context.Background(), "AK-47", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].App != "730" || items[0].SellPrice != 510 {
		t.Errorf("market item wrong: %+v", items[0])
	}
}

func TestMarketPriceParse(t *testing.T) {
	srv := routerServer(t, map[string]string{
		"/market/priceoverview": `{"success":true,"lowest_price":"$5.10","median_price":"$5.25","volume":"1,234"}`,
	})
	defer srv.Close()
	p, err := testClient(srv.URL).MarketPrice(context.Background(), "730", "AK-47 | Redline (Field-Tested)")
	if err != nil {
		t.Fatal(err)
	}
	if p.LowestPrice != "$5.10" || p.App != "730" {
		t.Errorf("price wrong: %+v", p)
	}
}

func TestMarketPriceNotFound(t *testing.T) {
	srv := routerServer(t, map[string]string{"/market/priceoverview": `{"success":false}`})
	defer srv.Close()
	if _, err := testClient(srv.URL).MarketPrice(context.Background(), "730", "nope"); err != ErrNotFound {
		t.Errorf("MarketPrice(missing) = %v, want ErrNotFound", err)
	}
}
