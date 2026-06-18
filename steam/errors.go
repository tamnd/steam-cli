package steam

import "errors"

// Sentinel errors the library returns; domain.go's mapErr turns each into the kit
// error kind that carries the right exit code (see spec 8031 section 4.5). There
// is no need-key error, because st reads only keyless surfaces.
var (
	// ErrBlocked is the wall: a 403, or an anti-bot interstitial served with a 200.
	// Maps to need-auth (exit 4). The remedy is to slow down or try again later;
	// st does not defeat the wall.
	ErrBlocked = errors.New("blocked: Steam served an anti-bot wall here. Slow down with --rate or try again later")

	// ErrNotFound is a missing entity: an appdetails/packagedetails success:false,
	// a market priceoverview success:false, a player-count result != 1, or a 404.
	// Maps to exit 6.
	ErrNotFound = errors.New("not found")

	// ErrRateLimited is a sustained 429 after retries. Maps to exit 5.
	ErrRateLimited = errors.New("rate limited by Steam: slow down with --rate or try again later")

	// ErrNetwork is a transport failure that survives every retry: a connection
	// refused, a timeout, a DNS error, or a 5xx that never recovered. Maps to exit 8.
	ErrNetwork = errors.New("network error reaching Steam")

	// ErrUsage is a bad argument the library catches (an unrecognized reference, a
	// missing market name). Maps to exit 2.
	ErrUsage = errors.New("usage")
)
