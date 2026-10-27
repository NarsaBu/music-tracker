package notify

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

// FeedSize is the number of entries in an RSS/Atom feed.
const FeedSize = 100

// BuildFeed returns the latest feed entries for a valid token (FR-5.6).
// A wrong token and a disabled feed look the same: not found.
type BuildFeed struct {
	settings      usecase.SettingsRepository
	notifications usecase.NotificationRepository
}

func NewBuildFeed(settings usecase.SettingsRepository, notifications usecase.NotificationRepository) *BuildFeed {
	return &BuildFeed{settings: settings, notifications: notifications}
}

func (uc *BuildFeed) Execute(ctx context.Context, token string) ([]domain.NotificationItem, error) {
	s, err := uc.settings.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("build feed: %w", err)
	}
	if !s.RSS.Enabled || subtle.ConstantTimeCompare([]byte(token), []byte(s.RSS.FeedToken)) != 1 {
		return nil, fmt.Errorf("feed: %w", domain.ErrNotFound)
	}
	items, err := uc.notifications.ListFeed(ctx, usecase.FeedQuery{Limit: FeedSize})
	if err != nil {
		return nil, fmt.Errorf("build feed: %w", err)
	}
	return items, nil
}

const (
	DefaultReleasesLimit = 50
	MaxReleasesLimit     = 100
)

type ListReleasesInput struct {
	Cursor string // opaque, from a previous page
	Type   string
	Role   string
	Sort   string // "" = usecase.FeedByFound
	Limit  int    // 0 = DefaultReleasesLimit
}

type ReleasesPage struct {
	Items      []domain.NotificationItem
	NextCursor string // "" on the last page
}

// ListReleases is the web feed of found releases (US-9).
type ListReleases struct {
	notifications usecase.NotificationRepository
}

func NewListReleases(notifications usecase.NotificationRepository) *ListReleases {
	return &ListReleases{notifications: notifications}
}

func (uc *ListReleases) Execute(ctx context.Context, in ListReleasesInput) (ReleasesPage, error) {
	q := usecase.FeedQuery{
		Sort: usecase.FeedSort(in.Sort), Type: domain.PrimaryType(in.Type), Role: domain.CreditRole(in.Role), Limit: in.Limit,
	}
	if q.Limit == 0 {
		q.Limit = DefaultReleasesLimit
	}
	if q.Sort == "" {
		q.Sort = usecase.FeedByFound
	}
	var fields []domain.FieldError
	if q.Limit < 1 || q.Limit > MaxReleasesLimit {
		fields = append(fields, domain.FieldError{Field: "limit", Message: fmt.Sprintf("must be between 1 and %d", MaxReleasesLimit)})
	}
	if q.Type != "" && !slices.Contains(domain.PrimaryTypes, q.Type) {
		fields = append(fields, domain.FieldError{Field: "type", Message: fmt.Sprintf("unknown value %q", in.Type)})
	}
	if q.Role != "" && !slices.Contains(domain.CreditRoles, q.Role) {
		fields = append(fields, domain.FieldError{Field: "role", Message: fmt.Sprintf("unknown value %q", in.Role)})
	}
	if !slices.Contains(usecase.FeedSorts, q.Sort) {
		fields = append(fields, domain.FieldError{Field: "sort", Message: fmt.Sprintf("unknown value %q", in.Sort)})
	}
	if in.Cursor != "" {
		c, err := decodeCursor(in.Cursor, q.Sort)
		if err != nil {
			fields = append(fields, domain.FieldError{Field: "cursor", Message: "invalid"})
		}
		q.Before = c
	}
	if len(fields) > 0 {
		return ReleasesPage{}, &domain.ValidationError{Fields: fields}
	}

	// One extra row tells whether there is a next page.
	want := q.Limit
	q.Limit++
	items, err := uc.notifications.ListFeed(ctx, q)
	if err != nil {
		return ReleasesPage{}, fmt.Errorf("list releases: %w", err)
	}
	page := ReleasesPage{Items: items}
	if len(items) > want {
		page.Items = items[:want]
		last := page.Items[want-1]
		page.NextCursor = encodeCursor(cursorAfter(last, q.Sort), q.Sort)
	}
	return page, nil
}

func cursorAfter(it domain.NotificationItem, sort usecase.FeedSort) usecase.FeedCursor {
	c := usecase.FeedCursor{At: it.CreatedAt, ID: it.ID}
	if sort == usecase.FeedByRelease {
		c.At = time.Time{}
		if d := it.ReleaseGroup.FirstRelease.Day; d != nil {
			c.At = *d
		}
	}
	return c
}

// A cursor carries its sort order: one taken under another order is invalid.
func encodeCursor(c usecase.FeedCursor, sort usecase.FeedSort) string {
	raw := string(sort) + "|" + strconv.FormatInt(c.At.UnixMicro(), 10) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(s string, sort usecase.FeedSort) (*usecase.FeedCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return nil, fmt.Errorf("cursor: malformed")
	}
	if usecase.FeedSort(parts[0]) != sort {
		return nil, fmt.Errorf("cursor: taken under another sort order")
	}
	at, id := parts[1], parts[2]
	micros, err := strconv.ParseInt(at, 10, 64)
	if err != nil {
		return nil, err
	}
	if _, err := domain.ParseMBID(id); err != nil { // notification ids are UUIDs too
		return nil, err
	}
	return &usecase.FeedCursor{At: time.UnixMicro(micros).UTC(), ID: id}, nil
}
