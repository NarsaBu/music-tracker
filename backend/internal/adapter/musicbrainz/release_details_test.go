package musicbrainz

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
)

const (
	detailsRG  = "9ba6f467-d94e-40d2-9bb6-1ee5e38c7830"
	rocky      = "25b7b584-d952-4662-a8b9-dd8cdfbfeb64"
	guestID    = "f4fdbb4c-e4b7-47a0-b83b-d91bbfcfa387"
	original   = "11111111-1111-4111-8111-111111111111"
	deluxe     = "22222222-2222-4222-8222-222222222222"
	rgResponse = `{
 "genres": [{"name": "hip hop", "count": 3}, {"name": "cloud rap", "count": 5}],
 "artist-credit": [{"name": "A$AP Rocky", "joinphrase": "", "artist": {"id": "` + rocky + `"}}],
 "releases": [
  {"id": "` + deluxe + `", "title": "ALL $MILE$ (deluxe)", "status": "Official", "date": "2026-02-01", "country": "XW"},
  {"id": "` + original + `", "title": "ALL $MILE$", "status": "Official", "date": "2026-01-16", "country": "US"}
 ]
}`
	releaseResponse = `{
 "media": [{"position": 1, "format": "Digital Media", "title": "", "tracks": [
  {"number": "1", "title": "GRADUALLY", "length": 185000,
   "artist-credit": [{"name": "A$AP Rocky", "joinphrase": "", "artist": {"id": "` + rocky + `"}}]},
  {"number": "2", "title": "DUET", "length": null,
   "artist-credit": [{"name": "A$AP Rocky", "joinphrase": " feat. ", "artist": {"id": "` + rocky + `"}},
                     {"name": "Guest", "joinphrase": "", "artist": {"id": "` + guestID + `"}}]}
 ]}]
}`
)

func TestReleaseGroupDetails_TracklistOfEarliestEdition(t *testing.T) {
	var paths []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path+"?inc="+r.URL.Query().Get("inc"))
		switch r.URL.Path {
		case "/ws/2/release-group/" + detailsRG:
			writeJSON(w, []byte(rgResponse))
		case "/ws/2/release/" + original:
			writeJSON(w, []byte(releaseResponse))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	got, err := c.ReleaseGroupDetails(context.Background(), detailsRG)
	if err != nil {
		t.Fatal(err)
	}
	wantPaths := []string{
		"/ws/2/release-group/" + detailsRG + "?inc=genres releases artist-credits",
		"/ws/2/release/" + original + "?inc=recordings artist-credits",
	}
	if !slices.Equal(paths, wantPaths) {
		t.Errorf("requests = %v, want %v", paths, wantPaths)
	}
	if !slices.Equal(got.Genres, []string{"cloud rap", "hip hop"}) {
		t.Errorf("genres = %v, want most voted first", got.Genres)
	}
	if len(got.Editions) != 2 || got.Editions[1].MBID != deluxe {
		t.Errorf("editions = %+v, want the original then the deluxe one", got.Editions)
	}
	if got.Edition.MBID != original || got.Edition.Country != "US" {
		t.Errorf("edition = %+v", got.Edition)
	}
	if len(got.Media) != 1 || len(got.Media[0].Tracks) != 2 || got.Media[0].Format != "Digital Media" {
		t.Fatalf("media = %+v", got.Media)
	}
	first, second := got.Media[0].Tracks[0], got.Media[0].Tracks[1]
	if first.Length != 185*time.Second || second.Length != 0 || second.ArtistCredit != "A$AP Rocky feat. Guest" {
		t.Errorf("tracks = %+v, %+v", first, second)
	}
	if len(got.Artists) != 1 || got.Artists[0].ArtistMBID != rocky {
		t.Errorf("artists = %+v", got.Artists)
	}
	if len(got.Guests) != 1 || got.Guests[0] != (domain.CreditName{ArtistMBID: guestID, Name: "Guest"}) {
		t.Errorf("guests = %+v", got.Guests)
	}
}

func TestReleaseGroupDetails_NoEditionYet(t *testing.T) {
	calls := 0
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeJSON(w, []byte(`{"genres": [], "artist-credit": [], "releases": []}`))
	})
	got, err := c.ReleaseGroupDetails(context.Background(), detailsRG)
	if err != nil || calls != 1 || len(got.Media) != 0 || got.Edition.MBID != "" {
		t.Errorf("got %+v, err = %v, calls = %d; want no tracklist and no second request", got, err, calls)
	}
}
