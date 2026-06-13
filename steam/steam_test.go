package steam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte(`"ok"`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	c := NewClient(cfg)

	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `"ok"` {
		t.Errorf("body = %q", body)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`"recovered"`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `"recovered"` {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestGetNullReturnsErrNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("null"))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	c := NewClient(cfg)

	var v any
	err := c.getJSON(context.Background(), srv.URL, &v)
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestFeaturedItemsToGames(t *testing.T) {
	resp := featuredCategoriesResp{
		TopSellers: featuredSection{
			Items: []featuredItem{
				{ID: 570, Name: "Dota 2", FinalPrice: 0, DiscountPercent: 0},
				{ID: 620, Name: "Portal 2", FinalPrice: 999, DiscountPercent: 0},
			},
		},
	}

	items := resp.TopSellers.Items
	games := make([]Game, len(items))
	for i, item := range items {
		games[i] = featuredItemToGame(item, i+1)
	}

	if len(games) != 2 {
		t.Fatalf("got %d games, want 2", len(games))
	}
	if games[0].ID != 570 {
		t.Errorf("games[0].ID = %d, want 570", games[0].ID)
	}
	if games[0].Price != "" {
		t.Errorf("games[0].Price = %q, want empty (0 cents, not free)", games[0].Price)
	}
	if games[1].Price != "$9.99" {
		t.Errorf("games[1].Price = %q, want $9.99", games[1].Price)
	}
	if games[1].URL != "https://store.steampowered.com/app/620/" {
		t.Errorf("games[1].URL = %q", games[1].URL)
	}
}

func TestStoreSearchDecoding(t *testing.T) {
	body, _ := json.Marshal(storeSearchResp{
		Total: 1,
		Items: []storeSearchItem{
			{
				ID:   620,
				Name: "Portal 2",
				Price: struct {
					Final           int    `json:"final"`
					DiscountPercent int    `json:"discount_percent"`
					Currency        string `json:"currency"`
				}{Final: 999, DiscountPercent: 0, Currency: "USD"},
			},
		},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	c := NewClient(cfg)

	var resp storeSearchResp
	if err := c.getJSON(context.Background(), srv.URL, &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(resp.Items))
	}
	g := searchItemToGame(resp.Items[0], 1)
	if g.Price != "$9.99" {
		t.Errorf("price = %q, want $9.99", g.Price)
	}
}

func TestParseAppID(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"570", 570, false},
		{"620", 620, false},
		{"https://store.steampowered.com/app/570/", 570, false},
		{"https://store.steampowered.com/app/570/Dota_2/", 570, false},
		{"not-a-number", 0, true},
		{"", 0, true},
		{"-5", 0, true},
	}
	for _, tc := range cases {
		got, err := ParseAppID(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseAppID(%q): want error, got %d", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseAppID(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseAppID(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
