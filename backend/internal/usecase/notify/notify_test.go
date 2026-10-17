package notify

import (
	"context"
	"errors"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase"
	"music-tracker/backend/internal/usecase/memory"
)

const (
	artistA domain.MBID = "5b11f4ce-a62d-471e-81fc-a69a8278c7da"
	artistB domain.MBID = "a74b1b7f-71a5-4011-9441-d0b5e4122711"
)

var now = time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)

func rgID(n int) domain.MBID {
	return domain.MBID("00000000-0000-0000-0000-00000000000" + string(rune('0'+n)))
}

func link(artist domain.MBID, n int, pt domain.PrimaryType, date string, role domain.CreditRole) domain.ArtistReleaseGroup {
	return domain.ArtistReleaseGroup{
		ArtistMBID: artist, Role: role,
		ReleaseGroup: domain.ReleaseGroup{MBID: rgID(n), Title: "RG", PrimaryType: pt, FirstRelease: domain.ParseReleaseDate(date)},
	}
}

func newStore() *memory.Store {
	s := memory.NewStore()
	s.Artists[artistA] = domain.Artist{MBID: artistA, Name: "A"}
	s.Artists[artistB] = domain.Artist{MBID: artistB, Name: "B"}
	s.Subs[artistA] = domain.Subscription{}
	s.Subs[artistB] = domain.Subscription{}
	return s
}

func fanout(s *memory.Store) *FanOut {
	return NewFanOut(s.SettingsRepo(), s.SubscriptionRepo(), s.NotificationRepo(), &memory.Clock{T: now})
}

func channels(s *memory.Store, rg domain.MBID) map[domain.Channel]domain.NotificationStatus {
	out := map[domain.Channel]domain.NotificationStatus{}
	for _, n := range s.Notifs {
		if n.ReleaseGroup == rg {
			out[n.Channel] = n.Status
		}
	}
	return out
}

func TestFanOut_FiltersAndChannels(t *testing.T) {
	s := newStore()
	s.Settings.Email = domain.EmailSettings{Enabled: true, To: "me@example.com", Digest: s.Settings.Email.Digest}
	links := []domain.ArtistReleaseGroup{
		link(artistA, 1, domain.PrimaryAlbum, "2026-10-10", domain.RoleMain),
		link(artistA, 2, domain.PrimaryBroadcast, "2026-10-10", domain.RoleMain),   // type not allowed
		link(artistA, 3, domain.PrimarySingle, "2026-10-10", domain.RoleFeatured),  // featured off
		link(artistA, 4, domain.PrimaryEP, "2026-12-01", domain.RoleCollaboration), // announcement
	}

	n, err := fanout(s).Execute(context.Background(), artistA, links)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("matched = %d, want 2", n)
	}
	if got := channels(s, rgID(1)); got[domain.ChannelRSS] != domain.StatusSent || got[domain.ChannelEmail] != domain.StatusPending {
		t.Errorf("rg1 channels = %v, want rss sent + email pending", got)
	}
	if len(channels(s, rgID(2))) != 0 || len(channels(s, rgID(3))) != 0 {
		t.Error("filtered releases must not be notified")
	}
	if len(channels(s, rgID(4))) != 2 {
		t.Error("announcements are notified by default")
	}
}

func TestFanOut_OverrideAnnouncementsAndDisabledEmail(t *testing.T) {
	s := newStore()
	s.Settings.NotifyAnnouncements = false
	on := true
	s.Subs[artistA] = domain.Subscription{Override: domain.FilterOverride{IncludeFeatured: &on}}

	n, err := fanout(s).Execute(context.Background(), artistA, []domain.ArtistReleaseGroup{
		link(artistA, 1, domain.PrimarySingle, "2026-10-10", domain.RoleFeatured),
		link(artistA, 2, domain.PrimaryAlbum, "2027-01", domain.RoleMain),
	})
	if err != nil || n != 1 {
		t.Fatalf("matched = %d, %v; want the featured single only", n, err)
	}
	if got := channels(s, rgID(1)); len(got) != 1 || got[domain.ChannelRSS] != domain.StatusSent {
		t.Errorf("channels = %v, want rss only while email is off", got)
	}
}

func TestFanOut_OneNotificationPerReleaseMainWins(t *testing.T) {
	s := newStore()
	on := true
	s.Subs[artistB] = domain.Subscription{Override: domain.FilterOverride{IncludeFeatured: &on}}
	uc := fanout(s)

	// B is featured on A's album and is processed first; then A, the main artist.
	if _, err := uc.Execute(context.Background(), artistB, []domain.ArtistReleaseGroup{link(artistB, 1, domain.PrimaryAlbum, "2026-10-10", domain.RoleFeatured)}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Execute(context.Background(), artistA, []domain.ArtistReleaseGroup{link(artistA, 1, domain.PrimaryAlbum, "2026-10-10", domain.RoleMain)}); err != nil {
		t.Fatal(err)
	}
	if len(s.Notifs) != 1 || s.Notifs[0].Artist != artistA || s.Notifs[0].Role != domain.RoleMain {
		t.Errorf("notifications = %+v, want one, found via the main artist", s.Notifs)
	}
}

func TestFanOut_UnsubscribedArtistNotifiesNothing(t *testing.T) {
	s := newStore()
	delete(s.Subs, artistA)
	n, err := fanout(s).Execute(context.Background(), artistA, []domain.ArtistReleaseGroup{link(artistA, 1, domain.PrimaryAlbum, "", domain.RoleMain)})
	if err != nil || n != 0 || len(s.Notifs) != 0 {
		t.Errorf("n = %d, err = %v, notifs = %d", n, err, len(s.Notifs))
	}
}

type fakeMailer struct {
	digests []domain.Digest
	tests   []string
	err     error
}

func (m *fakeMailer) SendDigest(_ context.Context, _ string, d domain.Digest) error {
	if m.err != nil {
		return m.err
	}
	m.digests = append(m.digests, d)
	return nil
}

func (m *fakeMailer) SendTest(_ context.Context, to string) error {
	m.tests = append(m.tests, to)
	return m.err
}

// digestStore has email on, a digest due now and two pending email items.
func digestStore(t *testing.T) *memory.Store {
	t.Helper()
	s := newStore()
	due := now
	s.Settings.Email = domain.EmailSettings{Enabled: true, To: "me@example.com", Digest: domain.DigestSchedule{Frequency: domain.DigestDaily, Hour: 9, Timezone: "UTC"}}
	s.Settings.NextDigestAt = &due
	for i, l := range []domain.ArtistReleaseGroup{
		link(artistA, 1, domain.PrimaryAlbum, "2026-10-10", domain.RoleMain),
		link(artistA, 2, domain.PrimaryAlbum, "2026-12-01", domain.RoleMain),
	} {
		s.Releases[l.ReleaseGroup.MBID] = l.ReleaseGroup
		if _, err := fanout(s).Execute(context.Background(), artistA, []domain.ArtistReleaseGroup{l}); err != nil {
			t.Fatal(i, err)
		}
	}
	return s
}

func sendDigest(s *memory.Store, m *fakeMailer) *SendDigest {
	return NewSendDigest(s.SettingsRepo(), s.NotificationRepo(), m, memory.TxManager{}, &memory.Clock{T: now})
}

func TestScheduleDigest(t *testing.T) {
	s := digestStore(t)
	uc := NewScheduleDigest(s.SettingsRepo(), s.Queue(), &memory.Clock{T: now})
	for range 2 {
		if err := uc.Execute(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.Jobs) != 1 || s.Jobs[0].Kind != usecase.JobSendDigest {
		t.Errorf("jobs = %+v, want one send_digest", s.Jobs)
	}

	later := now.Add(time.Hour)
	s.Settings.NextDigestAt = &later
	s.Jobs = nil
	_ = uc.Execute(context.Background())
	if len(s.Jobs) != 0 {
		t.Error("a digest in the future must not be queued")
	}
}

func TestSendDigest_SendsAndReschedules(t *testing.T) {
	s := digestStore(t)
	m := &fakeMailer{}

	sent, err := sendDigest(s, m).Execute(context.Background(), now, false)
	if err != nil {
		t.Fatal(err)
	}
	if sent != 2 || len(m.digests) != 1 {
		t.Fatalf("sent = %d, digests = %d", sent, len(m.digests))
	}
	if d := m.digests[0]; len(d.Released) != 1 || len(d.Announcements) != 1 {
		t.Errorf("digest = %+v", d)
	}
	if got := channels(s, rgID(1))[domain.ChannelEmail]; got != domain.StatusSent {
		t.Errorf("email status = %s, want sent", got)
	}
	wantNext := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	if s.Settings.NextDigestAt == nil || !s.Settings.NextDigestAt.Equal(wantNext) || s.Settings.LastDigestAt == nil {
		t.Errorf("next = %v, last = %v", s.Settings.NextDigestAt, s.Settings.LastDigestAt)
	}

	// The same slot again (a duplicate job) does nothing.
	if sent, _ := sendDigest(s, m).Execute(context.Background(), now, false); sent != 0 || len(m.digests) != 1 {
		t.Error("an outdated slot must not send again")
	}
}

func TestSendDigest_NothingNewSendsNothing(t *testing.T) {
	s := newStore()
	due := now
	s.Settings.Email = domain.EmailSettings{Enabled: true, To: "me@example.com", Digest: domain.DigestSchedule{Frequency: domain.DigestDaily, Hour: 9, Timezone: "UTC"}}
	s.Settings.NextDigestAt = &due
	m := &fakeMailer{}

	if _, err := sendDigest(s, m).Execute(context.Background(), now, false); err != nil {
		t.Fatal(err)
	}
	if len(m.digests) != 0 || s.Settings.NextDigestAt.Equal(now) || s.Settings.LastDigestAt != nil {
		t.Errorf("digests = %d, next = %v, last = %v; want no email, schedule moved", len(m.digests), s.Settings.NextDigestAt, s.Settings.LastDigestAt)
	}
}

func TestSendDigest_SMTPFailure(t *testing.T) {
	s := digestStore(t)
	m := &fakeMailer{err: errors.New("connection refused")}

	_, err := sendDigest(s, m).Execute(context.Background(), now, false)
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Fatalf("error = %v", err)
	}
	if !s.Settings.NextDigestAt.Equal(now) {
		t.Error("a retried digest keeps its slot")
	}
	for _, n := range s.Notifs {
		if n.Channel == domain.ChannelEmail && (n.Status != domain.StatusPending || n.Attempts != 1) {
			t.Errorf("notification = %+v, want pending with 1 attempt", n)
		}
	}

	_, _ = sendDigest(s, m).Execute(context.Background(), now, true)
	if s.Settings.NextDigestAt.Equal(now) {
		t.Error("after the last attempt the schedule must move on")
	}
}

func TestSendTestEmail(t *testing.T) {
	s := newStore()
	m := &fakeMailer{}
	uc := NewSendTestEmail(s.SettingsRepo(), m)
	if err := uc.Execute(context.Background()); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("no address: %v, want ErrValidation", err)
	}
	s.Settings.Email.To = "me@example.com"
	if err := uc.Execute(context.Background()); err != nil || len(m.tests) != 1 {
		t.Errorf("err = %v, tests = %v", err, m.tests)
	}
	m.err = errors.New("auth failed")
	if err := uc.Execute(context.Background()); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("smtp error: %v, want ErrUpstreamUnavailable", err)
	}
}

func TestBuildFeed(t *testing.T) {
	s := digestStore(t)
	uc := NewBuildFeed(s.SettingsRepo(), s.NotificationRepo())

	items, err := uc.Execute(context.Background(), "token")
	if err != nil || len(items) != 2 {
		t.Fatalf("items = %d, err = %v", len(items), err)
	}
	if _, err := uc.Execute(context.Background(), "wrong"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("wrong token: %v", err)
	}
	s.Settings.RSS.Enabled = false
	if _, err := uc.Execute(context.Background(), "token"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("disabled feed: %v", err)
	}
}

func TestListReleases_Paging(t *testing.T) {
	s := newStore()
	for i := 1; i <= 5; i++ {
		l := link(artistA, i, domain.PrimaryAlbum, "2026-10-10", domain.RoleMain)
		s.Releases[l.ReleaseGroup.MBID] = l.ReleaseGroup
		at := now.Add(time.Duration(i) * time.Minute)
		uc := NewFanOut(s.SettingsRepo(), s.SubscriptionRepo(), s.NotificationRepo(), &memory.Clock{T: at})
		if _, err := uc.Execute(context.Background(), artistA, []domain.ArtistReleaseGroup{l}); err != nil {
			t.Fatal(err)
		}
	}
	uc := NewListReleases(s.NotificationRepo())

	var seen []domain.MBID
	cursor := ""
	for page := 0; page < 5; page++ {
		res, err := uc.Execute(context.Background(), ListReleasesInput{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range res.Items {
			seen = append(seen, it.ReleaseGroup.MBID)
		}
		if cursor = res.NextCursor; cursor == "" {
			break
		}
	}
	want := []domain.MBID{rgID(5), rgID(4), rgID(3), rgID(2), rgID(1)}
	if len(seen) != len(want) {
		t.Fatalf("seen = %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("seen = %v, want %v", seen, want)
			break
		}
	}

	for _, in := range []ListReleasesInput{{Cursor: "%%%"}, {Type: "lp"}, {Role: "guest"}, {Limit: 1000}} {
		if _, err := uc.Execute(context.Background(), in); !errors.Is(err, domain.ErrValidation) {
			t.Errorf("%+v: %v, want ErrValidation", in, err)
		}
	}
}
