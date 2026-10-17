package importer

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"music-tracker/backend/internal/adapter/importfile"
	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
	"music-tracker/backend/internal/usecase/artist"
	"music-tracker/backend/internal/usecase/memory"
)

const (
	nirvanaUS domain.MBID = "5b11f4ce-a62d-471e-81fc-a69a8278c7da"
	nirvanaUK domain.MBID = "9282c8b4-ca0b-4c6b-b7e3-4f7762dfc4d6"
	radiohead domain.MBID = "a74b1b7f-71a5-4011-9441-d0b5e4122711"
)

var now = time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)

// fakeCatalog answers searches by query and lookups by MBID, counting the calls.
type fakeCatalog struct {
	found    map[string][]domain.ArtistCandidate
	artists  map[domain.MBID]domain.Artist
	err      error
	searches []string
	lookups  []domain.MBID
}

func (f *fakeCatalog) SearchArtists(_ context.Context, q string, _ int) ([]domain.ArtistCandidate, error) {
	f.searches = append(f.searches, q)
	return f.found[q], f.err
}

func (f *fakeCatalog) GetArtist(_ context.Context, mbid domain.MBID) (domain.Artist, error) {
	f.lookups = append(f.lookups, mbid)
	if f.err != nil {
		return domain.Artist{}, f.err
	}
	a, ok := f.artists[mbid]
	if !ok {
		return domain.Artist{}, domain.ErrNotFound
	}
	return a, nil
}

func (f *fakeCatalog) ListReleaseGroups(context.Context, domain.MBID) ([]domain.ReleaseGroup, error) {
	return nil, nil
}

func card(mbid domain.MBID, name, disambiguation string) domain.Artist {
	return domain.Artist{MBID: mbid, Name: name, SortName: name, Disambiguation: disambiguation}
}

func newCatalog() *fakeCatalog {
	return &fakeCatalog{
		found: map[string][]domain.ArtistCandidate{
			"Radiohead": {{Artist: card(radiohead, "Radiohead", ""), Score: 100}},
			"Nirvana": {
				{Artist: card(nirvanaUS, "Nirvana", "90s US grunge band"), Score: 100},
				{Artist: card(nirvanaUK, "Nirvana", "60s band from the UK"), Score: 100},
			},
		},
		artists: map[domain.MBID]domain.Artist{nirvanaUS: card(nirvanaUS, "Nirvana", "90s US grunge band")},
	}
}

type harness struct {
	store   *memory.Store
	catalog *fakeCatalog
	create  *CreateImport
	process *ProcessItem
	resolve *ResolveItem
	items   *ListItems
}

func newHarness() *harness {
	store := memory.NewStore()
	catalog := newCatalog()
	clock := &memory.Clock{T: now}
	tx := memory.TxManager{}
	sub := artist.NewSubscribe(catalog, store.ArtistRepo(), store.SubscriptionRepo(), store, store.Queue(), tx, clock)
	return &harness{
		store:   store,
		catalog: catalog,
		create:  NewCreateImport(importfile.Parser{}, store.ImportRepo(), store.Queue(), tx, clock),
		process: NewProcessItem(catalog, store.ImportRepo(), sub, tx, clock),
		resolve: NewResolveItem(catalog, store.ImportRepo(), sub, tx),
		items:   NewListItems(store.ImportRepo()),
	}
}

// runImportJobs plays the worker: handles every queued import_item job once.
func (h *harness) runImportJobs(t *testing.T) {
	t.Helper()
	for _, j := range h.store.Jobs {
		if j.Kind != usecase.JobImportItem {
			continue
		}
		payload, _ := json.Marshal(j.Payload)
		if err := h.process.HandleJob(context.Background(), payload, false); err != nil {
			t.Fatalf("job %s: %v", j.DedupKey, err)
		}
	}
}

func (h *harness) item(t *testing.T, raw string) domain.ImportItem {
	t.Helper()
	for _, it := range h.store.Items {
		if it.RawValue == raw {
			return it
		}
	}
	t.Fatalf("no item %q", raw)
	return domain.ImportItem{}
}

func TestImport_EndToEnd(t *testing.T) {
	h := newHarness()
	ctx := context.Background()
	file := "name,mbid,include_featured\nRadiohead,,true\nNirvana,,\nZzqxj,,\n," + string(nirvanaUS) + ",\n"

	job, err := h.create.Execute(ctx, `C:\Users\me\artists.csv`, strings.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	if job.Filename != "artists.csv" || job.Total != 4 || job.Counts[domain.ImportPending] != 4 || job.Finished() {
		t.Errorf("job = %+v", job)
	}
	if len(h.store.Jobs) != 4 || h.store.Jobs[0].Kind != usecase.JobImportItem {
		t.Fatalf("queued jobs = %+v", h.store.Jobs)
	}

	h.runImportJobs(t)

	got, _ := h.store.ImportRepo().Get(ctx, job.ID)
	want := map[domain.ImportItemStatus]int{domain.ImportMatched: 2, domain.ImportNeedsReview: 1, domain.ImportNotFound: 1}
	for st, n := range want {
		if got.Counts[st] != n {
			t.Errorf("counts = %v, want %v", got.Counts, want)
			break
		}
	}
	if !got.Finished() {
		t.Error("import not finished although nothing is pending")
	}

	// Confident name: subscribed from the search result, with the CSV override.
	rh := h.item(t, "Radiohead")
	if rh.Status != domain.ImportMatched || rh.Artist == nil || rh.Artist.Name != "Radiohead" {
		t.Errorf("radiohead = %+v", rh)
	}
	if sub, ok := h.store.Subs[radiohead]; !ok || sub.Override.IncludeFeatured == nil || !*sub.Override.IncludeFeatured {
		t.Errorf("radiohead subscription = %+v, %t", sub, ok)
	}
	// MBID: looked up, not searched; the baseline sync is queued.
	if _, ok := h.store.Subs[nirvanaUS]; !ok {
		t.Error("MBID line not subscribed")
	}
	if strings.Join(h.catalog.searches, ",") != "Radiohead,Nirvana,Zzqxj" || len(h.catalog.lookups) != 1 {
		t.Errorf("searches = %v, lookups = %v", h.catalog.searches, h.catalog.lookups)
	}
	// Namesakes wait for the user with both candidates.
	nv := h.item(t, "Nirvana")
	if nv.Status != domain.ImportNeedsReview || len(nv.Candidates) != 2 {
		t.Errorf("nirvana = %+v", nv)
	}
	if h.item(t, "Zzqxj").Status != domain.ImportNotFound {
		t.Error("unknown name not marked not_found")
	}

	review, err := h.items.Execute(ctx, job.ID, "needs_review")
	if err != nil || len(review) != 1 || review[0].ID != nv.ID {
		t.Errorf("needs_review items = %+v, %v", review, err)
	}
	if open, _ := h.items.Execute(ctx, job.ID, "needs_review, not_found,"); len(open) != 2 {
		t.Errorf("needs_review,not_found items = %+v", open)
	}

	// The user picks the UK band: no catalog request, the card comes from the candidates.
	lookups := len(h.catalog.lookups)
	resolved, err := h.resolve.Execute(ctx, ResolveInput{JobID: job.ID, ItemID: nv.ID, MBID: string(nirvanaUK)})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Status != domain.ImportMatched || resolved.Artist.MBID != nirvanaUK || len(h.catalog.lookups) != lookups {
		t.Errorf("resolved = %+v, lookups %d → %d", resolved, lookups, len(h.catalog.lookups))
	}
	if h.store.Subs[nirvanaUK].Artist.Disambiguation != "60s band from the UK" {
		t.Errorf("subscribed artist = %+v", h.store.Subs[nirvanaUK].Artist)
	}

	// A resolved line cannot be resolved again.
	_, err = h.resolve.Execute(ctx, ResolveInput{JobID: job.ID, ItemID: nv.ID, Skip: true})
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("second resolve err = %v, want ErrConflict", err)
	}

	zz := h.item(t, "Zzqxj")
	skipped, err := h.resolve.Execute(ctx, ResolveInput{JobID: job.ID, ItemID: zz.ID, Skip: true})
	if err != nil || skipped.Status != domain.ImportSkipped {
		t.Errorf("skip = %+v, %v", skipped, err)
	}
}

func TestProcessItem_IsIdempotent(t *testing.T) {
	h := newHarness()
	ctx := context.Background()
	if _, err := h.create.Execute(ctx, "a.txt", strings.NewReader("Radiohead\n")); err != nil {
		t.Fatal(err)
	}
	h.runImportJobs(t)
	h.runImportJobs(t) // e.g. a job redelivered after a worker crash
	if len(h.catalog.searches) != 1 {
		t.Errorf("searches = %v, want one", h.catalog.searches)
	}
}

func TestProcessItem_RetriesThenFails(t *testing.T) {
	h := newHarness()
	ctx := context.Background()
	job, err := h.create.Execute(ctx, "a.txt", strings.NewReader("Radiohead\n"))
	if err != nil {
		t.Fatal(err)
	}
	h.catalog.err = domain.ErrUpstreamUnavailable
	payload, _ := json.Marshal(h.store.Jobs[0].Payload)

	if err := h.process.HandleJob(ctx, payload, false); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if it := h.item(t, "Radiohead"); it.Status != domain.ImportPending {
		t.Errorf("after a retryable failure status = %s, want pending", it.Status)
	}

	if err := h.process.HandleJob(ctx, payload, true); err == nil {
		t.Fatal("last attempt succeeded")
	}
	it := h.item(t, "Radiohead")
	if it.Status != domain.ImportFailed || !strings.Contains(it.Error, "upstream unavailable") {
		t.Errorf("after the last attempt item = %+v", it)
	}
	if got, _ := h.store.ImportRepo().Get(ctx, job.ID); !got.Finished() {
		t.Error("import with a failed line not finished")
	}

	// A failed line can still be resolved by hand, with an MBID link.
	h.catalog.err = nil
	res, err := h.resolve.Execute(ctx, ResolveInput{
		JobID: job.ID, ItemID: it.ID, MBID: "https://musicbrainz.org/artist/" + string(nirvanaUS),
	})
	if err != nil || res.Status != domain.ImportMatched || res.Error != "" {
		t.Errorf("resolve failed line = %+v, %v", res, err)
	}
}

func TestProcessItem_UnknownMBID(t *testing.T) {
	h := newHarness()
	if _, err := h.create.Execute(context.Background(), "a.txt", strings.NewReader(string(radiohead))); err != nil {
		t.Fatal(err)
	}
	h.runImportJobs(t)
	if it := h.item(t, string(radiohead)); it.Status != domain.ImportNotFound || it.Error == "" {
		t.Errorf("item = %+v", it)
	}
}

func TestCreateImport_RejectsBadFile(t *testing.T) {
	h := newHarness()
	_, err := h.create.Execute(context.Background(), "a.pdf", strings.NewReader("x"))
	if !errors.Is(err, domain.ErrValidation) || len(h.store.Imports) != 0 {
		t.Errorf("err = %v, imports = %d", err, len(h.store.Imports))
	}
}

func TestResolveItem_Validation(t *testing.T) {
	h := newHarness()
	ctx := context.Background()
	job, _ := h.create.Execute(ctx, "a.txt", strings.NewReader("Nirvana\n"))
	h.runImportJobs(t)
	id := h.item(t, "Nirvana").ID

	tests := []struct {
		name string
		in   ResolveInput
		want error
	}{
		{"neither", ResolveInput{JobID: job.ID, ItemID: id}, domain.ErrValidation},
		{"both", ResolveInput{JobID: job.ID, ItemID: id, MBID: string(radiohead), Skip: true}, domain.ErrValidation},
		{"unknown artist", ResolveInput{JobID: job.ID, ItemID: id, MBID: string(radiohead)}, domain.ErrValidation},
		{"other job", ResolveInput{JobID: "import-9", ItemID: id, Skip: true}, domain.ErrNotFound},
	}
	for _, tt := range tests {
		if _, err := h.resolve.Execute(ctx, tt.in); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
	}
	if _, err := h.items.Execute(ctx, job.ID, "bogus"); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("bad status filter err = %v", err)
	}
}
