package musicbrainz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
)

const userAgent = "MusicTrackerTest/0.0 ( test@example.com )"

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// newTestClient returns a client with fast limits and backoff against handler.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(Config{
		BaseURL:     srv.URL + "/ws/2/",
		UserAgent:   userAgent,
		RateLimit:   1000,
		BaseBackoff: time.Millisecond,
	})
}

func writeJSON(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func TestGetArtist_MapsLookup(t *testing.T) {
	body := fixture(t, "lookup_nirvana.json")
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ws/2/artist/5b11f4ce-a62d-471e-81fc-a69a8278c7da" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("fmt") != "json" {
			t.Errorf("fmt = %q, want json", r.URL.Query().Get("fmt"))
		}
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		writeJSON(w, body)
	})

	got, err := c.GetArtist(context.Background(), "5b11f4ce-a62d-471e-81fc-a69a8278c7da")
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Artist{
		MBID:           "5b11f4ce-a62d-471e-81fc-a69a8278c7da",
		Name:           "Nirvana",
		SortName:       "Nirvana",
		Disambiguation: "1980s–1990s US grunge band",
		Type:           "Group",
		Country:        "US",
		LifeSpan:       domain.LifeSpan{Begin: "1987", End: "1994-04-05", Ended: true},
	}
	if got != want {
		t.Errorf("artist =\n%+v\nwant\n%+v", got, want)
	}
}

func TestGetArtist_NotFound(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusBadRequest} {
		c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":"Invalid mbid."}`, status)
		})
		_, err := c.GetArtist(context.Background(), "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("status %d: error = %v, want ErrNotFound", status, err)
		}
	}
}

func TestSearchArtists_PhraseQuery(t *testing.T) {
	body := fixture(t, "search_nirvana.json")
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("query"); got != `artist:"nirvana"` {
			t.Errorf("query = %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "3" {
			t.Errorf("limit = %q", got)
		}
		writeJSON(w, body)
	})

	got, err := c.SearchArtists(context.Background(), "nirvana", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d candidates, want 3", len(got))
	}
	if got[0].Name != "Nirvana" || got[0].Score != 100 || got[0].Country != "US" {
		t.Errorf("first candidate = %+v", got[0])
	}
	for _, a := range got {
		if a.SortName == "" {
			t.Errorf("candidate %s has empty sort name", a.MBID)
		}
	}
}

func TestSearchArtists_FallsBackToPlainQuery(t *testing.T) {
	empty, found := fixture(t, "search_empty.json"), fixture(t, "search_nirvana.json")
	var queries []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query().Get("query"))
		if len(queries) == 1 {
			writeJSON(w, empty)
			return
		}
		writeJSON(w, found)
	})

	got, err := c.SearchArtists(context.Background(), "AC/DC", 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{`artist:"AC\/DC"`, `AC\/DC`}
	if len(queries) != 2 || queries[0] != want[0] || queries[1] != want[1] {
		t.Errorf("queries = %q, want %q", queries, want)
	}
	if len(got) != 3 {
		t.Errorf("got %d candidates, want 3 from the fallback", len(got))
	}
}

func TestRetry_RecoversFromTransientErrors(t *testing.T) {
	body := fixture(t, "lookup_nirvana.json")
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			writeJSON(w, body)
		}
	})

	if _, err := c.GetArtist(context.Background(), "5b11f4ce-a62d-471e-81fc-a69a8278c7da"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestRetry_ExhaustedIsUpstreamUnavailable(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	})

	_, err := c.GetArtist(context.Background(), "5b11f4ce-a62d-471e-81fc-a69a8278c7da")
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("error = %v, want ErrUpstreamUnavailable", err)
	}
	if calls.Load() != 4 {
		t.Errorf("calls = %d, want 4 attempts", calls.Load())
	}
}

func TestRetry_ClientErrorsAreNotRetried(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	})

	_, err := c.SearchArtists(context.Background(), "x", 1)
	if err == nil || errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("error = %v, want a plain error", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}

func TestRetry_StopsOnContextCancel(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.GetArtist(ctx, "5b11f4ce-a62d-471e-81fc-a69a8278c7da")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want DeadlineExceeded", err)
	}
	if time.Since(start) > time.Second {
		t.Error("client kept waiting after the context was done")
	}
}

func TestBackoff(t *testing.T) {
	c := NewClient(Config{BaseBackoff: time.Second, MaxRetryAfter: 10 * time.Second})
	for attempt, base := range map[int]time.Duration{1: time.Second, 2: 2 * time.Second, 3: 4 * time.Second} {
		got := c.backoff(attempt, nil)
		if got < base*3/4 || got > base*5/4 {
			t.Errorf("backoff(%d) = %v, want %v ±25%%", attempt, got, base)
		}
	}

	if got := c.backoff(1, &retryableError{wait: 5 * time.Second}); got != 5*time.Second {
		t.Errorf("Retry-After 5s: backoff = %v", got)
	}
	if got := c.backoff(1, &retryableError{wait: time.Hour}); got != 10*time.Second {
		t.Errorf("Retry-After 1h: backoff = %v, want capped 10s", got)
	}
}

func TestRateLimiter_SpacesRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []byte(`{"artists":[]}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL, UserAgent: userAgent, RateLimit: 20}) // 50ms apart

	start := time.Now()
	for range 3 {
		if _, err := c.searchArtists(context.Background(), "x", 1); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed < 90*time.Millisecond {
		t.Errorf("3 requests took %v, want ≥ 100ms at 20 rps", elapsed)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := map[string]time.Duration{"": 0, "3": 3 * time.Second, " 1 ": time.Second, "-1": 0, "Wed, 21 Oct 2015 07:28:00 GMT": 0}
	for in, want := range tests {
		if got := parseRetryAfter(in); got != want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestEscapeLucene(t *testing.T) {
	tests := map[string]string{
		"Radiohead":        "Radiohead",
		"AC/DC":            `AC\/DC`,
		`Sigur Rós`:        `Sigur Rós`,
		`"Weird Al" (Yan)`: `\"Weird Al\" \(Yan\)`,
		`a+b-c && d || !e`: `a\+b\-c \&\& d \|\| \!e`,
		`{}[]^~*?:\`:       `\{\}\[\]\^\~\*\?\:\\`,
	}
	for in, want := range tests {
		if got := escapeLucene(in); got != want {
			t.Errorf("escapeLucene(%q) = %q, want %q", in, got, want)
		}
	}
}
