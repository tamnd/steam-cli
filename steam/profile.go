package steam

import (
	"context"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnd/steam-cli/pkg/steamid"
)

// profile.go reads a public community profile from its XML view. The same fetch
// works for a SteamID64 (/profiles/<id64>?xml=1) and a vanity (/id/<vanity>?xml=1),
// and the returned steamID64 is the canonical id either way, so Resolve reuses it
// to turn a vanity into a SteamID. A private profile returns the public subset; the
// record records the visibility rather than failing. Each most-played game links
// back into the app graph through its store URL.

type profileXML struct {
	XMLName          xml.Name     `xml:"profile"`
	SteamID64        string       `xml:"steamID64"`
	SteamID          string       `xml:"steamID"`
	OnlineState      string       `xml:"onlineState"`
	StateMessage     string       `xml:"stateMessage"`
	PrivacyState     string       `xml:"privacyState"`
	VisibilityState  int          `xml:"visibilityState"`
	AvatarIcon       string       `xml:"avatarIcon"`
	AvatarFull       string       `xml:"avatarFull"`
	VacBanned        int          `xml:"vacBanned"`
	TradeBanState    string       `xml:"tradeBanState"`
	IsLimitedAccount int          `xml:"isLimitedAccount"`
	CustomURL        string       `xml:"customURL"`
	MemberSince      string       `xml:"memberSince"`
	HoursPlayed2Wk   string       `xml:"hoursPlayed2Wk"`
	Location         string       `xml:"location"`
	RealName         string       `xml:"realname"`
	Summary          string       `xml:"summary"`
	MostPlayedGames  []mostPlayed `xml:"mostPlayedGames>mostPlayedGame"`
	Groups           []profGroup  `xml:"groups>group"`
}

type mostPlayed struct {
	GameName      string `xml:"gameName"`
	GameLink      string `xml:"gameLink"`
	HoursOnRecord string `xml:"hoursOnRecord"`
}

type profGroup struct {
	GroupID64 string `xml:"groupID64"`
}

// errResponseRE detects the <response><error>...</error></response> a missing
// profile or unresolved vanity returns instead of a <profile> document.
var errResponseRE = regexp.MustCompile(`(?s)<response>.*?<error>(.*?)</error>`)

var appLinkRE = regexp.MustCompile(`/app/(\d+)`)

// Profile fetches a public profile by SteamID64, vanity, or community URL.
func (c *Client) Profile(ctx context.Context, ref string) (*Profile, error) {
	u, err := c.profileURL(ref)
	if err != nil {
		return nil, err
	}
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	if errResponseRE.Match(body) {
		return nil, ErrNotFound
	}
	var p profileXML
	if err := xml.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("parse profile xml: %w", err)
	}
	if p.SteamID64 == "" {
		return nil, ErrNotFound
	}
	return profileToRecord(&p), nil
}

// Resolve turns a vanity name (or any profile reference) into a SteamID record by
// reading the community XML for its steamID64, then deriving every id form offline.
func (c *Client) Resolve(ctx context.Context, ref string) (*SteamID, error) {
	// A bare SteamID needs no network.
	if id, err := steamid.Parse(strings.TrimSpace(ref)); err == nil {
		return steamIDRecord(ref, "steamid", id), nil
	}
	u, err := c.profileURL(ref)
	if err != nil {
		return nil, err
	}
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	if errResponseRE.Match(body) {
		return nil, ErrNotFound
	}
	var p profileXML
	if err := xml.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("parse profile xml: %w", err)
	}
	if p.SteamID64 == "" {
		return nil, ErrNotFound
	}
	n, err := strconv.ParseUint(p.SteamID64, 10, 64)
	if err != nil {
		return nil, ErrNotFound
	}
	id, err := steamid.FromID64(n)
	if err != nil {
		return nil, ErrNotFound
	}
	rec := steamIDRecord(ref, "vanity", id)
	rec.Vanity = p.CustomURL
	return rec, nil
}

// profileURL builds the XML URL for a reference: a numeric id uses /profiles, a
// vanity uses /id.
func (c *Client) profileURL(ref string) (string, error) {
	r := Classify(ref)
	switch r.Kind {
	case "profile":
		if numRE.MatchString(r.ID) {
			return fmt.Sprintf("%s/profiles/%s?xml=1", c.cfg.CommunityURL, r.ID), nil
		}
		return fmt.Sprintf("%s/id/%s?xml=1", c.cfg.CommunityURL, r.ID), nil
	case "vanity":
		return fmt.Sprintf("%s/id/%s?xml=1", c.cfg.CommunityURL, r.ID), nil
	default:
		return "", fmt.Errorf("%w: not a profile reference: %q", ErrUsage, ref)
	}
}

func profileToRecord(p *profileXML) *Profile {
	rec := &Profile{
		ID:             p.SteamID64,
		PersonaName:    squish(p.SteamID),
		RealName:       squish(p.RealName),
		CustomURL:      p.CustomURL,
		Visibility:     visibility(p),
		OnlineState:    p.OnlineState,
		StateMessage:   squish(p.StateMessage),
		Location:       squish(p.Location),
		Summary:        squish(stripTags(p.Summary)),
		MemberSince:    p.MemberSince,
		Avatar:         p.AvatarIcon,
		AvatarFull:     p.AvatarFull,
		VACBanned:      p.VacBanned == 1,
		TradeBanState:  p.TradeBanState,
		LimitedAccount: p.IsLimitedAccount == 1,
		HoursTwoWeeks:  parseFloat(p.HoursPlayed2Wk),
		URL:            CommunityURL + "/profiles/" + p.SteamID64,
	}
	for _, g := range p.MostPlayedGames {
		gl := GameLink{Name: squish(g.GameName)}
		if m := appLinkRE.FindStringSubmatch(g.GameLink); m != nil {
			gl.AppID = m[1]
			rec.MostPlayedRefs = append(rec.MostPlayedRefs, m[1])
		}
		rec.MostPlayed = append(rec.MostPlayed, gl)
	}
	for _, g := range p.Groups {
		if g.GroupID64 != "" {
			rec.Groups = append(rec.Groups, g.GroupID64)
		}
	}
	return rec
}

// visibility maps the XML privacy fields onto a stable label.
func visibility(p *profileXML) string {
	switch strings.ToLower(p.PrivacyState) {
	case "public":
		return "public"
	case "private":
		return "private"
	case "friendsonly":
		return "friendsonly"
	}
	switch p.VisibilityState {
	case 3:
		return "public"
	case 1:
		return "private"
	case 2:
		return "friendsonly"
	}
	return ""
}

// steamIDRecord builds a SteamID record from a parsed id, filling every form.
func steamIDRecord(input, kind string, id steamid.ID) *SteamID {
	return &SteamID{
		Input:     input,
		Kind:      kind,
		SteamID64: id.ID64String(),
		SteamID3:  id.ID3(),
		Steam2:    id.Steam2(),
		AccountID: strconv.FormatUint(uint64(id.AccountID), 10),
		URL:       CommunityURL + "/profiles/" + id.ID64String(),
	}
}

var tagStripRE = regexp.MustCompile(`<[^>]+>`)

// stripTags removes HTML tags from XML CDATA text (a profile summary may carry
// markup).
func stripTags(s string) string { return tagStripRE.ReplaceAllString(s, " ") }

// parseFloat reads a possibly comma-grouped number ("1,234.5") into a float64.
func parseFloat(s string) float64 {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
