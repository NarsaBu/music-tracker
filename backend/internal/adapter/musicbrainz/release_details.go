package musicbrainz

import (
	"cmp"
	"context"
	"fmt"
	"net/url"
	"slices"
	"time"

	"music-tracker/backend/internal/domain"
)

type genreDTO struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type editionDTO struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Date    string `json:"date"`
	Country string `json:"country"`
}

type releaseGroupLookupDTO struct {
	Genres       []genreDTO   `json:"genres"`
	ArtistCredit []creditDTO  `json:"artist-credit"`
	Releases     []editionDTO `json:"releases"`
}

type releaseLookupDTO struct {
	Media []struct {
		Position int    `json:"position"`
		Format   string `json:"format"`
		Title    string `json:"title"`
		Tracks   []struct {
			Number       string      `json:"number"`
			Title        string      `json:"title"`
			Length       *int        `json:"length"` // ms
			ArtistCredit []creditDTO `json:"artist-credit"`
		} `json:"tracks"`
	} `json:"media"`
}

// ReleaseGroupDetails implements usecase.ReleaseCatalog: genres and credit of the
// release group, then the tracklist of its earliest official edition — two requests.
func (c *Client) ReleaseGroupDetails(ctx context.Context, rg domain.MBID) (domain.ReleaseDetails, error) {
	var group releaseGroupLookupDTO
	q := url.Values{"inc": {"genres releases artist-credits"}}
	if err := c.get(ctx, "/release-group/"+string(rg), q, &group); err != nil {
		return domain.ReleaseDetails{}, err
	}

	genres := slices.Clone(group.Genres)
	slices.SortStableFunc(genres, func(a, b genreDTO) int { return cmp.Compare(b.Count, a.Count) })
	out := domain.ReleaseDetails{Genres: make([]string, 0, len(genres))}
	for _, g := range genres {
		out.Genres = append(out.Genres, g.Name)
	}
	credit := creditsOf(group.ArtistCredit)
	out.Artists = domain.UniqueArtists(credit)

	editions := make([]domain.Edition, 0, len(group.Releases))
	for _, r := range group.Releases {
		id, err := domain.ParseMBID(r.ID)
		if err != nil {
			continue
		}
		editions = append(editions, domain.Edition{
			MBID: id, Title: r.Title, Status: r.Status, Country: r.Country, Date: domain.ParseReleaseDate(r.Date),
		})
	}
	edition, ok := domain.PickEdition(editions)
	if !ok {
		return out, nil // an announcement may have no release yet
	}
	out.Edition = edition

	var release releaseLookupDTO
	if err := c.get(ctx, "/release/"+string(edition.MBID), url.Values{"inc": {"recordings artist-credits"}}, &release); err != nil {
		return domain.ReleaseDetails{}, fmt.Errorf("release %s: %w", edition.MBID, err)
	}
	for _, m := range release.Media {
		medium := domain.Medium{Position: m.Position, Format: m.Format, Title: m.Title}
		for _, t := range m.Tracks {
			credits := creditsOf(t.ArtistCredit)
			track := domain.Track{Number: t.Number, Title: t.Title, ArtistCredit: credits.String(), Credits: credits}
			if t.Length != nil {
				track.Length = time.Duration(*t.Length) * time.Millisecond
			}
			medium.Tracks = append(medium.Tracks, track)
		}
		out.Media = append(out.Media, medium)
	}
	out.Guests = domain.GuestsOf(credit, out.Media)
	return out, nil
}
