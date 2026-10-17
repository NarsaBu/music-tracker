package domain

import "time"

// Subscription means "notify me about new release groups of this artist".
// The app is single-user, so the artist MBID identifies the subscription.
type Subscription struct {
	Artist   Artist
	Override FilterOverride
	// BaselineAt: release groups discovered before this moment are not notified (ADR-3).
	BaselineAt time.Time
	CreatedAt  time.Time
}
