package domain

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// Settings is the singleton configuration of the instance (ADR-8).
type Settings struct {
	Filter              ReleaseFilter
	NotifyAnnouncements bool
	// NewReleaseDays: a release older than this, found only now, is not news (FR-3.4).
	NewReleaseDays int
	Email          EmailSettings
	RSS            RSSSettings
	LastDigestAt   *time.Time
	// NextDigestAt is nil while email is disabled.
	NextDigestAt *time.Time
}

type EmailSettings struct {
	Enabled bool
	To      string
	Digest  DigestSchedule
}

type RSSSettings struct {
	// Enabled controls the public feed endpoint. The web feed is always on.
	Enabled   bool
	FeedToken string
}

// NewReleaseDays bounds; the database checks the same range.
const (
	MinNewReleaseDays = 1
	MaxNewReleaseDays = 3650
)

// StaleThreshold is NewReleaseDays as a duration.
func (s Settings) StaleThreshold() time.Duration {
	return time.Duration(s.NewReleaseDays) * 24 * time.Hour
}

// EmailActive reports whether digests should be produced at all.
func (s Settings) EmailActive() bool { return s.Email.Enabled && s.Email.To != "" }

// Validate checks every user-editable field; field names follow PUT /settings.
func (s Settings) Validate() error {
	fields := s.Filter.Validate("filters.")
	if s.NewReleaseDays < MinNewReleaseDays || s.NewReleaseDays > MaxNewReleaseDays {
		fields = append(fields, FieldError{
			Field: "filters.new_release_days", Message: fmt.Sprintf("must be between %d and %d", MinNewReleaseDays, MaxNewReleaseDays),
		})
	}
	if s.Email.Enabled || s.Email.To != "" {
		if _, err := mail.ParseAddress(s.Email.To); err != nil || strings.ContainsAny(s.Email.To, "<> ") {
			fields = append(fields, FieldError{Field: "email.to", Message: "must be an email address"})
		}
	}
	fields = append(fields, s.Email.Digest.validate("email.digest.")...)
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

type DigestFrequency string

const (
	DigestDaily  DigestFrequency = "daily"
	DigestWeekly DigestFrequency = "weekly"
)

// DigestSchedule says when the email digest goes out: every day, or on one weekday,
// at a wall-clock time in the user's time zone.
type DigestSchedule struct {
	Frequency DigestFrequency
	Weekday   time.Weekday // weekly only
	Hour      int
	Minute    int
	Timezone  string // IANA name, e.g. "Europe/Moscow"
}

func (d DigestSchedule) validate(prefix string) []FieldError {
	var fields []FieldError
	if d.Frequency != DigestDaily && d.Frequency != DigestWeekly {
		fields = append(fields, FieldError{Field: prefix + "frequency", Message: "must be daily or weekly"})
	}
	if d.Weekday < time.Sunday || d.Weekday > time.Saturday {
		fields = append(fields, FieldError{Field: prefix + "weekday", Message: "must be between 0 (Sunday) and 6"})
	}
	if d.Hour < 0 || d.Hour > 23 || d.Minute < 0 || d.Minute > 59 {
		fields = append(fields, FieldError{Field: prefix + "time", Message: "must be HH:MM"})
	}
	if _, err := time.LoadLocation(d.Timezone); err != nil || d.Timezone == "" {
		fields = append(fields, FieldError{Field: prefix + "timezone", Message: fmt.Sprintf("unknown time zone %q", d.Timezone)})
	}
	return fields
}

// Next returns the first send time strictly after `after`, in UTC. Wall-clock times
// skipped by a DST jump are moved forward by the jump, as time.Date does.
func (d DigestSchedule) Next(after time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(d.Timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("digest time zone: %w", err)
	}
	local := after.In(loc)
	// 8 days cover a full week plus today.
	for i := range 8 {
		day := local.AddDate(0, 0, i)
		at := time.Date(day.Year(), day.Month(), day.Day(), d.Hour, d.Minute, 0, 0, loc)
		if !at.After(after) {
			continue
		}
		if d.Frequency == DigestWeekly && at.Weekday() != d.Weekday {
			continue
		}
		return at.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("digest schedule %+v has no next time", d)
}
