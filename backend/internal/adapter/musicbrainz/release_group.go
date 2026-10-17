package musicbrainz

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"music-tracker/backend/internal/domain"
)

type releaseGroupDTO struct {
	ID               string      `json:"id"`
	Title            string      `json:"title"`
	PrimaryType      string      `json:"primary-type"`
	SecondaryTypes   []string    `json:"secondary-types"`
	FirstReleaseDate string      `json:"first-release-date"`
	ArtistCredit     []creditDTO `json:"artist-credit"`
}

type creditDTO struct {
	Name       string `json:"name"`
	JoinPhrase string `json:"joinphrase"`
	Artist     struct {
		ID string `json:"id"`
	} `json:"artist"`
}

type releaseGroupBrowseDTO struct {
	Count         int               `json:"release-group-count"`
	Offset        int               `json:"release-group-offset"`
	ReleaseGroups []releaseGroupDTO `json:"release-groups"`
}

// ListReleaseGroups browses every page of the artist's release groups, credits included.
// Any page failing fails the whole call: a partial catalog must not be stored.
func (c *Client) ListReleaseGroups(ctx context.Context, artist domain.MBID) ([]domain.ReleaseGroup, error) {
	var out []domain.ReleaseGroup
	for offset := 0; ; {
		var page releaseGroupBrowseDTO
		q := url.Values{
			"artist": {string(artist)},
			"inc":    {"artist-credits"},
			"limit":  {strconv.Itoa(c.pageSize)},
			"offset": {strconv.Itoa(offset)},
		}
		if err := c.get(ctx, "/release-group", q, &page); err != nil {
			return nil, err
		}
		for _, d := range page.ReleaseGroups {
			rg, err := d.toDomain()
			if err != nil {
				return nil, err
			}
			out = append(out, rg)
		}
		offset += len(page.ReleaseGroups)
		if len(page.ReleaseGroups) == 0 || offset >= page.Count {
			return out, nil
		}
	}
}

func (d releaseGroupDTO) toDomain() (domain.ReleaseGroup, error) {
	mbid, err := domain.ParseMBID(d.ID)
	if err != nil {
		return domain.ReleaseGroup{}, fmt.Errorf("musicbrainz: release group id %q: %w", d.ID, err)
	}
	credits := make(domain.ArtistCredit, 0, len(d.ArtistCredit))
	for _, c := range d.ArtistCredit {
		// An unparsable artist id is kept empty: RoleOf then treats the artist as absent.
		artistID, _ := domain.ParseMBID(c.Artist.ID)
		credits = append(credits, domain.CreditName{ArtistMBID: artistID, Name: c.Name, JoinPhrase: c.JoinPhrase})
	}
	secondary := d.SecondaryTypes
	if secondary == nil {
		secondary = []string{}
	}
	return domain.ReleaseGroup{
		MBID:           mbid,
		Title:          d.Title,
		PrimaryType:    primaryType(d.PrimaryType),
		SecondaryTypes: secondary,
		FirstRelease:   domain.ParseReleaseDate(d.FirstReleaseDate),
		ArtistCredit:   credits.String(),
		Credits:        credits,
	}, nil
}

// primaryType maps "Album", "EP", … to the stored lowercase type; an unknown one becomes "other".
func primaryType(s string) domain.PrimaryType {
	if s == "" {
		return ""
	}
	t := domain.PrimaryType(strings.ToLower(s))
	for _, known := range domain.PrimaryTypes {
		if t == known {
			return t
		}
	}
	return domain.PrimaryOther
}
