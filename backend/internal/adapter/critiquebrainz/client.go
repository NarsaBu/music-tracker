// Package critiquebrainz implements usecase.ReviewSource on top of the
// CritiqueBrainz API: Creative Commons reviews of MusicBrainz entities.
package critiquebrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"music-tracker/backend/internal/domain"
)

const (
	// PageSize is how many reviews, most popular first, one answer holds.
	PageSize = 10
	// maxTextRunes bounds a stored review; the full text stays on CritiqueBrainz.
	maxTextRunes = 1500
)

type Config struct {
	BaseURL   string // default https://critiquebrainz.org/ws/1
	UserAgent string
	Timeout   time.Duration // default 15s
}

// Client is safe for concurrent use.
type Client struct {
	http      *http.Client
	baseURL   string
	userAgent string
}

func NewClient(cfg Config) *Client {
	c := &Client{
		http:      &http.Client{Timeout: cfg.Timeout},
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		userAgent: cfg.UserAgent,
	}
	if c.http.Timeout == 0 {
		c.http.Timeout = 15 * time.Second
	}
	if c.baseURL == "" {
		c.baseURL = "https://critiquebrainz.org/ws/1"
	}
	return c
}

type reviewListJSON struct {
	Count         int `json:"count"`
	AverageRating *struct {
		Count  int     `json:"count"`
		Rating float64 `json:"rating"`
	} `json:"average_rating"`
	Reviews []reviewJSON `json:"reviews"`
}

type reviewJSON struct {
	ID          string `json:"id"`
	Rating      *int   `json:"rating"`
	Text        string `json:"text"`
	Language    string `json:"language"`
	PublishedOn string `json:"published_on"`
	Created     string `json:"created"`
	FullName    string `json:"full_name"` // license
	User        struct {
		DisplayName string `json:"display_name"`
	} `json:"user"`
}

// ReleaseGroupReviews returns the most popular reviews of a release group.
func (c *Client) ReleaseGroupReviews(ctx context.Context, rg domain.MBID) (domain.Reviews, error) {
	q := url.Values{
		"entity_id":   {string(rg)},
		"entity_type": {"release_group"},
		"sort":        {"popularity"},
		"limit":       {strconv.Itoa(PageSize)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/review/?"+q.Encode(), nil)
	if err != nil {
		return domain.Reviews{}, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return domain.Reviews{}, ctx.Err()
		}
		return domain.Reviews{}, fmt.Errorf("%w: critiquebrainz: %w", domain.ErrUpstreamUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return domain.Reviews{}, fmt.Errorf("%w: critiquebrainz: status %d", domain.ErrUpstreamUnavailable, resp.StatusCode)
	}

	var body reviewListJSON
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return domain.Reviews{}, fmt.Errorf("%w: critiquebrainz: decode response: %w", domain.ErrUpstreamUnavailable, err)
	}
	out := domain.Reviews{Total: body.Count, Items: make([]domain.Review, 0, len(body.Reviews))}
	if a := body.AverageRating; a != nil && a.Count > 0 {
		out.Average, out.Rated = a.Rating, a.Count
	}
	for _, r := range body.Reviews {
		rev := domain.Review{
			ID:          r.ID,
			Author:      r.User.DisplayName,
			Text:        plainText(r.Text),
			Language:    r.Language,
			PublishedAt: parseTime(r.PublishedOn, r.Created),
			License:     r.FullName,
		}
		if r.Rating != nil {
			rev.Rating = *r.Rating
		}
		out.Items = append(out.Items, rev)
	}
	return out, nil
}

// plainText drops Markdown emphasis and bounds the length; the review links to the full text.
func plainText(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "*", ""))
	if utf8.RuneCountInString(s) <= maxTextRunes {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:maxTextRunes])) + "…"
}

// parseTime reads the first parsable HTTP date ("Wed, 25 Apr 2007 00:00:00 GMT").
func parseTime(values ...string) time.Time {
	for _, v := range values {
		if t, err := http.ParseTime(v); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
