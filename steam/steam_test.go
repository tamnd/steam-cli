package steam

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tamnd/any-cli/kit/errs"
)

// testClient returns a client with no pacing and the disk cache off, pointed at
// base for all three hosts so a fixture server can stand in for every Steam host.
func testClient(base string) *Client {
	cfg := DefaultConfig()
	cfg.Delay = 0
	cfg.StoreURL = base
	cfg.CommunityURL = base
	cfg.APIURL = base
	cfg.NoCache = true
	return NewClient(cfg)
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	body, err := testClient(srv.URL).get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want ok", body)
	}
}

func TestGetRetriesOn500(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	c.cfg.Retries = 5

	start := time.Now()
	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

// A 5xx that never recovers ends as ErrNetwork, so mapErr reports exit 8.
func TestGetNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	c.cfg.Retries = 1

	_, err := c.get(context.Background(), srv.URL)
	if !errors.Is(err, ErrNetwork) {
		t.Fatalf("err = %v, want ErrNetwork", err)
	}
	if code := errs.ExitCode(mapErr(err)); code != 8 {
		t.Errorf("mapErr exit code = %d, want 8", code)
	}
}

func TestWallAndExitCodes(t *testing.T) {
	cases := []struct {
		name     string
		handler  http.HandlerFunc
		want     error
		wantExit int
	}{
		{
			"403 is the wall",
			func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) },
			ErrBlocked, 4,
		},
		{
			"503 is the wall",
			func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) },
			ErrBlocked, 4,
		},
		{
			"404 is not found",
			func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) },
			ErrNotFound, 6,
		},
		{
			"cloudflare interstitial body is the wall",
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`<html><head><title>Just a moment...</title></head><body><script src="https://challenges.cloudflare.com/turnstile/v0/api.js"></script></body></html>`))
			},
			ErrBlocked, 4,
		},
		{
			"clean body passes",
			func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"ok":true}`)) },
			nil, 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()
			c := testClient(srv.URL)
			c.cfg.Retries = 0
			_, err := c.get(context.Background(), srv.URL)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if code := errs.ExitCode(mapErr(err)); code != tc.wantExit {
				t.Errorf("exit code = %d, want %d", code, tc.wantExit)
			}
		})
	}
}
