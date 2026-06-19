package steam

// types.go holds the exported records the commands emit. Their json tags name the
// fields a reader sees (lower case, the same names -o json prints and --template
// reads). kit:"id" marks the key the record store upserts on, kit:"body" marks the
// long-text field `st cat` and the Markdown export print, table:",truncate" keeps
// wide free text from blowing up a terminal table, and table:"-" hides a column
// from the default grid while keeping it in JSON.
//
// Each record carries only fields a keyless read can fill: no owner-only stat, no
// viewer state (whether you own the game, whether you follow the curator), no
// private profile field. A field the source did not fill is omitted, never guessed.
//
// The kit:"link" edges connect the records into one graph a host walks for
// breadth-first crawls; a slice-valued edge yields one edge per element. The
// numeric appid is the universal app key, so the store record, its reviews, its
// news, its player count, and its global achievements all point at the same app:
//
//	search/featured/applist --(App)--> app
//	app   --dlc_refs-->> app           app --fullgame_ref--> app
//	app   --package_refs-->> package   app --reviews_ref--> reviews --author_ref--> profile
//	app   --news_ref--> news           app --(players)--> player count
//	package --app_refs-->> app         review --author_ref--> profile
//	profile --most_played_refs-->> app market --(MarketItem)--> app
//
// A crawler that starts at a search reaches apps, from an app reaches DLC, the base
// game, packages, reviews, news, the player count, and achievements, from a review
// reaches the author profile, and from a profile reaches the most-played apps, so
// the walk continues over the public estate, all keyless. No record is a dead leaf.

// App is the core record: a store entry (a game, DLC, demo, music, video, or
// software). The id is the numeric appid, the key the store, the reviews, the news,
// the player count, and the achievements all share.
type App struct {
	ID                  string     `json:"id" kit:"id"` // the numeric appid
	Name                string     `json:"name,omitempty" table:",truncate"`
	Type                string     `json:"type,omitempty"` // game, dlc, demo, music, video, ...
	IsFree              bool       `json:"is_free,omitempty"`
	ShortDescription    string     `json:"short_description,omitempty" table:",truncate"`
	DetailedDescription string     `json:"detailed_description,omitempty" table:"-" kit:"body"`
	AboutTheGame        string     `json:"about_the_game,omitempty" table:"-"`
	SupportedLanguages  string     `json:"supported_languages,omitempty" table:"-"`
	Developers          []string   `json:"developers,omitempty" table:"-"`
	Publishers          []string   `json:"publishers,omitempty" table:"-"`
	Price               *Price     `json:"price,omitempty" table:"-"`
	Platforms           *Platforms `json:"platforms,omitempty" table:"-"`
	Categories          []IDName   `json:"categories,omitempty" table:"-"`
	Genres              []IDName   `json:"genres,omitempty" table:"-"`
	ReleaseDate         string     `json:"release_date,omitempty"`
	ComingSoon          bool       `json:"coming_soon,omitempty" table:"-"`
	RequiredAge         int        `json:"required_age,omitempty" table:"-"`
	ControllerSupport   string     `json:"controller_support,omitempty" table:"-"`
	Metacritic          int        `json:"metacritic,omitempty" table:"-"`
	MetacriticURL       string     `json:"metacritic_url,omitempty" table:"-"`
	Recommendations     int        `json:"recommendations,omitempty" table:"-"`
	AchievementsTotal   int        `json:"achievements_total,omitempty" table:"-"`
	ReviewScore         string     `json:"review_score,omitempty" table:"-"` // schema.org rating from the store-page island
	Website             string     `json:"website,omitempty" table:"-"`
	HeaderImage         string     `json:"header_image,omitempty" table:"-"`
	CapsuleImage        string     `json:"capsule_image,omitempty" table:"-"`
	CapsuleImageV5      string     `json:"capsule_image_v5,omitempty" table:"-"`
	Background          string     `json:"background,omitempty" table:"-"`
	BackgroundRaw       string     `json:"background_raw,omitempty" table:"-"`
	Screenshots         []Media    `json:"screenshots,omitempty" table:"-"`
	Movies              []Media    `json:"movies,omitempty" table:"-"`
	Tags                []string   `json:"tags,omitempty" table:"-"` // user tags from the store-page island
	ContentDescriptors  []string   `json:"content_descriptors,omitempty" table:"-"`
	Ratings             []Rating   `json:"ratings,omitempty" table:"-"` // per-board content ratings
	SupportURL          string     `json:"support_url,omitempty" table:"-"`
	SupportEmail        string     `json:"support_email,omitempty" table:"-"`
	LegalNotice         string     `json:"legal_notice,omitempty" table:"-"`
	DRMNotice           string     `json:"drm_notice,omitempty" table:"-"`
	ExtUserAccount      string     `json:"ext_user_account_notice,omitempty" table:"-"`

	// System requirements per platform (the source carries them as HTML).
	PCRequirements    *Requirements `json:"pc_requirements,omitempty" table:"-"`
	MacRequirements   *Requirements `json:"mac_requirements,omitempty" table:"-"`
	LinuxRequirements *Requirements `json:"linux_requirements,omitempty" table:"-"`

	// Review summary from the appreviews query_summary (folded in best-effort).
	ReviewScoreDesc string `json:"review_score_desc,omitempty" table:"-"` // e.g. "Overwhelmingly Positive"
	TotalReviews    int    `json:"total_reviews,omitempty" table:"-"`
	TotalPositive   int    `json:"total_positive,omitempty" table:"-"`
	TotalNegative   int    `json:"total_negative,omitempty" table:"-"`

	// Embedded relations the source fills inline.
	Fullgame                *GameLink         `json:"fullgame,omitempty" table:"-"` // set when this app is DLC or a demo
	DLC                     []GameLink        `json:"dlc,omitempty" table:"-"`
	Demos                   []GameLink        `json:"demos,omitempty" table:"-"`
	Packages                []int             `json:"packages,omitempty" table:"-"`
	BuyOptions              []BuyOption       `json:"buy_options,omitempty" table:"-"` // from package_groups
	HighlightedAchievements []AchievementInfo `json:"highlighted_achievements,omitempty" table:"-"`

	URL string `json:"url"` // the store page

	// Graph edges (one per slice element).
	DLCRefs     []string `json:"dlc_refs,omitempty" table:"-" kit:"link,kind=steam/app"`
	DemoRefs    []string `json:"demo_refs,omitempty" table:"-" kit:"link,kind=steam/app"`
	FullgameRef string   `json:"fullgame_ref,omitempty" table:"-" kit:"link,kind=steam/app"`
	PackageRefs []string `json:"package_refs,omitempty" table:"-" kit:"link,kind=steam/package"`
	ReviewsRef  string   `json:"reviews_ref,omitempty" table:"-" kit:"link,kind=steam/reviews"` // = ID
	NewsRef     string   `json:"news_ref,omitempty" table:"-" kit:"link,kind=steam/news"`       // = ID
}

// Review is one user review of an app, emitted by reviews. App is the edge back to
// the app; AuthorRef is the edge to the author's public profile.
type Review struct {
	ID               string  `json:"id" kit:"id"` // recommendationid
	App              string  `json:"app,omitempty" table:"-" kit:"link,kind=steam/app"`
	Author           string  `json:"author,omitempty"` // author SteamID64
	Language         string  `json:"language,omitempty" table:"-"`
	Body             string  `json:"body,omitempty" table:",truncate" kit:"body"`
	VotedUp          bool    `json:"voted_up,omitempty"`
	VotesUp          int     `json:"votes_up,omitempty"`
	VotesFunny       int     `json:"votes_funny,omitempty" table:"-"`
	WeightedScore    float64 `json:"weighted_score,omitempty" table:"-"`
	Comments         int     `json:"comments,omitempty" table:"-"`
	SteamPurchase    bool    `json:"steam_purchase,omitempty" table:"-"`
	ReceivedFree     bool    `json:"received_free,omitempty" table:"-"`
	EarlyAccess      bool    `json:"early_access,omitempty" table:"-"`
	PlaytimeAtReview int     `json:"playtime_at_review,omitempty" table:"-"` // minutes
	PlaytimeForever  int     `json:"playtime_forever,omitempty" table:"-"`   // minutes
	Created          string  `json:"created,omitempty"`
	Updated          string  `json:"updated,omitempty" table:"-"`
	AuthorRef        string  `json:"author_ref,omitempty" table:"-" kit:"link,kind=steam/profile"`
}

// Package is a store package (a sub): a bundle of apps sold as one purchase,
// emitted by package. AppRefs is the apps it bundles, each a walkable app edge.
type Package struct {
	ID          string     `json:"id" kit:"id"` // packageid
	Name        string     `json:"name,omitempty" table:",truncate"`
	Price       *Price     `json:"price,omitempty" table:"-"`
	Platforms   *Platforms `json:"platforms,omitempty" table:"-"`
	Controller  string     `json:"controller,omitempty" table:"-"` // e.g. full_gamepad
	ReleaseDate string     `json:"release_date,omitempty"`
	ComingSoon  bool       `json:"coming_soon,omitempty" table:"-"`
	PageImage   string     `json:"page_image,omitempty" table:"-"`
	SmallLogo   string     `json:"small_logo,omitempty" table:"-"`
	Apps        []GameLink `json:"apps,omitempty" table:"-"`
	URL         string     `json:"url"`
	AppRefs     []string   `json:"app_refs,omitempty" table:"-" kit:"link,kind=steam/app"`
}

// NewsItem is one news or announcement item for an app, emitted by news.
type NewsItem struct {
	ID        string `json:"id" kit:"id"` // gid
	App       string `json:"app,omitempty" table:"-" kit:"link,kind=steam/app"`
	Title     string `json:"title,omitempty" table:",truncate"`
	URL       string `json:"url,omitempty" table:"-"`
	Author    string `json:"author,omitempty" table:"-"`
	Body      string `json:"body,omitempty" table:",truncate" kit:"body"`
	FeedLabel string `json:"feed_label,omitempty" table:"-"`
	FeedName  string `json:"feed_name,omitempty" table:"-"`
	FeedType  int    `json:"feed_type,omitempty" table:"-"`
	External  bool   `json:"external,omitempty" table:"-"`
	Date      string `json:"date,omitempty"`
}

// PlayerCount is the live concurrent player count for an app, emitted by players.
type PlayerCount struct {
	ID    string `json:"id" kit:"id"` // appid
	Count int    `json:"count"`
	URL   string `json:"url"`
	App   string `json:"app,omitempty" table:"-" kit:"link,kind=steam/app"`
}

// Achievement is the global unlock rate of one achievement, emitted by
// achievements. The keyless percentages endpoint gives only the api name and the
// percent, so st does not invent a display name or an icon it cannot read.
type Achievement struct {
	ID      string  `json:"id" kit:"id"` // the achievement api name
	App     string  `json:"app,omitempty" table:"-" kit:"link,kind=steam/app"`
	Percent float64 `json:"percent"` // global unlock rate
}

// Profile is a public community profile, parsed from the XML, emitted by profile.
// A private profile carries the public subset and sets Visibility accordingly.
type Profile struct {
	ID             string     `json:"id" kit:"id"` // SteamID64
	PersonaName    string     `json:"persona_name,omitempty" table:",truncate"`
	RealName       string     `json:"real_name,omitempty" table:"-"`
	CustomURL      string     `json:"custom_url,omitempty" table:"-"` // the vanity, if set
	Visibility     string     `json:"visibility,omitempty"`           // public, private, friendsonly
	OnlineState    string     `json:"online_state,omitempty"`         // online, offline, in-game
	StateMessage   string     `json:"state_message,omitempty" table:"-"`
	Location       string     `json:"location,omitempty" table:"-"`
	Summary        string     `json:"summary,omitempty" table:",truncate" kit:"body"`
	MemberSince    string     `json:"member_since,omitempty" table:"-"`
	Avatar         string     `json:"avatar,omitempty" table:"-"`
	AvatarFull     string     `json:"avatar_full,omitempty" table:"-"`
	VACBanned      bool       `json:"vac_banned,omitempty" table:"-"`
	TradeBanState  string     `json:"trade_ban_state,omitempty" table:"-"`
	LimitedAccount bool       `json:"limited_account,omitempty" table:"-"`
	HoursTwoWeeks  float64    `json:"hours_two_weeks,omitempty" table:"-"`
	MostPlayed     []GameLink `json:"most_played,omitempty" table:"-"`
	Groups         []string   `json:"groups,omitempty" table:"-"` // group ids
	URL            string     `json:"url"`
	MostPlayedRefs []string   `json:"most_played_refs,omitempty" table:"-" kit:"link,kind=steam/app"`
}

// MarketItem is one community market listing, emitted by market.
type MarketItem struct {
	ID            string `json:"id" kit:"id"` // market hash name
	Name          string `json:"name,omitempty" table:",truncate"`
	App           string `json:"app,omitempty" table:"-" kit:"link,kind=steam/app"`
	AppName       string `json:"app_name,omitempty" table:",truncate"`
	SellListings  int    `json:"sell_listings,omitempty"`
	SellPrice     int    `json:"sell_price,omitempty"` // cents
	SellPriceText string `json:"sell_price_text,omitempty"`
	Type          string `json:"type,omitempty" table:"-"`
	Icon          string `json:"icon,omitempty" table:"-"`
	URL           string `json:"url"`
}

// MarketPrice is the lowest, median, and volume for one market item, emitted by
// price.
type MarketPrice struct {
	ID          string `json:"id" kit:"id"` // market hash name
	App         string `json:"app,omitempty" table:"-" kit:"link,kind=steam/app"`
	Currency    int    `json:"currency,omitempty"`
	LowestPrice string `json:"lowest_price,omitempty"`
	MedianPrice string `json:"median_price,omitempty"`
	Volume      string `json:"volume,omitempty"`
	URL         string `json:"url"`
}

// SteamID is the offline conversion result of `st resolve` and `st ref steamid`.
type SteamID struct {
	Input     string `json:"input"`
	Kind      string `json:"kind"` // steamid64, steamid3, steam2, vanity
	SteamID64 string `json:"steamid64,omitempty"`
	SteamID3  string `json:"steamid3,omitempty"`
	Steam2    string `json:"steam2,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Vanity    string `json:"vanity,omitempty"`
	URL       string `json:"url,omitempty"`
}

// Ref is the result of `st ref id`/`st ref url`: the canonical (kind, id) a
// reference resolves to, plus the URL, all without touching the network.
type Ref struct {
	Input string `json:"input"`
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	URL   string `json:"url"`
}

// --- embedded value types ---

// Price is an app or package price. The amounts are in the currency's minor unit
// (cents for USD).
type Price struct {
	Currency         string `json:"currency,omitempty"`
	Initial          int    `json:"initial,omitempty"`
	Final            int    `json:"final,omitempty"`
	Individual       int    `json:"individual,omitempty"` // packages: the unbundled total
	DiscountPct      int    `json:"discount_pct,omitempty"`
	InitialFormatted string `json:"initial_formatted,omitempty"`
	FinalFormatted   string `json:"final_formatted,omitempty"`
}

// Requirements is the system requirements for one platform, as the store returns
// them (HTML fragments, kept verbatim).
type Requirements struct {
	Minimum     string `json:"minimum,omitempty"`
	Recommended string `json:"recommended,omitempty"`
}

// Rating is one content-rating board's verdict for an app (USK, DEJUS, ESRB,
// PEGI, steam_germany, igrs, and others), keyed by the board name.
type Rating struct {
	Board       string `json:"board"` // usk, dejus, esrb, pegi, steam_germany, igrs, ...
	Rating      string `json:"rating,omitempty"`
	RequiredAge string `json:"required_age,omitempty"`
	Descriptors string `json:"descriptors,omitempty"`
	Banned      string `json:"banned,omitempty"`
	UseAgeGate  string `json:"use_age_gate,omitempty"`
}

// BuyOption is one purchase option from an app's package_groups: a sub a reader
// can buy, with its price and any discount.
type BuyOption struct {
	PackageID  string `json:"package_id"`
	Text       string `json:"text,omitempty"`        // the human option label
	PriceCents int    `json:"price_cents,omitempty"` // price with the current discount
	SavingsPct int    `json:"savings_pct,omitempty"`
	IsFree     bool   `json:"is_free,omitempty"`
}

// AchievementInfo is one highlighted achievement's display name and icon, folded
// in from appdetails. The percentages endpoint gives only api names, so this is
// the only keyless source of an achievement's localized name and icon.
type AchievementInfo struct {
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"` // the full icon URL
}

// CrawlNode is one record visited by a breadth-first walk of the public graph,
// emitted by crawl. The id is a "kind:ref" composite so nodes of different kinds
// never collide; edges lists the "kind:ref" neighbors the walk found.
type CrawlNode struct {
	ID     string   `json:"id" kit:"id"` // "kind:ref"
	Kind   string   `json:"kind"`
	Ref    string   `json:"ref"`
	Name   string   `json:"name,omitempty" table:",truncate"`
	Depth  int      `json:"depth"`
	Parent string   `json:"parent,omitempty" table:"-"`
	URL    string   `json:"url"`
	Edges  []string `json:"edges,omitempty" table:"-"`
}

// Platforms is the set of operating systems an app or package supports.
type Platforms struct {
	Windows bool `json:"windows,omitempty"`
	Mac     bool `json:"mac,omitempty"`
	Linux   bool `json:"linux,omitempty"`
}

// IDName is a store facet: a category or a genre, with its id and label.
type IDName struct {
	ID          string `json:"id,omitempty"`
	Description string `json:"description,omitempty"`
}

// GameLink is the embedded reference to another app: a base game, a DLC entry, a
// bundled app, or a most-played game.
type GameLink struct {
	AppID string `json:"appid,omitempty"`
	Name  string `json:"name,omitempty"`
}

// Media is one screenshot or movie in an app's gallery.
type Media struct {
	ID    string `json:"id,omitempty"`
	Type  string `json:"type,omitempty"` // screenshot, movie
	Thumb string `json:"thumb,omitempty"`
	Full  string `json:"full,omitempty"` // the full image, or the mp4/webm for a movie
}
