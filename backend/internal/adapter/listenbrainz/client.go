// Package listenbrainz implements usecase.SimilarArtistsSource on top of the
// ListenBrainz Labs similar-artists dataset: artists listened to in the same
// sessions. Labs is experimental, so failures surface as ErrUpstreamUnavailable
// and the caller falls back to its cache.
package listenbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"music-tracker/backend/internal/domain"
)

// DefaultAlgorithm: sessions over ~20 years, the widest coverage for less-played artists.
const DefaultAlgorithm = "session_based_days_7500_session_300_contribution_5_threshold_10_limit_100_filter_True_skip_30"

type Config struct {
	BaseURL   string // default https://labs.api.listenbrainz.org
	Algorithm string // default DefaultAlgorithm
	UserAgent string
	Timeout   time.Duration // default 15s
}

// Client is safe for concurrent use.
type Client struct {
	http      *http.Client
	baseURL   string
	algorithm string
	userAgent string
}

func NewClient(cfg Config) *Client {
	c := &Client{
		http:      &http.Client{Timeout: cfg.Timeout},
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		algorithm: cfg.Algorithm,
		userAgent: cfg.UserAgent,
	}
	if c.http.Timeout == 0 {
		c.http.Timeout = 15 * time.Second
	}
	if c.baseURL == "" {
		c.baseURL = "https://labs.api.listenbrainz.org"
	}
	if c.algorithm == "" {
		c.algorithm = DefaultAlgorithm
	}
	return c
}

type similarArtistJSON struct {
	ArtistMBID string  `json:"artist_mbid"`
	Name       string  `json:"name"`
	Comment    string  `json:"comment"`
	Type       *string `json:"type"`
	Score      int     `json:"score"`
}

// SimilarArtists returns the dataset rows for the artist, most similar first.
func (c *Client) SimilarArtists(ctx context.Context, artist domain.MBID) ([]domain.SimilarArtist, error) {
	q := url.Values{"artist_mbids": {string(artist)}, "algorithm": {c.algorithm}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/similar-artists/json?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: listenbrainz: %w", domain.ErrUpstreamUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		// A 400 here means the algorithm name went stale: Labs renames datasets.
		return nil, fmt.Errorf("%w: listenbrainz: status %d", domain.ErrUpstreamUnavailable, resp.StatusCode)
	}

	var rows []similarArtistJSON
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("%w: listenbrainz: decode response: %w", domain.ErrUpstreamUnavailable, err)
	}
	out := make([]domain.SimilarArtist, 0, len(rows))
	for _, r := range rows {
		mbid, err := domain.ParseMBID(r.ArtistMBID)
		if err != nil {
			continue
		}
		a := domain.SimilarArtist{MBID: mbid, Name: r.Name, Disambiguation: r.Comment, Score: r.Score}
		if r.Type != nil {
			a.Type = *r.Type
		}
		out = append(out, a)
	}
	return out, nil
}
