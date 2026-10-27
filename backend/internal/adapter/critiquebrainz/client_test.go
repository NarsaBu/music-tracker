package critiquebrainz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
)

const (
	userAgent              = "MusicTrackerTest/0.0 ( test@example.com )"
	okComputer domain.MBID = "b1392450-e666-3926-a536-22c65f834433"
)

// A trimmed real response.
const reviewsBody = `{
 "average_rating": {"count": 1, "rating": 5.0},
 "count": 2, "limit": 10, "offset": 0,
 "reviews": [
  {"id": "c508a8ec-1bd0-4f92-bcd8-581a2cd2e75f", "rating": null, "language": "en",
   "text": "I've never quite got my head around the fact that *OK Computer *is considered...",
   "created": "Wed, 25 Apr 2007 00:00:00 GMT", "published_on": "Wed, 25 Apr 2007 00:00:00 GMT",
   "full_name": "Creative Commons Attribution-NonCommercial-ShareAlike 3.0 Unported",
   "user": {"display_name": "Jon Lusk"}},
  {"id": "2b3c4d5e-0000-4000-8000-000000000000", "rating": 5, "language": "en",
   "text": "A masterpiece.", "created": "Mon, 01 Jan 2018 10:00:00 GMT", "published_on": null,
   "full_name": "Creative Commons Attribution-ShareAlike 3.0 Unported",
   "user": {"display_name": "listener"}}
 ]
}`

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(Config{BaseURL: srv.URL + "/ws/1/", UserAgent: userAgent})
}

func TestReleaseGroupReviews_MapsReviews(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/ws/1/review/" || q.Get("entity_id") != string(okComputer) || q.Get("entity_type") != "release_group" {
			t.Errorf("request = %s", r.URL)
		}
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte(reviewsBody))
	})

	got, err := c.ReleaseGroupReviews(context.Background(), okComputer)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 2 || got.Average != 5 || got.Rated != 1 || len(got.Items) != 2 {
		t.Fatalf("got %+v", got)
	}
	first := got.Items[0]
	if first.Author != "Jon Lusk" || first.Rating != 0 || strings.Contains(first.Text, "*") ||
		!first.PublishedAt.Equal(time.Date(2007, 4, 25, 0, 0, 0, 0, time.UTC)) || !strings.HasPrefix(first.License, "Creative Commons") {
		t.Errorf("first = %+v", first)
	}
	// No published_on: the creation time is used.
	if second := got.Items[1]; second.Rating != 5 || !second.PublishedAt.Equal(time.Date(2018, 1, 1, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("second = %+v", second)
	}
}

func TestReleaseGroupReviews_LongTextIsCut(t *testing.T) {
	if got := plainText(strings.Repeat("я", maxTextRunes+10)); len([]rune(got)) != maxTextRunes+1 || !strings.HasSuffix(got, "…") {
		t.Errorf("cut text has %d runes", len([]rune(got)))
	}
}

func TestReleaseGroupReviews_FailuresAreUpstreamUnavailable(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"server error": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) },
		"not json":     func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) },
	} {
		t.Run(name, func(t *testing.T) {
			_, err := newTestClient(t, handler).ReleaseGroupReviews(context.Background(), okComputer)
			if !errors.Is(err, domain.ErrUpstreamUnavailable) {
				t.Errorf("err = %v, want ErrUpstreamUnavailable", err)
			}
		})
	}
}
