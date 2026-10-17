package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/artist"
	"music-tracker/backend/internal/usecase/notify"
	settingsuc "music-tracker/backend/internal/usecase/settings"
)

type (
	settingsGetFunc    func(context.Context) (settingsuc.View, error)
	settingsUpdateFunc func(context.Context, settingsuc.UpdateInput) (settingsuc.View, error)
	rotateFunc         func(context.Context) (string, error)
	testEmailFunc      func(context.Context) error
	releasesListFunc   func(context.Context, notify.ListReleasesInput) (notify.ReleasesPage, error)
	feedFunc           func(context.Context, string) ([]domain.NotificationItem, error)
	updateSubFunc      func(context.Context, artist.UpdateSubscriptionInput) (artist.SubscriptionView, error)
)

func (f settingsGetFunc) Execute(ctx context.Context) (settingsuc.View, error) { return f(ctx) }
func (f settingsUpdateFunc) Execute(ctx context.Context, in settingsuc.UpdateInput) (settingsuc.View, error) {
	return f(ctx, in)
}
func (f rotateFunc) Execute(ctx context.Context) (string, error) { return f(ctx) }
func (f testEmailFunc) Execute(ctx context.Context) error        { return f(ctx) }
func (f releasesListFunc) Execute(ctx context.Context, in notify.ListReleasesInput) (notify.ReleasesPage, error) {
	return f(ctx, in)
}
func (f feedFunc) Execute(ctx context.Context, token string) ([]domain.NotificationItem, error) {
	return f(ctx, token)
}
func (f updateSubFunc) Execute(ctx context.Context, in artist.UpdateSubscriptionInput) (artist.SubscriptionView, error) {
	return f(ctx, in)
}

var next = time.Date(2026, 10, 23, 6, 0, 0, 0, time.UTC)

func sampleSettings() settingsuc.View {
	return settingsuc.View{
		Settings: domain.Settings{
			Filter:              domain.ReleaseFilter{PrimaryTypes: []domain.PrimaryType{"album"}},
			NotifyAnnouncements: true,
			Email: domain.EmailSettings{Enabled: true, To: "me@example.com", Digest: domain.DigestSchedule{
				Frequency: domain.DigestWeekly, Weekday: time.Friday, Hour: 9, Minute: 5, Timezone: "Europe/Moscow",
			}},
			RSS:          domain.RSSSettings{Enabled: true},
			NextDigestAt: &next,
		},
		FeedURL: "https://tracker.example/feeds/tok.atom",
	}
}

func sampleItem() domain.NotificationItem {
	return domain.NotificationItem{
		ID: "00000000-0000-0000-0000-0000000000aa", Role: domain.RoleFeatured, CreatedAt: next,
		FoundVia: domain.Artist{MBID: nirvana, Name: "Nirvana"},
		ReleaseGroup: domain.ReleaseGroup{
			MBID: "00000000-0000-0000-0000-000000000001", Title: "Duet", PrimaryType: domain.PrimarySingle,
			FirstRelease: domain.ParseReleaseDate("2999-01-01"), ArtistCredit: "X feat. Nirvana",
		},
	}
}

type notifyFakes struct {
	updateIn   settingsuc.UpdateInput
	releasesIn notify.ListReleasesInput
	patchIn    artist.UpdateSubscriptionInput
	testErr    error
	nextCursor string
}

func newNotifyRouter(f *notifyFakes) http.Handler {
	return NewRouter(Deps{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Readiness: fakePinger{},
		AppURL:    "https://tracker.example",
		Settings: SettingsUseCases{
			Get: settingsGetFunc(func(context.Context) (settingsuc.View, error) { return sampleSettings(), nil }),
			Update: settingsUpdateFunc(func(_ context.Context, in settingsuc.UpdateInput) (settingsuc.View, error) {
				f.updateIn = in
				return sampleSettings(), nil
			}),
			Rotate:    rotateFunc(func(context.Context) (string, error) { return "https://tracker.example/feeds/new.atom", nil }),
			TestEmail: testEmailFunc(func(context.Context) error { return f.testErr }),
		},
		Releases: releasesListFunc(func(_ context.Context, in notify.ListReleasesInput) (notify.ReleasesPage, error) {
			f.releasesIn = in
			return notify.ReleasesPage{Items: []domain.NotificationItem{sampleItem()}, NextCursor: f.nextCursor}, nil
		}),
		Feeds: feedFunc(func(_ context.Context, token string) ([]domain.NotificationItem, error) {
			if token != "tok" {
				return nil, domain.ErrNotFound
			}
			return []domain.NotificationItem{sampleItem()}, nil
		}),
		Artists: ArtistUseCases{
			Update: updateSubFunc(func(_ context.Context, in artist.UpdateSubscriptionInput) (artist.SubscriptionView, error) {
				f.patchIn = in
				return sampleView(), nil
			}),
		},
	})
}

func TestSettings_GetAndPut(t *testing.T) {
	f := &notifyFakes{}
	h := newNotifyRouter(f)

	rec := do(h, http.MethodGet, "/api/v1/settings", "")
	var got settingsDTO
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Email.Digest.Time != "09:05" || got.Email.Digest.Weekday != 5 || got.RSS.FeedURL == "" ||
		got.Filters.ExcludedSecondaryTypes == nil || got.Email.NextDigestAt == nil {
		t.Errorf("settings = %+v", got)
	}

	// The GET body is a valid PUT body.
	body, _ := json.Marshal(got)
	if rec := do(h, http.MethodPut, "/api/v1/settings", string(body)); rec.Code != http.StatusOK {
		t.Fatalf("put: %d %s", rec.Code, rec.Body)
	}
	if d := f.updateIn.Email.Digest; d.Hour != 9 || d.Minute != 5 || d.Weekday != time.Friday || d.Timezone != "Europe/Moscow" {
		t.Errorf("digest input = %+v", d)
	}

	for name, b := range map[string]string{
		"bad time":         strings.Replace(string(body), `"09:05"`, `"9am"`, 1),
		"no primary types": strings.Replace(string(body), `"primary_types":["album"]`, `"primary_types":null`, 1),
	} {
		if rec := do(h, http.MethodPut, "/api/v1/settings", b); rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: status %d", name, rec.Code)
		}
	}
}

func TestSettings_TestEmailAndRotate(t *testing.T) {
	f := &notifyFakes{}
	h := newNotifyRouter(f)
	if rec := do(h, http.MethodPost, "/api/v1/settings/email/test", ""); rec.Code != http.StatusNoContent {
		t.Errorf("test email: %d", rec.Code)
	}
	f.testErr = errors.Join(domain.ErrUpstreamUnavailable, errors.New("smtp"))
	if rec := do(h, http.MethodPost, "/api/v1/settings/email/test", ""); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("smtp down: %d", rec.Code)
	}
	if rec := do(h, http.MethodPost, "/api/v1/settings/feed-token/rotate", ""); !strings.Contains(rec.Body.String(), `"feed_url":"https://tracker.example/feeds/new.atom"`) {
		t.Errorf("rotate: %s", rec.Body)
	}
	if rec := do(h, http.MethodGet, "/api/v1/meta/release-types", ""); !strings.Contains(rec.Body.String(), `"Audio drama"`) {
		t.Errorf("meta: %s", rec.Body)
	}
}

func TestPatchSubscription_OptionalFields(t *testing.T) {
	f := &notifyFakes{}
	h := newNotifyRouter(f)
	rec := do(h, http.MethodPatch, "/api/v1/subscriptions/"+nirvana, `{"primary_types":null,"include_featured":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body)
	}
	in := f.patchIn
	if !in.PrimaryTypes.Set || in.PrimaryTypes.Value != nil {
		t.Errorf("null must reset: %+v", in.PrimaryTypes)
	}
	if !in.IncludeFeatured.Set || in.IncludeFeatured.Value == nil || !*in.IncludeFeatured.Value {
		t.Errorf("include_featured = %+v", in.IncludeFeatured)
	}
	if in.ExcludedSecondaryTypes.Set {
		t.Error("an absent field must stay unchanged")
	}
	if rec := do(h, http.MethodPatch, "/api/v1/subscriptions/"+nirvana, `{"include_featured":"yes"}`); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong type: %d", rec.Code)
	}
}

func TestReleases(t *testing.T) {
	f := &notifyFakes{nextCursor: "abc"}
	rec := do(newNotifyRouter(f), http.MethodGet, "/api/v1/releases?type=single&role=featured&cursor=xyz&limit=20", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if f.releasesIn != (notify.ListReleasesInput{Cursor: "xyz", Type: "single", Role: "featured", Limit: 20}) {
		t.Errorf("input = %+v", f.releasesIn)
	}
	var body struct {
		Items []struct {
			FoundVia       foundViaDTO `json:"found_via"`
			IsAnnouncement bool        `json:"is_announcement"`
			CoverURL       string      `json:"cover_url"`
		} `json:"items"`
		NextCursor *string `json:"next_cursor"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	it := body.Items[0]
	if it.FoundVia.Name != "Nirvana" || it.FoundVia.CreditRole != domain.RoleFeatured || !it.IsAnnouncement || body.NextCursor == nil || *body.NextCursor != "abc" {
		t.Errorf("body = %+v", body)
	}
}

func TestFeeds(t *testing.T) {
	h := newNotifyRouter(&notifyFakes{})

	rec := do(h, http.MethodGet, "/feeds/tok.atom", "")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/atom+xml") ||
		!strings.Contains(rec.Body.String(), "urn:musicbrainz:release-group:") {
		t.Fatalf("atom: %d %s", rec.Code, rec.Header())
	}
	etag := rec.Header().Get("ETag")
	if etag == "" || rec.Header().Get("Last-Modified") == "" || rec.Header().Get("Cache-Control") != "max-age=900" {
		t.Errorf("headers = %v", rec.Header())
	}

	req := httptest.NewRequest(http.MethodGet, "/feeds/tok.atom", nil)
	req.Header.Set("If-None-Match", etag)
	cached := httptest.NewRecorder()
	h.ServeHTTP(cached, req)
	if cached.Code != http.StatusNotModified || cached.Body.Len() != 0 {
		t.Errorf("conditional GET: %d", cached.Code)
	}

	if rec := do(h, http.MethodGet, "/feeds/tok.rss", ""); !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/rss+xml") {
		t.Errorf("rss content type = %s", rec.Header().Get("Content-Type"))
	}
	for _, path := range []string{"/feeds/wrong.atom", "/feeds/tok.xml"} {
		if rec := do(h, http.MethodGet, path, ""); rec.Code != http.StatusNotFound {
			t.Errorf("%s: %d", path, rec.Code)
		}
	}
}
