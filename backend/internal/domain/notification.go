package domain

import "time"

type Channel string

const (
	ChannelEmail Channel = "email"
	// ChannelRSS records feed entries; it also backs the web feed, so it is always written.
	ChannelRSS Channel = "rss"
)

type NotificationStatus string

const (
	StatusPending NotificationStatus = "pending"
	StatusSent    NotificationStatus = "sent"
	StatusFailed  NotificationStatus = "failed"
)

// MaxEmailAttempts: after this many failed digests an email notification is marked failed.
const MaxEmailAttempts = 8

// Notification is an outbox record: one release group in one channel (FR-5.8).
type Notification struct {
	ReleaseGroup MBID
	// Artist is the subscribed artist through whom the release was found ("found via").
	Artist  MBID
	Role    CreditRole
	Channel Channel
	Status  NotificationStatus
	At      time.Time
}

// NotificationItem is a notification with everything needed to show it.
type NotificationItem struct {
	ID           string
	ReleaseGroup ReleaseGroup
	FoundVia     Artist
	Role         CreditRole
	CreatedAt    time.Time
}

// Digest is the content of one email: items in three sections, as in docs/08 §2.1.
type Digest struct {
	Frequency     DigestFrequency
	Released      []NotificationItem
	Announcements []NotificationItem
	Featured      []NotificationItem
}

// NewDigest sorts items into sections: featured first, then by release date.
func NewDigest(freq DigestFrequency, items []NotificationItem, now time.Time) Digest {
	d := Digest{Frequency: freq}
	for _, it := range items {
		switch {
		case it.Role == RoleFeatured:
			d.Featured = append(d.Featured, it)
		case it.ReleaseGroup.FirstRelease.IsAnnouncement(now):
			d.Announcements = append(d.Announcements, it)
		default:
			d.Released = append(d.Released, it)
		}
	}
	return d
}

// Len is the total number of items.
func (d Digest) Len() int { return len(d.Released) + len(d.Announcements) + len(d.Featured) }
