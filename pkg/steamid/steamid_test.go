package steamid

import "testing"

// The canonical example: SteamID64 76561197960287930 is account id 22202, which
// writes as [U:1:22202] and STEAM_1:0:11101.
func TestForms(t *testing.T) {
	id, err := FromID64(76561197960287930)
	if err != nil {
		t.Fatal(err)
	}
	if id.AccountID != 22202 {
		t.Errorf("AccountID = %d, want 22202", id.AccountID)
	}
	if got := id.ID64String(); got != "76561197960287930" {
		t.Errorf("ID64String = %q", got)
	}
	if got := id.ID3(); got != "[U:1:22202]" {
		t.Errorf("ID3 = %q, want [U:1:22202]", got)
	}
	if got := id.Steam2(); got != "STEAM_1:0:11101" {
		t.Errorf("Steam2 = %q, want STEAM_1:0:11101", got)
	}
}

func TestParseRoundTrip(t *testing.T) {
	forms := []string{
		"76561197960287930",
		"[U:1:22202]",
		"U:1:22202",
		"STEAM_1:0:11101",
		"STEAM_0:0:11101",
	}
	for _, in := range forms {
		id, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", in, err)
		}
		if id.AccountID != 22202 {
			t.Errorf("Parse(%q) AccountID = %d, want 22202", in, id.AccountID)
		}
		if id.ID64String() != "76561197960287930" {
			t.Errorf("Parse(%q) ID64 = %q", in, id.ID64String())
		}
	}
}

// The Steam2 Y bit is the low bit of the account id, so an odd account id must
// survive the round trip through STEAM_1:1:Z.
func TestSteam2OddAccount(t *testing.T) {
	id := FromAccountID(22203) // odd
	if got := id.Steam2(); got != "STEAM_1:1:11101" {
		t.Errorf("Steam2 = %q, want STEAM_1:1:11101", got)
	}
	back, err := Parse("STEAM_1:1:11101")
	if err != nil {
		t.Fatal(err)
	}
	if back.AccountID != 22203 {
		t.Errorf("AccountID = %d, want 22203", back.AccountID)
	}
}

func TestParseVanityAndInvalid(t *testing.T) {
	if _, err := Parse("gabelogannewell"); err != ErrVanity {
		t.Errorf("Parse(vanity) = %v, want ErrVanity", err)
	}
	if _, err := Parse("gaben"); err != ErrVanity {
		t.Errorf("Parse(short vanity) = %v, want ErrVanity", err)
	}
	for _, bad := range []string{"", "!!!", "with space", "STEAM_9:0:1"} {
		if _, err := Parse(bad); err != ErrInvalid {
			t.Errorf("Parse(%q) = %v, want ErrInvalid", bad, err)
		}
	}
}

// A SteamID64 below the individual-account base is not a valid individual id.
func TestFromID64BelowBase(t *testing.T) {
	if _, err := FromID64(1); err != ErrInvalid {
		t.Errorf("FromID64(1) = %v, want ErrInvalid", err)
	}
}
