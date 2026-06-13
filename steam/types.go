package steam

import (
	"fmt"
	"strings"
	"time"
)

// Game is the record emitted for list commands: top-sellers, new-releases, specials, search.
type Game struct {
	Rank     int    `json:"rank"`
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Price    string `json:"price"`
	Discount int    `json:"discount"`
	URL      string `json:"url"`
}

// GameDetail is the record emitted by the game command.
type GameDetail struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Developers  string `json:"developers"`
	Publishers  string `json:"publishers"`
	Price       string `json:"price"`
	Discount    int    `json:"discount"`
	Released    string `json:"released"`
	Metacritic  int    `json:"metacritic"`
	Genres      string `json:"genres"`
	IsFree      bool   `json:"is_free"`
	URL         string `json:"url"`
}

// Review is the record emitted by the reviews command.
type Review struct {
	Rank      int    `json:"rank"`
	Voted     string `json:"voted"`
	Author    string `json:"author"`
	Playtime  int    `json:"playtime"`
	Review    string `json:"review"`
	Timestamp string `json:"timestamp"`
}

// ─── wire types ──────────────────────────────────────────────────────────────

type featuredCategoriesResp struct {
	TopSellers  featuredSection `json:"top_sellers"`
	NewReleases featuredSection `json:"new_releases"`
	Specials    featuredSection `json:"specials"`
}

type featuredSection struct {
	Items []featuredItem `json:"items"`
}

type featuredItem struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	DiscountPercent int    `json:"discount_percent"`
	FinalPrice      int    `json:"final_price"`
	Currency        string `json:"currency"`
}

type storeSearchResp struct {
	Items []storeSearchItem `json:"items"`
	Total int               `json:"total"`
}

type storeSearchItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price struct {
		Final           int    `json:"final"`
		DiscountPercent int    `json:"discount_percent"`
		Currency        string `json:"currency"`
	} `json:"price"`
	Type string `json:"type"`
}

type appDetailsOuter map[string]appDetailsEntry

type appDetailsEntry struct {
	Success bool        `json:"success"`
	Data    appDataWire `json:"data"`
}

type appDataWire struct {
	AppID            int      `json:"steam_appid"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	ShortDescription string   `json:"short_description"`
	Developers       []string `json:"developers"`
	Publishers       []string `json:"publishers"`
	IsFree           bool     `json:"is_free"`
	PriceOverview    struct {
		Final           int `json:"final"`
		DiscountPercent int `json:"discount_percent"`
	} `json:"price_overview"`
	ReleaseDate struct {
		Date string `json:"date"`
	} `json:"release_date"`
	Metacritic struct {
		Score int `json:"score"`
	} `json:"metacritic"`
	Genres []struct {
		Description string `json:"description"`
	} `json:"genres"`
}

type reviewsResp struct {
	Success int          `json:"success"`
	Cursor  string       `json:"cursor"`
	Reviews []reviewWire `json:"reviews"`
}

type reviewWire struct {
	RecommendationID string `json:"recommendationid"`
	Author           struct {
		SteamID         string `json:"steamid"`
		PlaytimeForever int    `json:"playtime_forever"`
	} `json:"author"`
	Review           string `json:"review"`
	TimestampCreated int64  `json:"timestamp_created"`
	VotedUp          bool   `json:"voted_up"`
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func storeURL(id int) string {
	return fmt.Sprintf("https://store.steampowered.com/app/%d/", id)
}

// formatPrice converts a cent-value price integer to a human-readable string.
// isFree overrides everything; 0 cents with isFree=false means unlisted → "".
func formatPrice(finalCents int, isFree bool) string {
	if isFree {
		return "Free"
	}
	if finalCents == 0 {
		return ""
	}
	return fmt.Sprintf("$%.2f", float64(finalCents)/100)
}

func featuredItemToGame(item featuredItem, rank int) Game {
	return Game{
		Rank:     rank,
		ID:       item.ID,
		Name:     item.Name,
		Price:    formatPrice(item.FinalPrice, false),
		Discount: item.DiscountPercent,
		URL:      storeURL(item.ID),
	}
}

func searchItemToGame(item storeSearchItem, rank int) Game {
	return Game{
		Rank:     rank,
		ID:       item.ID,
		Name:     item.Name,
		Price:    formatPrice(item.Price.Final, false),
		Discount: item.Price.DiscountPercent,
		URL:      storeURL(item.ID),
	}
}

func appDataToDetail(d appDataWire) GameDetail {
	devs := strings.Join(d.Developers, ";")
	pubs := strings.Join(d.Publishers, ";")
	genres := make([]string, len(d.Genres))
	for i, g := range d.Genres {
		genres[i] = g.Description
	}
	return GameDetail{
		ID:          d.AppID,
		Name:        d.Name,
		Type:        d.Type,
		Description: d.ShortDescription,
		Developers:  devs,
		Publishers:  pubs,
		Price:       formatPrice(d.PriceOverview.Final, d.IsFree),
		Discount:    d.PriceOverview.DiscountPercent,
		Released:    d.ReleaseDate.Date,
		Metacritic:  d.Metacritic.Score,
		Genres:      strings.Join(genres, ";"),
		IsFree:      d.IsFree,
		URL:         storeURL(d.AppID),
	}
}

func reviewWireToReview(rw reviewWire, rank int) Review {
	voted := "not recommended"
	if rw.VotedUp {
		voted = "recommended"
	}
	ts := time.Unix(rw.TimestampCreated, 0).UTC().Format(time.RFC3339)
	return Review{
		Rank:      rank,
		Voted:     voted,
		Author:    rw.Author.SteamID,
		Playtime:  rw.Author.PlaytimeForever / 60,
		Review:    rw.Review,
		Timestamp: ts,
	}
}
