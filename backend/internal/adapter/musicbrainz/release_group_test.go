package musicbrainz

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
)

const sigurRos = "f6f2326f-6b25-4170-b89d-e235b25508e8"

func TestListReleaseGroups_ReadsAllPages(t *testing.T) {
	pages := map[string][]byte{"0": fixture(t, "rg_page1.json"), "3": fixture(t, "rg_page2.json")}
	var offsets []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/ws/2/release-group" || q.Get("artist") != sigurRos || q.Get("inc") != "artist-credits" {
			t.Errorf("request = %s", r.URL)
		}
		offsets = append(offsets, q.Get("offset"))
		writeJSON(w, pages[q.Get("offset")])
	})
	c.pageSize = 3

	got, err := c.ListReleaseGroups(context.Background(), sigurRos)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 || len(offsets) != 2 {
		t.Fatalf("got %d groups in %d requests (%v), want 6 in 2", len(got), len(offsets), offsets)
	}

	first := got[0]
	if first.Title != "Ágætis byrjun" || first.PrimaryType != domain.PrimaryAlbum || first.FirstRelease.Raw != "1999-06" {
		t.Errorf("first = %+v", first)
	}
	if first.FirstRelease.Day == nil || !first.FirstRelease.Day.Equal(time.Date(1999, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("first release day = %v", first.FirstRelease.Day)
	}
	if first.ArtistCredit != "Sigur Rós" || len(first.Credits) != 1 || first.Credits[0].ArtistMBID != sigurRos {
		t.Errorf("credit = %q %+v", first.ArtistCredit, first.Credits)
	}
	if first.SecondaryTypes == nil {
		t.Error("secondary types must be an empty slice, not nil")
	}
}

func TestListReleaseGroups_FailedPageFailsAll(t *testing.T) {
	page1 := fixture(t, "rg_page1.json")
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") == "0" {
			writeJSON(w, page1)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	c.pageSize = 3

	got, err := c.ListReleaseGroups(context.Background(), sigurRos)
	if !errors.Is(err, domain.ErrUpstreamUnavailable) || got != nil {
		t.Errorf("got %d groups, err %v; want nothing and ErrUpstreamUnavailable", len(got), err)
	}
}

func TestPrimaryType(t *testing.T) {
	tests := map[string]domain.PrimaryType{
		"Album": domain.PrimaryAlbum, "EP": domain.PrimaryEP, "Single": domain.PrimarySingle,
		"Broadcast": domain.PrimaryBroadcast, "Other": domain.PrimaryOther, "Weird": domain.PrimaryOther, "": "",
	}
	for in, want := range tests {
		if got := primaryType(in); got != want {
			t.Errorf("primaryType(%q) = %q, want %q", in, got, want)
		}
	}
}
