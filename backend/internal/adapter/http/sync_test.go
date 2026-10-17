package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/artist"
)

type (
	getFunc      func(context.Context, string) (artist.SubscriptionView, error)
	syncOneFunc  func(context.Context, string) error
	releasesFunc func(context.Context, string, string) ([]domain.ArtistReleaseGroup, error)
	startFunc    func(context.Context, domain.SyncTrigger) (domain.SyncRun, bool, error)
	latestFunc   func(context.Context) (*domain.SyncRun, error)
)

func (f getFunc) Execute(ctx context.Context, m string) (artist.SubscriptionView, error) {
	return f(ctx, m)
}
func (f syncOneFunc) Execute(ctx context.Context, m string) error { return f(ctx, m) }
func (f releasesFunc) Execute(ctx context.Context, m, r string) ([]domain.ArtistReleaseGroup, error) {
	return f(ctx, m, r)
}

func (f startFunc) Execute(ctx context.Context, t domain.SyncTrigger) (domain.SyncRun, bool, error) {
	return f(ctx, t)
}
func (f latestFunc) Execute(ctx context.Context) (*domain.SyncRun, error) { return f(ctx) }

var started = time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)

func newSyncRouter(latest *domain.SyncRun) http.Handler {
	future := domain.ParseReleaseDate("2999-01")
	return NewRouter(Deps{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Readiness: fakePinger{},
		Artists: ArtistUseCases{
			Get: getFunc(func(_ context.Context, m string) (artist.SubscriptionView, error) {
				if m != nirvana {
					return artist.SubscriptionView{}, domain.ErrNotFound
				}
				return sampleView(), nil
			}),
			Sync: syncOneFunc(func(_ context.Context, m string) error {
				if m != nirvana {
					return domain.ErrNotFound
				}
				return nil
			}),
			ReleaseGroups: releasesFunc(func(_ context.Context, _ string, role string) ([]domain.ArtistReleaseGroup, error) {
				if role == "bad" {
					return nil, &domain.ValidationError{Fields: []domain.FieldError{{Field: "role", Message: "unknown"}}}
				}
				return []domain.ArtistReleaseGroup{{
					ReleaseGroup: domain.ReleaseGroup{
						MBID: "00000000-0000-0000-0000-000000000001", Title: "Soon",
						SecondaryTypes: []string{}, FirstRelease: future, ArtistCredit: "Nirvana feat. X",
					},
					Role: domain.RoleFeatured, DiscoveredAt: started,
				}}, nil
			}),
		},
		Sync: SyncUseCases{
			Start: startFunc(func(_ context.Context, trigger domain.SyncTrigger) (domain.SyncRun, bool, error) {
				return domain.SyncRun{ID: "r1", Trigger: trigger, StartedAt: started, ArtistsTotal: 3}, true, nil
			}),
			Latest: latestFunc(func(context.Context) (*domain.SyncRun, error) { return latest, nil }),
		},
	})
}

func TestGetSubscription(t *testing.T) {
	h := newSyncRouter(nil)
	if rec := do(h, http.MethodGet, "/api/v1/subscriptions/"+nirvana, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"Nirvana"`) {
		t.Errorf("status %d, body %s", rec.Code, rec.Body)
	}
	if rec := do(h, http.MethodGet, "/api/v1/subscriptions/a74b1b7f-71a5-4011-9441-d0b5e4122711", ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown: status %d", rec.Code)
	}
}

func TestSyncSubscription(t *testing.T) {
	h := newSyncRouter(nil)
	if rec := do(h, http.MethodPost, "/api/v1/subscriptions/"+nirvana+"/sync", ""); rec.Code != http.StatusAccepted {
		t.Errorf("status %d, want 202", rec.Code)
	}
	if rec := do(h, http.MethodPost, "/api/v1/subscriptions/a74b1b7f-71a5-4011-9441-d0b5e4122711/sync", ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown: status %d, want 404", rec.Code)
	}
}

func TestArtistReleaseGroups(t *testing.T) {
	h := newSyncRouter(nil)
	rec := do(h, http.MethodGet, "/api/v1/artists/"+nirvana+"/release-groups?role=featured", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	it := body.Items[0]
	if it["credit_role"] != "featured" || it["primary_type"] != nil || it["is_announcement"] != true ||
		it["first_release_date"] != "2999-01" || !strings.HasSuffix(it["cover_url"].(string), "/front-250") {
		t.Errorf("item = %v", it)
	}

	if rec := do(h, http.MethodGet, "/api/v1/artists/"+nirvana+"/release-groups?role=bad", ""); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("bad role: status %d", rec.Code)
	}
}

func TestStartSync(t *testing.T) {
	rec := do(newSyncRouter(nil), http.MethodPost, "/api/v1/sync", "")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"trigger":"manual"`) || !strings.Contains(rec.Body.String(), `"finished_at":null`) {
		t.Errorf("body = %s", rec.Body)
	}
}

func TestLatestSyncRun(t *testing.T) {
	if rec := do(newSyncRouter(nil), http.MethodGet, "/api/v1/sync/runs/latest", ""); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "null" {
		t.Errorf("none: status %d, body %s", rec.Code, rec.Body)
	}
	run := &domain.SyncRun{ID: "r1", Trigger: domain.TriggerSchedule, StartedAt: started, ArtistsTotal: 4, ArtistsDone: 4, FinishedAt: &started}
	rec := do(newSyncRouter(run), http.MethodGet, "/api/v1/sync/runs/latest", "")
	if !strings.Contains(rec.Body.String(), `"artists_done":4`) {
		t.Errorf("body = %s", rec.Body)
	}
}
