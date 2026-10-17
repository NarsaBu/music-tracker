package musicbrainz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"music-tracker/backend/internal/domain"
)

type artistDTO struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	SortName       string      `json:"sort-name"`
	Disambiguation string      `json:"disambiguation"`
	Type           string      `json:"type"`
	Country        string      `json:"country"`
	LifeSpan       lifeSpanDTO `json:"life-span"`
	Score          int         `json:"score"` // search results only
}

type lifeSpanDTO struct {
	Begin string `json:"begin"`
	End   string `json:"end"`
	Ended *bool  `json:"ended"` // null in some responses
}

type artistSearchDTO struct {
	Artists []artistDTO `json:"artists"`
}

func (d artistDTO) toDomain() (domain.Artist, error) {
	mbid, err := domain.ParseMBID(d.ID)
	if err != nil {
		return domain.Artist{}, fmt.Errorf("musicbrainz: artist id %q: %w", d.ID, err)
	}
	sortName := d.SortName
	if sortName == "" {
		sortName = d.Name
	}
	return domain.Artist{
		MBID:           mbid,
		Name:           d.Name,
		SortName:       sortName,
		Disambiguation: d.Disambiguation,
		Type:           d.Type,
		Country:        d.Country,
		LifeSpan: domain.LifeSpan{
			Begin: d.LifeSpan.Begin,
			End:   d.LifeSpan.End,
			Ended: d.LifeSpan.Ended != nil && *d.LifeSpan.Ended,
		},
	}, nil
}

// SearchArtists runs a phrase query on the artist name and falls back to a
// plain query over all default fields (aliases, sort name) when nothing matches.
func (c *Client) SearchArtists(ctx context.Context, query string, limit int) ([]domain.ArtistCandidate, error) {
	escaped := escapeLucene(query)
	found, err := c.searchArtists(ctx, `artist:"`+escaped+`"`, limit)
	if err != nil || len(found) > 0 {
		return found, err
	}
	return c.searchArtists(ctx, escaped, limit)
}

func (c *Client) searchArtists(ctx context.Context, lucene string, limit int) ([]domain.ArtistCandidate, error) {
	var resp artistSearchDTO
	q := url.Values{"query": {lucene}, "limit": {strconv.Itoa(limit)}}
	if err := c.get(ctx, "/artist", q, &resp); err != nil {
		return nil, err
	}
	out := make([]domain.ArtistCandidate, 0, len(resp.Artists))
	for _, d := range resp.Artists {
		a, err := d.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, domain.ArtistCandidate{Artist: a, Score: d.Score})
	}
	return out, nil
}

func (c *Client) GetArtist(ctx context.Context, mbid domain.MBID) (domain.Artist, error) {
	var d artistDTO
	err := c.get(ctx, "/artist/"+string(mbid), nil, &d)
	// MusicBrainz answers 400 "Invalid mbid." for well-formed UUIDs it does not know, e.g. the nil UUID.
	var se *statusError
	if errors.As(err, &se) && se.code == http.StatusBadRequest {
		return domain.Artist{}, fmt.Errorf("artist %s: %w", mbid, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Artist{}, err
	}
	return d.toDomain()
}

// luceneSpecial are the characters with meaning in Lucene query syntax.
// "&&" and "||" are covered by escaping single & and |.
const luceneSpecial = `+-&|!(){}[]^"~*?:\/`

func escapeLucene(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(luceneSpecial, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
