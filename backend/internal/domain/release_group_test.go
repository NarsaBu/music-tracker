package domain

import (
	"testing"
	"time"
)

func TestParseReleaseDate(t *testing.T) {
	tests := []struct {
		raw     string
		wantDay string // "" = nil
	}{
		{"2024-05-17", "2024-05-17"},
		{"2024-05", "2024-05-01"},
		{"2024", "2024-01-01"},
		{"", ""},
		{"2024-13", ""},
		{"20xx", ""},
		{"2024-5-1", ""},
	}
	for _, tt := range tests {
		d := ParseReleaseDate(tt.raw)
		got := ""
		if d.Day != nil {
			got = d.Day.Format("2006-01-02")
		}
		if got != tt.wantDay || d.Raw != tt.raw {
			t.Errorf("ParseReleaseDate(%q) = {%q, %s}, want day %s", tt.raw, d.Raw, got, tt.wantDay)
		}
	}
}

func TestReleaseDate_StaleAndAnnouncement(t *testing.T) {
	now := time.Date(2026, 10, 17, 6, 0, 0, 0, time.UTC)
	threshold := 90 * 24 * time.Hour
	tests := []struct {
		raw              string
		stale, announced bool
	}{
		{"2026-10-17", false, false},
		{"2026-08-01", false, false}, // 77 days ago
		{"2026-07-01", true, false},  // 108 days ago
		{"2026", true, false},        // normalized to Jan 1 — older than 90 days
		{"2026-11-07", false, true},
		{"", false, false},
	}
	for _, tt := range tests {
		d := ParseReleaseDate(tt.raw)
		if got := d.IsStale(now, threshold); got != tt.stale {
			t.Errorf("%q IsStale = %t, want %t", tt.raw, got, tt.stale)
		}
		if got := d.IsAnnouncement(now); got != tt.announced {
			t.Errorf("%q IsAnnouncement = %t, want %t", tt.raw, got, tt.announced)
		}
	}
}
