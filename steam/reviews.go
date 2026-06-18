package steam

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// reviews.go reads store.steampowered.com/appreviews/<appid>?json=1 and follows
// the cursor until it has limit reviews or the cursor stops advancing. Each review
// maps to a Review; the author SteamID64 fills both Author and the AuthorRef edge,
// so a crawl can walk a review to its author's public profile.

type reviewsResponse struct {
	Success int          `json:"success"`
	Reviews []reviewWire `json:"reviews"`
	Cursor  string       `json:"cursor"`
}

type reviewWire struct {
	RecommendationID string `json:"recommendationid"`
	Author           struct {
		SteamID          string `json:"steamid"`
		PlaytimeForever  int    `json:"playtime_forever"`
		PlaytimeAtReview int    `json:"playtime_at_review"`
	} `json:"author"`
	Language          string    `json:"language"`
	Review            string    `json:"review"`
	TimestampCreated  int64     `json:"timestamp_created"`
	TimestampUpdated  int64     `json:"timestamp_updated"`
	VotedUp           bool      `json:"voted_up"`
	VotesUp           int       `json:"votes_up"`
	VotesFunny        int       `json:"votes_funny"`
	WeightedVoteScore flexFloat `json:"weighted_vote_score"`
	CommentCount      int       `json:"comment_count"`
	SteamPurchase     bool      `json:"steam_purchase"`
	ReceivedForFree   bool      `json:"received_for_free"`
	EarlyAccess       bool      `json:"written_during_early_access"`
}

// Reviews returns up to limit reviews for appid, paging through the cursor.
func (c *Client) Reviews(ctx context.Context, appid string, limit int) ([]*Review, error) {
	appid = strings.TrimSpace(appid)
	if !numRE.MatchString(appid) {
		if r := Classify(appid); r.Kind == "app" {
			appid = r.ID
		} else {
			return nil, fmt.Errorf("%w: not an appid: %q", ErrUsage, appid)
		}
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	cursor := "*"
	var out []*Review
	for {
		perPage := limit - len(out)
		if perPage > 100 {
			perPage = 100
		}
		u := fmt.Sprintf("%s/appreviews/%s?json=1&filter=%s&language=%s&purchase_type=%s&num_per_page=%d&cursor=%s",
			c.cfg.StoreURL, appid,
			url.QueryEscape(c.cfg.ReviewFilter), url.QueryEscape(c.cfg.ReviewLanguage),
			url.QueryEscape(c.cfg.PurchaseType), perPage, url.QueryEscape(cursor))
		var resp reviewsResponse
		if err := c.getJSON(ctx, u, &resp); err != nil {
			return nil, err
		}
		if resp.Success != 1 {
			return nil, ErrNotFound
		}
		for i := range resp.Reviews {
			out = append(out, reviewToRecord(&resp.Reviews[i], appid))
			if len(out) >= limit {
				return out, nil
			}
		}
		if resp.Cursor == "" || resp.Cursor == cursor || len(resp.Reviews) == 0 {
			break
		}
		cursor = resp.Cursor
	}
	return out, nil
}

func reviewToRecord(r *reviewWire, appid string) *Review {
	return &Review{
		ID:               r.RecommendationID,
		App:              appid,
		Author:           r.Author.SteamID,
		Language:         r.Language,
		Body:             r.Review,
		VotedUp:          r.VotedUp,
		VotesUp:          r.VotesUp,
		VotesFunny:       r.VotesFunny,
		WeightedScore:    float64(r.WeightedVoteScore),
		Comments:         r.CommentCount,
		SteamPurchase:    r.SteamPurchase,
		ReceivedFree:     r.ReceivedForFree,
		EarlyAccess:      r.EarlyAccess,
		PlaytimeAtReview: r.Author.PlaytimeAtReview,
		PlaytimeForever:  r.Author.PlaytimeForever,
		Created:          unixToRFC(r.TimestampCreated),
		Updated:          unixToRFC(r.TimestampUpdated),
		AuthorRef:        r.Author.SteamID,
	}
}

// unixToRFC turns a unix timestamp into an RFC3339 UTC string, or "" for zero.
func unixToRFC(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

// flexFloat unmarshals from a JSON number or a quoted number, because Steam
// returns weighted_vote_score as a quoted float.
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = flexFloat(v)
	return nil
}
