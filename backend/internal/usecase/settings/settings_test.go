package settings

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/usecase/memory"
)

var now = time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)

var cfg = Config{PublicBaseURL: "https://tracker.example/"}

func input(s *memory.Store) UpdateInput {
	return UpdateInput{
		Filter:              s.Settings.Filter,
		NotifyAnnouncements: true,
		Email:               s.Settings.Email,
		RSSEnabled:          true,
	}
}

func TestGet_FeedURL(t *testing.T) {
	s := memory.NewStore()
	v, err := NewGet(s.SettingsRepo(), cfg).Execute(context.Background())
	if err != nil || v.FeedURL != "https://tracker.example/feeds/token.atom" {
		t.Errorf("view = %+v, %v", v, err)
	}
}

func TestUpdate_DigestScheduling(t *testing.T) {
	s := memory.NewStore()
	uc := NewUpdate(s.SettingsRepo(), &memory.Clock{T: now}, cfg)

	in := input(s)
	in.Email.Enabled, in.Email.To = true, "  me@example.com "
	v, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if v.Email.To != "me@example.com" {
		t.Errorf("address not trimmed: %q", v.Email.To)
	}
	want := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	if v.NextDigestAt == nil || !v.NextDigestAt.Equal(want) {
		t.Fatalf("next = %v, want %v", v.NextDigestAt, want)
	}

	// Unrelated change keeps the slot even if time moved on.
	later := NewUpdate(s.SettingsRepo(), &memory.Clock{T: now.Add(5 * time.Hour)}, cfg)
	in.NotifyAnnouncements = false
	if v, _ = later.Execute(context.Background(), in); !v.NextDigestAt.Equal(want) {
		t.Errorf("next = %v, want unchanged %v", v.NextDigestAt, want)
	}

	// A schedule change reschedules.
	in.Email.Digest = domain.DigestSchedule{Frequency: domain.DigestWeekly, Weekday: time.Friday, Hour: 9, Timezone: "UTC"}
	if v, _ = uc.Execute(context.Background(), in); !v.NextDigestAt.Equal(time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("weekly next = %v", v.NextDigestAt)
	}

	// Email off clears it.
	in.Email.Enabled = false
	if v, _ = uc.Execute(context.Background(), in); v.NextDigestAt != nil {
		t.Errorf("next = %v, want nil", v.NextDigestAt)
	}
}

func TestUpdate_Validation(t *testing.T) {
	s := memory.NewStore()
	in := input(s)
	in.Email.Enabled, in.Email.To = true, "nope"
	if _, err := NewUpdate(s.SettingsRepo(), &memory.Clock{T: now}, cfg).Execute(context.Background(), in); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error = %v, want ErrValidation", err)
	}
	if s.Settings.Email.To != "" {
		t.Error("invalid settings must not be saved")
	}
}

func TestRotateFeedToken(t *testing.T) {
	s := memory.NewStore()
	url, err := NewRotateFeedToken(s.SettingsRepo(), cfg).Execute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	token := s.Settings.RSS.FeedToken
	if token == "token" || len(token) != 43 || !strings.HasSuffix(url, "/feeds/"+token+".atom") {
		t.Errorf("token = %q, url = %q", token, url)
	}
}
