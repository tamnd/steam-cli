package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// app.go reads one app from the storefront. The appdetails JSON endpoint is the
// system of record and fills almost the whole App record; the store HTML page adds
// two structured islands the JSON omits (the schema.org VideoGame in the ld+json
// block, and the user tag list), which enrichApp folds in best-effort. A walled or
// missing page leaves the record complete minus tags and review score, and the
// command still succeeds.

// appDetails is the wire shape of store.steampowered.com/api/appdetails.
type appDetailsEnvelope map[string]struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

type appData struct {
	Type                string          `json:"type"`
	Name                string          `json:"name"`
	SteamAppID          int             `json:"steam_appid"`
	RequiredAge         flexInt         `json:"required_age"`
	IsFree              bool            `json:"is_free"`
	DLC                 []int           `json:"dlc"`
	DetailedDescription string          `json:"detailed_description"`
	AboutTheGame        string          `json:"about_the_game"`
	ShortDescription    string          `json:"short_description"`
	SupportedLanguages  string          `json:"supported_languages"`
	HeaderImage         string          `json:"header_image"`
	Background          string          `json:"background"`
	Website             string          `json:"website"`
	Developers          []string        `json:"developers"`
	Publishers          []string        `json:"publishers"`
	ControllerSupport   string          `json:"controller_support"`
	PriceOverview       *priceOverview  `json:"price_overview"`
	Packages            []int           `json:"packages"`
	Platforms           *platformsWire  `json:"platforms"`
	Metacritic          *metacriticWire `json:"metacritic"`
	Categories          []idNameWire    `json:"categories"`
	Genres              []idNameWire    `json:"genres"`
	Screenshots         []screenshot    `json:"screenshots"`
	Movies              []movie         `json:"movies"`
	Recommendations     *struct {
		Total int `json:"total"`
	} `json:"recommendations"`
	Achievements *struct {
		Total int `json:"total"`
	} `json:"achievements"`
	ReleaseDate *struct {
		ComingSoon bool   `json:"coming_soon"`
		Date       string `json:"date"`
	} `json:"release_date"`
	SupportInfo *struct {
		URL   string `json:"url"`
		Email string `json:"email"`
	} `json:"support_info"`
	ContentDescriptors *struct {
		IDs []int `json:"ids"`
	} `json:"content_descriptors"`
	Fullgame *struct {
		AppID string `json:"appid"`
		Name  string `json:"name"`
	} `json:"fullgame"`
}

type priceOverview struct {
	Currency        string `json:"currency"`
	Initial         int    `json:"initial"`
	Final           int    `json:"final"`
	DiscountPercent int    `json:"discount_percent"`
	FinalFormatted  string `json:"final_formatted"`
}

type platformsWire struct {
	Windows bool `json:"windows"`
	Mac     bool `json:"mac"`
	Linux   bool `json:"linux"`
}

type metacriticWire struct {
	Score int    `json:"score"`
	URL   string `json:"url"`
}

// idNameWire carries a category or genre. The id is an int for categories and a
// string for genres, so flexString reads either.
type idNameWire struct {
	ID          flexString `json:"id"`
	Description string     `json:"description"`
}

type screenshot struct {
	ID            int    `json:"id"`
	PathThumbnail string `json:"path_thumbnail"`
	PathFull      string `json:"path_full"`
}

type movie struct {
	ID        int               `json:"id"`
	Name      string            `json:"name"`
	Thumbnail string            `json:"thumbnail"`
	Webm      map[string]string `json:"webm"`
	Mp4       map[string]string `json:"mp4"`
}

// App fetches one app by appid and returns it as a record. It reads the appdetails
// JSON, then enriches with the store-page island when the page is reachable.
func (c *Client) App(ctx context.Context, appid string) (*App, error) {
	appid = strings.TrimSpace(appid)
	if !numRE.MatchString(appid) {
		if r := Classify(appid); r.Kind == "app" {
			appid = r.ID
		} else {
			return nil, fmt.Errorf("%w: not an appid: %q", ErrUsage, appid)
		}
	}
	u := fmt.Sprintf("%s/api/appdetails?appids=%s&cc=%s&l=%s",
		c.cfg.StoreURL, appid, url.QueryEscape(c.cfg.CC), url.QueryEscape(c.cfg.Lang))
	var env appDetailsEnvelope
	if err := c.getJSON(ctx, u, &env); err != nil {
		return nil, err
	}
	entry, ok := env[appid]
	if !ok || !entry.Success {
		return nil, ErrNotFound
	}
	var d appData
	if err := json.Unmarshal(entry.Data, &d); err != nil {
		return nil, fmt.Errorf("decode appdetails data: %w", err)
	}
	app := toApp(&d, appid)
	c.enrichApp(ctx, app)
	return app, nil
}

// toApp maps the appdetails data onto an App and wires its edges.
func toApp(d *appData, appid string) *App {
	app := &App{
		ID:                  appid,
		Name:                d.Name,
		Type:                d.Type,
		IsFree:              d.IsFree,
		ShortDescription:    squish(d.ShortDescription),
		DetailedDescription: d.DetailedDescription,
		AboutTheGame:        d.AboutTheGame,
		SupportedLanguages:  squish(d.SupportedLanguages),
		Developers:          d.Developers,
		Publishers:          d.Publishers,
		RequiredAge:         int(d.RequiredAge),
		ControllerSupport:   d.ControllerSupport,
		Website:             d.Website,
		HeaderImage:         d.HeaderImage,
		Background:          d.Background,
		URL:                 StoreURL + "/app/" + appid,
		ReviewsRef:          appid,
		NewsRef:             appid,
	}
	if d.PriceOverview != nil {
		app.Price = &Price{
			Currency:       d.PriceOverview.Currency,
			Initial:        d.PriceOverview.Initial,
			Final:          d.PriceOverview.Final,
			DiscountPct:    d.PriceOverview.DiscountPercent,
			FinalFormatted: d.PriceOverview.FinalFormatted,
		}
	}
	if d.Platforms != nil {
		app.Platforms = &Platforms{Windows: d.Platforms.Windows, Mac: d.Platforms.Mac, Linux: d.Platforms.Linux}
	}
	if d.Metacritic != nil {
		app.Metacritic = d.Metacritic.Score
		app.MetacriticURL = d.Metacritic.URL
	}
	for _, cat := range d.Categories {
		app.Categories = append(app.Categories, IDName{ID: string(cat.ID), Description: cat.Description})
	}
	for _, g := range d.Genres {
		app.Genres = append(app.Genres, IDName{ID: string(g.ID), Description: g.Description})
	}
	if d.ReleaseDate != nil {
		app.ReleaseDate = d.ReleaseDate.Date
		app.ComingSoon = d.ReleaseDate.ComingSoon
	}
	if d.Recommendations != nil {
		app.Recommendations = d.Recommendations.Total
	}
	if d.Achievements != nil {
		app.AchievementsTotal = d.Achievements.Total
	}
	if d.SupportInfo != nil {
		app.SupportURL = d.SupportInfo.URL
		app.SupportEmail = d.SupportInfo.Email
	}
	if d.ContentDescriptors != nil {
		for _, id := range d.ContentDescriptors.IDs {
			app.ContentDescriptors = append(app.ContentDescriptors, strconv.Itoa(id))
		}
	}
	for _, s := range d.Screenshots {
		app.Screenshots = append(app.Screenshots, Media{
			ID: strconv.Itoa(s.ID), Type: "screenshot", Thumb: s.PathThumbnail, Full: s.PathFull,
		})
	}
	for _, m := range d.Movies {
		app.Movies = append(app.Movies, Media{
			ID: strconv.Itoa(m.ID), Type: "movie", Thumb: m.Thumbnail, Full: pickMax(m.Mp4),
		})
	}
	if d.Fullgame != nil && d.Fullgame.AppID != "" {
		app.Fullgame = &GameLink{AppID: d.Fullgame.AppID, Name: d.Fullgame.Name}
		app.FullgameRef = d.Fullgame.AppID
	}
	for _, id := range d.DLC {
		s := strconv.Itoa(id)
		app.DLC = append(app.DLC, GameLink{AppID: s})
		app.DLCRefs = append(app.DLCRefs, s)
	}
	for _, id := range d.Packages {
		app.Packages = append(app.Packages, id)
		app.PackageRefs = append(app.PackageRefs, strconv.Itoa(id))
	}
	return app
}

var (
	ldJSONRE   = regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`)
	storeTagRE = regexp.MustCompile(`InitAppTagModal\(\s*\d+\s*,\s*(\[.*?\])`)
)

// enrichApp folds the store-page islands into app, best-effort. A walled or
// missing page is not an error: the record is already complete from the JSON.
func (c *Client) enrichApp(ctx context.Context, app *App) {
	u := fmt.Sprintf("%s/app/%s/?cc=%s&l=%s",
		c.cfg.StoreURL, app.ID, url.QueryEscape(c.cfg.CC), url.QueryEscape(c.cfg.Lang))
	body, err := c.get(ctx, u)
	if err != nil {
		return
	}
	// The store page carries more than one ld+json island (a BreadcrumbList ahead
	// of the VideoGame), so scan them all for the one with an aggregateRating.
	for _, m := range ldJSONRE.FindAllSubmatch(body, -1) {
		var ld struct {
			AggregateRating *struct {
				RatingValue json.RawMessage `json:"ratingValue"`
				ReviewCount json.RawMessage `json:"reviewCount"`
			} `json:"aggregateRating"`
		}
		if json.Unmarshal(m[1], &ld) != nil || ld.AggregateRating == nil {
			continue
		}
		rv := strings.Trim(string(ld.AggregateRating.RatingValue), `"`)
		rc := strings.Trim(string(ld.AggregateRating.ReviewCount), `"`)
		if rv != "" {
			app.ReviewScore = strings.TrimSpace(rv + " from " + rc + " reviews")
			break
		}
	}
	if m := storeTagRE.FindSubmatch(body); m != nil {
		var tags []struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(m[1], &tags) == nil {
			for _, t := range tags {
				if t.Name != "" {
					app.Tags = append(app.Tags, t.Name)
				}
			}
		}
	}
}

// pickMax returns the highest-quality entry from a movie's quality map, preferring
// "max" then the largest numeric key present.
func pickMax(m map[string]string) string {
	if m == nil {
		return ""
	}
	if v, ok := m["max"]; ok {
		return v
	}
	best, bestN := "", -1
	for k, v := range m {
		if n, err := strconv.Atoi(k); err == nil && n > bestN {
			best, bestN = v, n
		}
	}
	return best
}

// flexInt unmarshals from a JSON number or a quoted number, because Steam returns
// required_age as either across apps.
type flexInt int

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		// A value like "18+" reduces to its leading digits.
		n = leadingInt(s)
	}
	*f = flexInt(n)
	return nil
}

// flexString unmarshals from a JSON string or number, because Steam returns a
// category id as an int and a genre id as a string.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	*f = flexString(strings.Trim(string(b), `"`))
	return nil
}

func leadingInt(s string) int {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0
	}
	n, _ := strconv.Atoi(s[:i])
	return n
}
