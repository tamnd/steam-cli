package steam

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/tamnd/steam-cli/pkg/steamid"
)

// ids.go is the offline reference layer: Classify turns any Steam URL, path,
// SteamID, vanity, appid, or packageid into a canonical (kind, id), and URLFor
// builds an addressable URL for a (kind, id). Both are pure and never touch the
// network, so `st ref id` and `st ref url` (and a host's resolve/url) answer
// instantly.
//
// The numeric appid is the canonical key the store, the reviews, the news, the
// player count, and the achievements share; the packageid keys a package; the
// SteamID64 keys a profile. A pasted store link, a bare appid, and a search hit
// resolve to the same app.
//
// The kinds:
//   - app: a store app by numeric appid
//   - package: a package (sub) by numeric packageid
//   - profile: a community profile by SteamID64 or vanity
//   - reviews/news: an app's reviews or news, addressed by the appid
//   - vanity: a community vanity name pending network resolution
//   - search/featured/market: the list surfaces

var (
	numRE      = regexp.MustCompile(`^\d+$`)
	appPathRE  = regexp.MustCompile(`(?:^|/)app/(\d+)`)
	subPathRE  = regexp.MustCompile(`(?:^|/)(?:sub|bundle)/(\d+)`)
	profIDRE   = regexp.MustCompile(`(?:^|/)profiles/(\d+)`)
	vanityRE   = regexp.MustCompile(`(?:^|/)id/([A-Za-z0-9_-]+)`)
	bareVanity = regexp.MustCompile(`^[A-Za-z0-9_.-]{2,32}$`) // a bare vanity token
)

// Classify resolves a reference offline. It accepts a full Steam URL, a path, a
// SteamID in any form, a vanity name, or a bare appid.
func Classify(input string) Ref {
	in := strings.TrimSpace(input)
	r := Ref{Input: input, Kind: "unknown"}

	path := in
	wasURL := false
	if u, err := url.Parse(in); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		path = u.Host + u.Path
		wasURL = true
	}

	switch {
	case appPathRE.MatchString(path):
		r.Kind, r.ID = "app", appPathRE.FindStringSubmatch(path)[1]
	case subPathRE.MatchString(path):
		r.Kind, r.ID = "package", subPathRE.FindStringSubmatch(path)[1]
	case profIDRE.MatchString(path):
		r.Kind, r.ID = "profile", profIDRE.FindStringSubmatch(path)[1]
	case vanityRE.MatchString(path):
		r.Kind, r.ID = "profile", vanityRE.FindStringSubmatch(path)[1]
	default:
		// Not a URL we know: a bare SteamID, a bare appid, or a vanity token.
		clean := strings.Trim(path, "/")
		if id, err := steamid.Parse(clean); err == nil {
			r.Kind, r.ID = "profile", id.ID64String()
		} else if numRE.MatchString(clean) {
			r.Kind, r.ID = "app", clean
		} else if bareVanity.MatchString(clean) {
			r.Kind, r.ID = "vanity", clean
		}
	}

	if r.Kind != "unknown" {
		if wasURL {
			r.URL = in // the human page is more useful than a rebuilt URL
		} else {
			r.URL = URLFor(r.Kind, r.ID)
		}
	}
	return r
}

// URLFor builds an addressable URL for a (kind, id), or "" if it cannot. Reviews
// and news are sections of the app page, so they rebuild the app URL; the record's
// own url carries the specific surface after a fetch.
func URLFor(kind, id string) string {
	id = strings.Trim(id, "/")
	switch kind {
	case "app", "reviews", "news":
		if id == "" {
			return ""
		}
		return StoreURL + "/app/" + id
	case "package":
		if id == "" {
			return ""
		}
		return StoreURL + "/sub/" + id
	case "profile":
		if id == "" {
			return ""
		}
		if numRE.MatchString(id) {
			return CommunityURL + "/profiles/" + id
		}
		return CommunityURL + "/id/" + id
	case "vanity":
		if id == "" {
			return ""
		}
		return CommunityURL + "/id/" + id
	case "search", "browse":
		return StoreURL + "/search/"
	case "featured":
		return StoreURL + "/"
	case "market":
		return CommunityURL + "/market/"
	default:
		return ""
	}
}
