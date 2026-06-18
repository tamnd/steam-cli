package steam

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		in       string
		wantKind string
		wantID   string
	}{
		{"620", "app", "620"},
		{"https://store.steampowered.com/app/620/Portal_2/", "app", "620"},
		{"store.steampowered.com/app/620", "app", "620"},
		{"https://store.steampowered.com/sub/7877/", "package", "7877"},
		{"https://store.steampowered.com/bundle/232/", "package", "232"},
		{"https://steamcommunity.com/profiles/76561197960287930", "profile", "76561197960287930"},
		{"https://steamcommunity.com/id/gabelogannewell", "profile", "gabelogannewell"},
		{"76561197960287930", "profile", "76561197960287930"},
		{"STEAM_1:0:11101", "profile", "76561197960287930"},
		{"[U:1:22202]", "profile", "76561197960287930"},
		{"gabelogannewell", "vanity", "gabelogannewell"},
		{"!!!", "unknown", ""},
		{"with space", "unknown", ""},
	}
	for _, tc := range cases {
		r := Classify(tc.in)
		if r.Kind != tc.wantKind || r.ID != tc.wantID {
			t.Errorf("Classify(%q) = (%q, %q), want (%q, %q)", tc.in, r.Kind, r.ID, tc.wantKind, tc.wantID)
		}
	}
}

func TestURLFor(t *testing.T) {
	cases := []struct {
		kind, id, want string
	}{
		{"app", "620", "https://store.steampowered.com/app/620"},
		{"reviews", "620", "https://store.steampowered.com/app/620"},
		{"news", "620", "https://store.steampowered.com/app/620"},
		{"package", "7877", "https://store.steampowered.com/sub/7877"},
		{"profile", "76561197960287930", "https://steamcommunity.com/profiles/76561197960287930"},
		{"profile", "gabelogannewell", "https://steamcommunity.com/id/gabelogannewell"},
		{"vanity", "gaben", "https://steamcommunity.com/id/gaben"},
		{"search", "", "https://store.steampowered.com/search/"},
		{"featured", "", "https://store.steampowered.com/"},
		{"market", "", "https://steamcommunity.com/market/"},
		{"app", "", ""},
		{"nope", "1", ""},
	}
	for _, tc := range cases {
		if got := URLFor(tc.kind, tc.id); got != tc.want {
			t.Errorf("URLFor(%q, %q) = %q, want %q", tc.kind, tc.id, got, tc.want)
		}
	}
}
