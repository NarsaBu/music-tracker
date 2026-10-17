// Package feed renders found releases as Atom 1.0 or RSS 2.0 (docs/08-notifications.md §2.2).
package feed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/gorilla/feeds"

	"music-tracker/backend/internal/domain"
)

type Format string

const (
	Atom Format = "atom"
	RSS  Format = "rss"
)

func (f Format) ContentType() string {
	if f == RSS {
		return "application/rss+xml; charset=utf-8"
	}
	return "application/atom+xml; charset=utf-8"
}

var primaryLabels = map[domain.PrimaryType]string{
	domain.PrimaryAlbum: "Album", domain.PrimarySingle: "Single", domain.PrimaryEP: "EP",
	domain.PrimaryBroadcast: "Broadcast", domain.PrimaryOther: "Other",
}

// Render builds the feed document. appURL is the link of the feed itself.
func Render(items []domain.NotificationItem, f Format, appURL string) ([]byte, error) {
	feed := &feeds.Feed{
		Title:       "Music Tracker — новые релизы",
		Link:        &feeds.Link{Href: appURL},
		Description: "Новые релизы исполнителей из ваших подписок",
		Id:          "urn:music-tracker:feed",
		Updated:     LastModified(items),
	}
	for _, it := range items {
		rg := it.ReleaseGroup
		feed.Items = append(feed.Items, &feeds.Item{
			// Stable across rebuilds and token rotations: readers never show a duplicate.
			Id:          "urn:musicbrainz:release-group:" + string(rg.MBID),
			IsPermaLink: "false",
			Title:       Title(it),
			Link:        &feeds.Link{Href: rg.MusicBrainzURL()},
			Author:      &feeds.Author{Name: rg.ArtistCredit},
			Created:     it.CreatedAt,
			Updated:     it.CreatedAt,
			Description: html.EscapeString(details(rg)),
			Content:     content(it),
		})
	}
	var (
		doc string
		err error
	)
	if f == RSS {
		doc, err = feed.ToRss()
	} else {
		doc, err = feed.ToAtom()
	}
	if err != nil {
		return nil, fmt.Errorf("render %s feed: %w", f, err)
	}
	return []byte(doc), nil
}

// Title: "Radiohead — New Album (Album, 2026-11-07)"; featured releases get "[feat.] ".
func Title(it domain.NotificationItem) string {
	rg := it.ReleaseGroup
	t := rg.ArtistCredit + " — " + rg.Title + " (" + details(rg) + ")"
	if it.Role == domain.RoleFeatured {
		t = "[feat.] " + t
	}
	return t
}

// details is "Album + Live, 2026-11-07" or just the date.
func details(rg domain.ReleaseGroup) string {
	var parts []string
	if l, ok := primaryLabels[rg.PrimaryType]; ok {
		parts = append(parts, l)
	}
	parts = append(parts, rg.SecondaryTypes...)
	date := rg.FirstRelease.Raw
	if date == "" {
		date = "дата неизвестна"
	}
	if len(parts) == 0 {
		return date
	}
	return strings.Join(parts, " + ") + ", " + date
}

func content(it domain.NotificationItem) string {
	rg := it.ReleaseGroup
	var b strings.Builder
	fmt.Fprintf(&b, `<p><a href="%s"><img src="%s" width="250" height="250" alt=""></a></p>`,
		html.EscapeString(rg.MusicBrainzURL()), html.EscapeString(rg.CoverURL()))
	fmt.Fprintf(&b, `<p><strong>%s</strong><br>%s<br>%s</p>`,
		html.EscapeString(rg.Title), html.EscapeString(rg.ArtistCredit), html.EscapeString(details(rg)))
	if it.Role == domain.RoleFeatured {
		fmt.Fprintf(&b, `<p>Гостевое участие: %s</p>`, html.EscapeString(it.FoundVia.Name))
	}
	return b.String()
}

// LastModified is the newest entry time; the zero time for an empty feed.
func LastModified(items []domain.NotificationItem) time.Time {
	var last time.Time
	for _, it := range items {
		if it.CreatedAt.After(last) {
			last = it.CreatedAt
		}
	}
	return last
}

// ETag changes whenever an entry is added or the format differs (strong validator).
func ETag(items []domain.NotificationItem, f Format) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%d|%d", f, LastModified(items).UnixMicro(), len(items)) // hash writes never fail
	if len(items) > 0 {
		h.Write([]byte(items[0].ID))
	}
	return `"` + hex.EncodeToString(h.Sum(nil))[:32] + `"`
}

// Parse splits "<token>.atom" / "<token>.rss".
func Parse(file string) (token string, f Format, ok bool) {
	for _, format := range []Format{Atom, RSS} {
		if t, found := strings.CutSuffix(file, "."+string(format)); found && t != "" {
			return t, format, true
		}
	}
	return "", "", false
}
