// Package steamid converts between the three SteamID forms with no network call.
//
// Steam identifies an account three ways. The 64-bit SteamID (SteamID64) is the
// canonical numeric id, for example 76561197960287930. The modern text form,
// SteamID3, writes the same account as [U:1:22202]. The legacy form, Steam2,
// writes it as STEAM_1:0:11101. All three encode one 32-bit account id; the
// conversion is pure arithmetic over that account id and a fixed base, so a
// caller can translate any form into the others offline.
//
// A vanity name (the custom URL a user picks, like /id/gaben) is not a SteamID
// and cannot be resolved without the network; Parse reports it as ErrVanity so
// the caller routes it to the community resolver instead.
package steamid

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// base is the SteamID64 of account id 0 for an individual account in the public
// universe: 0x0110000100000000. An individual SteamID64 is base + accountID.
const base uint64 = 76561197960265728

// ErrVanity is returned by Parse when the input is a plausible vanity name
// rather than a SteamID form. Resolving it needs the network.
var ErrVanity = errors.New("input is a vanity name, not a SteamID; resolve it over the network")

// ErrInvalid is returned by Parse when the input is neither a SteamID form nor a
// plausible vanity name.
var ErrInvalid = errors.New("not a recognizable SteamID")

var (
	id64RE   = regexp.MustCompile(`^7656119\d{10}$`)
	id3RE    = regexp.MustCompile(`^\[?U:1:(\d+)\]?$`)
	steam2RE = regexp.MustCompile(`^STEAM_([0-5]):([01]):(\d+)$`)
	vanityRE = regexp.MustCompile(`^[A-Za-z0-9_-]{2,32}$`)
)

// ID is one parsed account, printable in every form.
type ID struct {
	AccountID uint32
}

// FromID64 builds an ID from a 64-bit SteamID. It returns ErrInvalid when n is
// below the individual-account base.
func FromID64(n uint64) (ID, error) {
	if n < base {
		return ID{}, ErrInvalid
	}
	return ID{AccountID: uint32(n - base)}, nil
}

// FromAccountID builds an ID from a bare 32-bit account id.
func FromAccountID(a uint32) ID { return ID{AccountID: a} }

// ID64 returns the 64-bit SteamID.
func (id ID) ID64() uint64 { return base + uint64(id.AccountID) }

// ID64String returns the 64-bit SteamID as a decimal string.
func (id ID) ID64String() string { return strconv.FormatUint(id.ID64(), 10) }

// ID3 returns the modern text form, for example "[U:1:22202]".
func (id ID) ID3() string { return fmt.Sprintf("[U:1:%d]", id.AccountID) }

// Steam2 returns the legacy form, for example "STEAM_1:0:11101". The universe
// digit is 1 (public); the low bit of the account id is the Y field and the rest
// is the Z field.
func (id ID) Steam2() string {
	return fmt.Sprintf("STEAM_1:%d:%d", id.AccountID&1, id.AccountID>>1)
}

// Parse reads any SteamID form (64-bit, [U:1:N], or STEAM_X:Y:Z) into an ID. A
// plausible vanity name returns ErrVanity; anything else returns ErrInvalid.
func Parse(input string) (ID, error) {
	s := strings.TrimSpace(input)
	switch {
	case id64RE.MatchString(s):
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return ID{}, ErrInvalid
		}
		return FromID64(n)
	case id3RE.MatchString(s):
		m := id3RE.FindStringSubmatch(s)
		a, err := strconv.ParseUint(m[1], 10, 32)
		if err != nil {
			return ID{}, ErrInvalid
		}
		return ID{AccountID: uint32(a)}, nil
	case steam2RE.MatchString(s):
		m := steam2RE.FindStringSubmatch(s)
		y, _ := strconv.ParseUint(m[2], 10, 32)
		z, err := strconv.ParseUint(m[3], 10, 32)
		if err != nil {
			return ID{}, ErrInvalid
		}
		return ID{AccountID: uint32(z*2 + y)}, nil
	case vanityRE.MatchString(s):
		return ID{}, ErrVanity
	default:
		return ID{}, ErrInvalid
	}
}
