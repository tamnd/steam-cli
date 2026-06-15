package steam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(srv *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 0
	return NewClientWithConfig(cfg)
}

func sampleResponse(n int) wireTopSellers {
	items := make([]wireItem, n)
	for i := range items {
		items[i] = wireItem{
			ID:              570 + i,
			Name:            "Game " + string(rune('A'+i)),
			DiscountPercent: i * 10,
			FinalPrice:      (i + 1) * 999,
			Currency:        "USD",
		}
	}
	return wireTopSellers{TopSellers: struct {
		Items []wireItem `json:"items"`
	}{Items: items}}
}

func TestTopSellers(t *testing.T) {
	resp := sampleResponse(5)
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	got, err := c.TopSellers(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Errorf("got %d games, want 5", len(got))
	}
	if got[0].Name != "Game A" {
		t.Errorf("Name = %q, want Game A", got[0].Name)
	}
}

func TestTopSellersLimit(t *testing.T) {
	resp := sampleResponse(10)
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	got, err := c.TopSellers(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("got %d games, want 3 (limit)", len(got))
	}
}

func TestFreeGame(t *testing.T) {
	resp := wireTopSellers{TopSellers: struct {
		Items []wireItem `json:"items"`
	}{Items: []wireItem{{ID: 570, Name: "Dota 2", FinalPrice: 0, DiscountPercent: 0}}}}
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	got, err := c.TopSellers(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Price != "Free" {
		t.Errorf("Price = %q, want Free", got[0].Price)
	}
}

func TestURLConstruction(t *testing.T) {
	resp := sampleResponse(1)
	body, _ := json.Marshal(resp)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	got, err := c.TopSellers(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://store.steampowered.com/app/570"
	if got[0].URL != want {
		t.Errorf("URL = %q, want %q", got[0].URL, want)
	}
}
