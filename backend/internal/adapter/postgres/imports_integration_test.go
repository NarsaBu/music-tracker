//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
)

func TestImportRepository_RoundTrip(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	repo := NewImportRepository(testPool)
	at := time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)
	yes := true

	job, items, err := repo.Create(ctx, "artists.csv", []domain.ImportLine{
		{LineNo: 2, Value: "Nirvana", IncludeFeatured: &yes},
		{LineNo: 3, Value: "https://musicbrainz.org/artist/" + string(nirvana), MBID: nirvana},
		{LineNo: 5, Value: "Zzqxj"},
	}, at)
	if err != nil {
		t.Fatal(err)
	}
	if job.Total != 3 || job.Counts[domain.ImportPending] != 3 || len(items) != 3 || items[1].ID == "" {
		t.Fatalf("job = %+v, items = %+v", job, items)
	}

	// Enqueue all items in one batch; a duplicate is skipped.
	queue := NewJobQueue(testPool)
	jobs := []usecase.Job{usecase.NewImportItemJob(items[0].ID, at), usecase.NewImportItemJob(items[1].ID, at), usecase.NewImportItemJob(items[0].ID, at)}
	if err := queue.EnqueueMany(ctx, jobs); err != nil {
		t.Fatal(err)
	}
	var queued int
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM jobs WHERE kind = 'import_item'").Scan(&queued); err != nil || queued != 2 {
		t.Errorf("queued = %d, %v", queued, err)
	}

	// The MBID line is subscribed: the item references the artist.
	if err := NewArtistRepository(testPool).Upsert(ctx, domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana"}); err != nil {
		t.Fatal(err)
	}
	err = NewTxManager(testPool).WithinTx(ctx, func(ctx context.Context) error {
		it, err := repo.GetItem(ctx, job.ID, items[1].ID)
		if err != nil {
			return err
		}
		if it.InputMBID != nirvana || it.Status != domain.ImportPending {
			t.Errorf("item = %+v", it)
		}
		it.Status, it.Artist = domain.ImportMatched, &domain.Artist{MBID: nirvana}
		return repo.UpdateItem(ctx, it)
	})
	if err != nil {
		t.Fatal(err)
	}
	review := items[0]
	review.Status = domain.ImportNeedsReview
	review.Candidates = []domain.ArtistCandidate{{
		Artist: domain.Artist{MBID: nirvana, Name: "Nirvana", SortName: "Nirvana", Disambiguation: "US grunge", Type: "Group",
			Country: "US", LifeSpan: domain.LifeSpan{Begin: "1987", End: "1994-04-05", Ended: true}},
		Score: 100,
	}}
	if err := repo.UpdateItem(ctx, review); err != nil {
		t.Fatal(err)
	}

	if err := repo.FinishIfDone(ctx, job.ID, at); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.Get(ctx, job.ID); got.Finished() {
		t.Error("finished with a pending line")
	}
	notFound := items[2]
	notFound.Status, notFound.Error = domain.ImportNotFound, "nothing"
	if err := repo.UpdateItem(ctx, notFound); err != nil {
		t.Fatal(err)
	}
	if err := repo.FinishIfDone(ctx, job.ID, at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Finished() || !got.FinishedAt.Equal(at.Add(time.Minute)) || got.Counts[domain.ImportPending] != 0 ||
		got.Counts[domain.ImportMatched] != 1 || got.Counts[domain.ImportNeedsReview] != 1 || got.Counts[domain.ImportNotFound] != 1 {
		t.Errorf("job = %+v", got)
	}

	all, err := repo.ListItems(ctx, job.ID, nil)
	if err != nil || len(all) != 3 || all[0].LineNo != 2 || all[2].LineNo != 5 {
		t.Fatalf("items = %+v, %v", all, err)
	}
	if all[1].Artist == nil || all[1].Artist.Name != "Nirvana" {
		t.Errorf("matched item artist = %+v", all[1].Artist)
	}
	if all[0].IncludeFeatured == nil || !*all[0].IncludeFeatured || all[0].Candidates[0].Artist != review.Candidates[0].Artist {
		t.Errorf("review item = %+v", all[0])
	}
	if all[2].Error != "nothing" || all[2].Candidates != nil {
		t.Errorf("not found item = %+v", all[2])
	}
	if only, _ := repo.ListItems(ctx, job.ID, []domain.ImportItemStatus{domain.ImportNeedsReview}); len(only) != 1 || only[0].ID != review.ID {
		t.Errorf("needs_review = %+v", only)
	}
	if two, _ := repo.ListItems(ctx, job.ID, []domain.ImportItemStatus{domain.ImportNotFound, domain.ImportNeedsReview}); len(two) != 2 {
		t.Errorf("needs_review,not_found = %+v", two)
	}

	list, err := repo.List(ctx, 10)
	if err != nil || len(list) != 1 || list[0].Counts[domain.ImportMatched] != 1 {
		t.Errorf("list = %+v, %v", list, err)
	}

	for _, tc := range []struct{ job, item string }{
		{"not-a-uuid", items[0].ID},
		{"00000000-0000-0000-0000-000000000000", items[0].ID},
		{job.ID, "00000000-0000-0000-0000-000000000000"},
	} {
		if _, err := repo.GetItem(ctx, tc.job, tc.item); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetItem(%s, %s) err = %v", tc.job, tc.item, err)
		}
	}
	if _, err := repo.Get(ctx, "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get(nope) err = %v", err)
	}
}
