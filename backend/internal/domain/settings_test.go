package domain

import (
	"errors"
	"testing"
	"time"
)

func TestReleaseFilter_Matches(t *testing.T) {
	f := ReleaseFilter{
		PrimaryTypes:           []PrimaryType{PrimaryAlbum, PrimaryEP},
		ExcludedSecondaryTypes: []string{"Compilation", "Live"},
	}
	album := ReleaseGroup{PrimaryType: PrimaryAlbum, SecondaryTypes: []string{}}
	tests := []struct {
		name string
		f    ReleaseFilter
		rg   ReleaseGroup
		role CreditRole
		want bool
	}{
		{"main album", f, album, RoleMain, true},
		{"collaboration always counts", f, album, RoleCollaboration, true},
		{"featured off", f, album, RoleFeatured, false},
		{"featured on", ReleaseFilter{PrimaryTypes: f.PrimaryTypes, IncludeFeatured: true}, album, RoleFeatured, true},
		{"type not allowed", f, ReleaseGroup{PrimaryType: PrimarySingle}, RoleMain, false},
		{"excluded secondary", f, ReleaseGroup{PrimaryType: PrimaryAlbum, SecondaryTypes: []string{"Soundtrack", "Live"}}, RoleMain, false},
		{"other secondary passes", f, ReleaseGroup{PrimaryType: PrimaryAlbum, SecondaryTypes: []string{"Soundtrack"}}, RoleMain, true},
		{"no primary type, other not allowed", f, ReleaseGroup{}, RoleMain, false},
		{"no primary type, other allowed", ReleaseFilter{PrimaryTypes: []PrimaryType{PrimaryOther}}, ReleaseGroup{}, RoleMain, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.Matches(tt.rg, tt.role); got != tt.want {
				t.Errorf("Matches = %t, want %t", got, tt.want)
			}
		})
	}
}

func validSettings() Settings {
	return Settings{
		Filter: ReleaseFilter{PrimaryTypes: []PrimaryType{PrimaryAlbum}, ExcludedSecondaryTypes: []string{"Live"}},
		Email: EmailSettings{Enabled: true, To: "me@example.com", Digest: DigestSchedule{
			Frequency: DigestWeekly, Weekday: time.Friday, Hour: 9, Timezone: "Europe/Moscow",
		}},
	}
}

func TestSettings_Validate(t *testing.T) {
	if err := validSettings().Validate(); err != nil {
		t.Fatalf("valid settings: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*Settings)
		field  string
	}{
		{"bad primary type", func(s *Settings) { s.Filter.PrimaryTypes = []PrimaryType{"lp"} }, "filters.primary_types[0]"},
		{"bad email", func(s *Settings) { s.Email.To = "not-an-email" }, "email.to"},
		{"display-name email", func(s *Settings) { s.Email.To = "Me <me@example.com>" }, "email.to"},
		{"empty email while enabled", func(s *Settings) { s.Email.To = "" }, "email.to"},
		{"bad frequency", func(s *Settings) { s.Email.Digest.Frequency = "hourly" }, "email.digest.frequency"},
		{"bad weekday", func(s *Settings) { s.Email.Digest.Weekday = 7 }, "email.digest.weekday"},
		{"bad hour", func(s *Settings) { s.Email.Digest.Hour = 24 }, "email.digest.time"},
		{"bad timezone", func(s *Settings) { s.Email.Digest.Timezone = "Mars/Olympus" }, "email.digest.timezone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := validSettings()
			tt.mutate(&s)
			var verr *ValidationError
			if err := s.Validate(); !errors.As(err, &verr) || verr.Fields[0].Field != tt.field {
				t.Errorf("error = %v, want field %s", err, tt.field)
			}
		})
	}

	off := validSettings()
	off.Email = EmailSettings{Digest: off.Email.Digest}
	if err := off.Validate(); err != nil {
		t.Errorf("disabled email without address must be valid: %v", err)
	}
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestDigestSchedule_Next(t *testing.T) {
	tests := []struct {
		name  string
		d     DigestSchedule
		after string
		want  string
	}{
		{"daily, later today", DigestSchedule{Frequency: DigestDaily, Hour: 9, Timezone: "UTC"},
			"2026-10-17T06:00:00Z", "2026-10-17T09:00:00Z"},
		{"daily, exactly at send time goes to tomorrow", DigestSchedule{Frequency: DigestDaily, Hour: 9, Timezone: "UTC"},
			"2026-10-17T09:00:00Z", "2026-10-18T09:00:00Z"},
		{"daily in Moscow (UTC+3)", DigestSchedule{Frequency: DigestDaily, Hour: 9, Timezone: "Europe/Moscow"},
			"2026-10-17T07:00:00Z", "2026-10-18T06:00:00Z"},
		{"local date differs from UTC date", DigestSchedule{Frequency: DigestDaily, Hour: 1, Timezone: "Asia/Tokyo"},
			"2026-10-17T15:30:00Z", "2026-10-17T16:00:00Z"},
		{"weekly Friday from Saturday", DigestSchedule{Frequency: DigestWeekly, Weekday: time.Friday, Hour: 9, Minute: 30, Timezone: "UTC"},
			"2026-10-17T10:00:00Z", "2026-10-23T09:30:00Z"},
		{"weekly, today but already passed → next week", DigestSchedule{Frequency: DigestWeekly, Weekday: time.Saturday, Hour: 9, Timezone: "UTC"},
			"2026-10-17T10:00:00Z", "2026-10-24T09:00:00Z"},
		// Europe/Berlin: 2026-03-29 02:00 → 03:00 (spring forward), 2026-10-25 03:00 → 02:00.
		{"before spring forward: offset +1", DigestSchedule{Frequency: DigestDaily, Hour: 9, Timezone: "Europe/Berlin"},
			"2026-03-28T09:00:00Z", "2026-03-29T07:00:00Z"},
		{"skipped wall-clock time moves forward", DigestSchedule{Frequency: DigestDaily, Hour: 2, Minute: 30, Timezone: "Europe/Berlin"},
			"2026-03-28T23:00:00Z", "2026-03-29T01:30:00Z"},
		{"after fall back: offset +1 again", DigestSchedule{Frequency: DigestDaily, Hour: 9, Timezone: "Europe/Berlin"},
			"2026-10-25T00:00:00Z", "2026-10-25T08:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.d.Next(mustTime(t, tt.after))
			if err != nil {
				t.Fatal(err)
			}
			if want := mustTime(t, tt.want); !got.Equal(want) || got.Location() != time.UTC {
				t.Errorf("Next = %v, want %v (UTC)", got, want)
			}
		})
	}

	if _, err := (DigestSchedule{Frequency: DigestDaily, Timezone: "Nowhere/Land"}).Next(time.Now()); err == nil {
		t.Error("unknown zone must fail")
	}
}

func TestNewDigest(t *testing.T) {
	now := mustTime(t, "2026-10-17T06:00:00Z")
	items := []NotificationItem{
		{ID: "out", Role: RoleMain, ReleaseGroup: ReleaseGroup{FirstRelease: ParseReleaseDate("2026-10-10")}},
		{ID: "soon", Role: RoleCollaboration, ReleaseGroup: ReleaseGroup{FirstRelease: ParseReleaseDate("2026-11-07")}},
		{ID: "guest", Role: RoleFeatured, ReleaseGroup: ReleaseGroup{FirstRelease: ParseReleaseDate("2026-11-07")}},
		{ID: "undated", Role: RoleMain},
	}
	d := NewDigest(DigestDaily, items, now)
	if len(d.Released) != 2 || len(d.Announcements) != 1 || len(d.Featured) != 1 || d.Len() != 4 {
		t.Errorf("digest = %+v", d)
	}
	if d.Featured[0].ID != "guest" || d.Announcements[0].ID != "soon" {
		t.Errorf("sections = %+v", d)
	}
}
