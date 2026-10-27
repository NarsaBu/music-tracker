package listenbrainz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"music-tracker/backend/internal/domain"
)

const (
	userAgent                = "MusicTrackerTest/0.0 ( test@example.com )"
	enterShikari domain.MBID = "329f95db-f02e-40bf-a5f7-0dea11529ce3"
)

// A trimmed real response.
const similarBody = `[
 {"artist_mbid": "074e3847-f67f-49f9-81f1-8c8cea147e8e", "name": "Bring Me the Horizon", "comment": "", "type": "Group", "gender": null, "score": 1181, "reference_mbid": "329f95db-f02e-40bf-a5f7-0dea11529ce3"},
 {"artist_mbid": "f59c5520-5f46-4d2c-b2c4-822eabf53419", "name": "Linkin Park", "comment": "American rock band", "type": null, "gender": null, "score": 962, "reference_mbid": "329f95db-f02e-40bf-a5f7-0dea11529ce3"},
 {"artist_mbid": "not-an-mbid", "name": "Broken", "comment": "", "type": null, "gender": null, "score": 1, "reference_mbid": "329f95db-f02e-40bf-a5f7-0dea11529ce3"}
]`

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(Config{BaseURL: srv.URL + "/", UserAgent: userAgent})
}

func TestSimilarArtists_MapsRows(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/similar-artists/json" || q.Get("artist_mbids") != string(enterShikari) || q.Get("algorithm") != DefaultAlgorithm {
			t.Errorf("request = %s", r.URL)
		}
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(similarBody))
	})

	got, err := c.SimilarArtists(context.Background(), enterShikari)
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.SimilarArtist{
		{MBID: "074e3847-f67f-49f9-81f1-8c8cea147e8e", Name: "Bring Me the Horizon", Type: "Group", Score: 1181},
		{MBID: "f59c5520-5f46-4d2c-b2c4-822eabf53419", Name: "Linkin Park", Disambiguation: "American rock band", Score: 962},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v (a row with a bad MBID is skipped)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSimilarArtists_FailuresAreUpstreamUnavailable(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"stale algorithm": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) },
		"server error":    func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) },
		"html instead of json": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html></html>"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := newTestClient(t, handler).SimilarArtists(context.Background(), enterShikari)
			if !errors.Is(err, domain.ErrUpstreamUnavailable) {
				t.Errorf("err = %v, want ErrUpstreamUnavailable", err)
			}
		})
	}
}
