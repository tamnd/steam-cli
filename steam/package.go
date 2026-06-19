package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// package.go reads store.steampowered.com/api/packagedetails and maps a package
// (a sub) onto a Package record. The apps it bundles fill both Apps and the AppRefs
// edges, so a crawl walks a package to each app it contains.

type packageEnvelope map[string]struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

type packageData struct {
	Name      string `json:"name"`
	PageImage string `json:"page_image"`
	SmallLogo string `json:"small_logo"`
	Apps      []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"apps"`
	Price *struct {
		Currency        string `json:"currency"`
		Initial         int    `json:"initial"`
		Final           int    `json:"final"`
		Individual      int    `json:"individual"`
		DiscountPercent int    `json:"discount_percent"`
	} `json:"price"`
	Platforms  *platformsWire `json:"platforms"`
	Controller *struct {
		FullGamepad bool `json:"full_gamepad"`
	} `json:"controller"`
	ReleaseDate *struct {
		ComingSoon bool   `json:"coming_soon"`
		Date       string `json:"date"`
	} `json:"release_date"`
}

// Package fetches one package by packageid and returns it as a record.
func (c *Client) Package(ctx context.Context, packageid string) (*Package, error) {
	packageid = strings.TrimSpace(packageid)
	if !numRE.MatchString(packageid) {
		if r := Classify(packageid); r.Kind == "package" {
			packageid = r.ID
		} else {
			return nil, fmt.Errorf("%w: not a packageid: %q", ErrUsage, packageid)
		}
	}
	u := fmt.Sprintf("%s/api/packagedetails?packageids=%s&cc=%s&l=%s",
		c.cfg.StoreURL, packageid, c.cfg.CC, c.cfg.Lang)
	var env packageEnvelope
	if err := c.getJSON(ctx, u, &env); err != nil {
		return nil, err
	}
	entry, ok := env[packageid]
	if !ok || !entry.Success {
		return nil, ErrNotFound
	}
	var d packageData
	if err := json.Unmarshal(entry.Data, &d); err != nil {
		return nil, fmt.Errorf("decode packagedetails data: %w", err)
	}
	return packageToRecord(&d, packageid), nil
}

func packageToRecord(d *packageData, packageid string) *Package {
	p := &Package{
		ID:        packageid,
		Name:      d.Name,
		PageImage: d.PageImage,
		SmallLogo: d.SmallLogo,
		URL:       StoreURL + "/sub/" + packageid,
	}
	if d.Price != nil {
		p.Price = &Price{
			Currency:    d.Price.Currency,
			Initial:     d.Price.Initial,
			Final:       d.Price.Final,
			Individual:  d.Price.Individual,
			DiscountPct: d.Price.DiscountPercent,
		}
	}
	if d.Controller != nil && d.Controller.FullGamepad {
		p.Controller = "full_gamepad"
	}
	if d.Platforms != nil {
		p.Platforms = &Platforms{Windows: d.Platforms.Windows, Mac: d.Platforms.Mac, Linux: d.Platforms.Linux}
	}
	if d.ReleaseDate != nil {
		p.ReleaseDate = d.ReleaseDate.Date
		p.ComingSoon = d.ReleaseDate.ComingSoon
	}
	for _, a := range d.Apps {
		id := strconv.Itoa(a.ID)
		p.Apps = append(p.Apps, GameLink{AppID: id, Name: a.Name})
		p.AppRefs = append(p.AppRefs, id)
	}
	return p
}
