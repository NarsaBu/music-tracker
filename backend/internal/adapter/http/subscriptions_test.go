package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
	"music-tracker/backend/internal/usecase/artist"
)

const nirvana = "5b11f4ce-a62d-471e-81fc-a69a8278c7da"

type fakeArtistUseCases struct {
	searchIn    artist.SearchInput
	subscribeIn artist.SubscribeInput
	listIn      artist.ListInput
	created     bool
	err         error
}

func (f *fakeArtistUseCases) search(_ context.Context, in artist.SearchInput) ([]artist.Candidate, error) {
	f.searchIn = in
	return []artist.Candidate{{
		ArtistCandidate: domain.ArtistCandidate{
			Artist: domain.Artist{
				MBID: nirvana, Name: "Nirvana", SortName: "Nirvana", Type: "Group",
				LifeSpan: domain.LifeSpan{Begin: "1987", End: "1994-04-05", Ended: true},
			},
			Score: 100,
		},
		Subscribed: true,
	}}, f.err
}

func (f *fakeArtistUseCases) subscribe(_ context.Context, in artist.SubscribeInput) (artist.SubscribeResult, error) {
	f.subscribeIn = in
	return artist.SubscribeResult{SubscriptionView: sampleView(), Created: f.created}, f.err
}

func (f *fakeArtistUseCases) list(_ context.Context, in artist.ListInput) (artist.ListResult, error) {
	f.listIn = in
	return artist.ListResult{
		Items: []artist.SubscriptionView{sampleView()}, Total: 7,
		Facets: usecase.SubscriptionFacets{Countries: []string{"US"}},
	}, f.err
}

func (f *fakeArtistUseCases) unsubscribe(_ context.Context, mbid string) error {
	if mbid != nirvana {
		return domain.ErrNotFound
	}
	return f.err
}

func sampleView() artist.SubscriptionView {
	created := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	return artist.SubscriptionView{
		Subscription: domain.Subscription{
			Artist:     domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana", Country: "US"},
			BaselineAt: created, CreatedAt: created,
		},
		Filter: domain.ResolvedFilter{
			PrimaryTypes:           domain.Overridable[[]domain.PrimaryType]{Value: []domain.PrimaryType{"album"}, Overridden: true},
			ExcludedSecondaryTypes: domain.Overridable[[]string]{Value: []string{"Live"}},
			IncludeFeatured:        domain.Overridable[bool]{Value: true, Overridden: true},
		},
	}
}

// funcs adapt plain functions to the use case interfaces.
type (
	searchFunc      func(context.Context, artist.SearchInput) ([]artist.Candidate, error)
	subscribeFunc   func(context.Context, artist.SubscribeInput) (artist.SubscribeResult, error)
	unsubscribeFunc func(context.Context, string) error
	listFunc        func(context.Context, artist.ListInput) (artist.ListResult, error)
)

func (f searchFunc) Execute(ctx context.Context, in artist.SearchInput) ([]artist.Candidate, error) {
	return f(ctx, in)
}

func (f subscribeFunc) Execute(ctx context.Context, in artist.SubscribeInput) (artist.SubscribeResult, error) {
	return f(ctx, in)
}
func (f unsubscribeFunc) Execute(ctx context.Context, mbid string) error { return f(ctx, mbid) }
func (f listFunc) Execute(ctx context.Context, in artist.ListInput) (artist.ListResult, error) {
	return f(ctx, in)
}

func newArtistRouter(f *fakeArtistUseCases) http.Handler {
	return NewRouter(Deps{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Readiness: fakePinger{},
		Artists: ArtistUseCases{
			Search:      searchFunc(f.search),
			Subscribe:   subscribeFunc(f.subscribe),
			Unsubscribe: unsubscribeFunc(f.unsubscribe),
			List:        listFunc(f.list),
		},
	})
}

func do(h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, r))
	return rec
}

func TestSearchArtists_Response(t *testing.T) {
	f := &fakeArtistUseCases{}
	rec := do(newArtistRouter(f), http.MethodGet, "/api/v1/catalog/artists?q=nirvana&limit=5", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if f.searchIn != (artist.SearchInput{Query: "nirvana", Limit: 5}) {
		t.Errorf("input = %+v", f.searchIn)
	}
	want := `{"items":[{"mbid":"` + nirvana + `","name":"Nirvana","sort_name":"Nirvana","disambiguation":"",` +
		`"type":"Group","country":null,"life_span":{"begin":"1987","end":"1994-04-05","ended":true},` +
		`"score":100,"subscribed":true}]}`
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("body =\n%s\nwant\n%s", got, want)
	}
}

func TestSearchArtists_BadLimit(t *testing.T) {
	rec := do(newArtistRouter(&fakeArtistUseCases{}), http.MethodGet, "/api/v1/catalog/artists?q=x&limit=ten", "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestSearchArtists_UpstreamDown(t *testing.T) {
	f := &fakeArtistUseCases{err: domain.ErrUpstreamUnavailable}
	rec := do(newArtistRouter(f), http.MethodGet, "/api/v1/catalog/artists?q=x", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestListSubscriptions_Response(t *testing.T) {
	f := &fakeArtistUseCases{}
	rec := do(newArtistRouter(f), http.MethodGet,
		"/api/v1/subscriptions?q=nir&status=synced&type=Group&country=US&filter=own&sort=added&limit=20&offset=40", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	want := artist.ListInput{
		Search: "nir", Status: "synced", ArtistType: "Group", Country: "US", Filter: "own", Sort: "added",
		Limit: 20, Offset: 40,
	}
	if f.listIn != want {
		t.Errorf("input = %+v", f.listIn)
	}
	var body struct {
		Items []struct {
			Artist struct{ Country *string } `json:"artist"`
			Filter struct {
				PrimaryTypes struct {
					Value      []string `json:"value"`
					Overridden bool     `json:"overridden"`
				} `json:"primary_types"`
				ExcludedSecondaryTypes struct {
					Overridden bool `json:"overridden"`
				} `json:"excluded_secondary_types"`
			} `json:"filter"`
			LastSyncedAt *time.Time `json:"last_synced_at"`
		} `json:"items"`
		Total  int `json:"total"`
		Facets struct {
			Countries []string `json:"countries"`
			Types     []string `json:"types"`
		} `json:"facets"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 7 || len(body.Items) != 1 {
		t.Fatalf("page = %+v", body)
	}
	// types is an empty list, not null: the client can iterate it directly.
	if len(body.Facets.Countries) != 1 || body.Facets.Types == nil {
		t.Errorf("facets = %+v", body.Facets)
	}
	it := body.Items[0]
	if it.Artist.Country == nil || *it.Artist.Country != "US" || it.LastSyncedAt != nil {
		t.Errorf("item = %+v", it)
	}
	if !it.Filter.PrimaryTypes.Overridden || it.Filter.PrimaryTypes.Value[0] != "album" || it.Filter.ExcludedSecondaryTypes.Overridden {
		t.Errorf("filter = %+v", it.Filter)
	}
}

func TestSubscribe_StatusAndOverride(t *testing.T) {
	tests := []struct {
		name       string
		created    bool
		wantStatus int
	}{
		{"new subscription", true, http.StatusCreated},
		{"already subscribed", false, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeArtistUseCases{created: tt.created}
			body := `{"mbid":"` + nirvana + `","include_featured":true,"excluded_secondary_types":[],"primary_types":null}`
			rec := do(newArtistRouter(f), http.MethodPost, "/api/v1/subscriptions", body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tt.wantStatus, rec.Body)
			}
			o := f.subscribeIn.Override
			if f.subscribeIn.MBID != nirvana || o.IncludeFeatured == nil || !*o.IncludeFeatured {
				t.Errorf("input = %+v", f.subscribeIn)
			}
			if o.ExcludedSecondaryTypes == nil || len(o.ExcludedSecondaryTypes) != 0 {
				t.Errorf("[] must become an empty override, got %#v", o.ExcludedSecondaryTypes)
			}
			if o.PrimaryTypes != nil {
				t.Errorf("null must mean no override, got %#v", o.PrimaryTypes)
			}
		})
	}
}

func TestSubscribe_BadBody(t *testing.T) {
	tests := map[string]string{
		"not json":      `{`,
		"unknown field": `{"mbid":"x","foo":1}`,
		"wrong type":    `{"mbid":42}`,
		"trailing data": `{"mbid":"x"} {}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			rec := do(newArtistRouter(&fakeArtistUseCases{}), http.MethodPost, "/api/v1/subscriptions", body)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("status = %d, want 422", rec.Code)
			}
		})
	}
}

func TestUnsubscribe(t *testing.T) {
	h := newArtistRouter(&fakeArtistUseCases{})
	if rec := do(h, http.MethodDelete, "/api/v1/subscriptions/"+nirvana, ""); rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	other := "a74b1b7f-71a5-4011-9441-d0b5e4122711"
	if rec := do(h, http.MethodDelete, "/api/v1/subscriptions/"+other, ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown: status = %d, want 404", rec.Code)
	}
}
