package steam

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "steam" {
		t.Errorf("Scheme = %q, want steam", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "steam" {
		t.Errorf("Identity.Binary = %q, want steam", info.Identity.Binary)
	}
}

func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}
	domains := h.Domains()
	found := false
	for _, d := range domains {
		if d == "steam" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("steam domain not registered; got %v", domains)
	}
}

func TestGameMint(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}
	g := &GameDetail{ID: 570, Name: "Dota 2"}
	u, err := h.Mint(g)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	want := "steam://game/570"
	if u.String() != want {
		t.Errorf("Mint = %q, want %q", u.String(), want)
	}
}
