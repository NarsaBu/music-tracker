package email

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
	"time"

	"music-tracker/backend/internal/domain"
)

type sent struct {
	from, to string
	msg      []byte
}

func newTestMailer(t *testing.T) (*Mailer, *[]sent) {
	t.Helper()
	m, err := NewMailer(Config{Host: "smtp.test", Port: 1025, From: "Music Tracker <tracker@example.com>", AppURL: "https://tracker.example"})
	if err != nil {
		t.Fatal(err)
	}
	var out []sent
	m.send = func(_ context.Context, from, to string, msg []byte) error {
		out = append(out, sent{from, to, msg})
		return nil
	}
	m.now = func() time.Time { return time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC) }
	return m, &out
}

func item(artist, title string, pt domain.PrimaryType, date string, role domain.CreditRole) domain.NotificationItem {
	return domain.NotificationItem{
		FoundVia: domain.Artist{Name: artist},
		Role:     role,
		ReleaseGroup: domain.ReleaseGroup{
			MBID: "62b427b8-1c52-34e7-b250-da2aa3e44860", Title: title, PrimaryType: pt,
			FirstRelease: domain.ParseReleaseDate(date), ArtistCredit: artist,
		},
	}
}

// parts decodes a sent message into its subject and text/html bodies.
func parts(t *testing.T, raw []byte) (subject, text, html string) {
	t.Helper()
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	subject, err = new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		t.Fatal(err)
	}
	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/alternative" {
		t.Fatalf("content type %q, %v", mediaType, err)
	}
	r := multipart.NewReader(msg.Body, params["boundary"])
	for {
		p, err := r.NextPart() // decodes quoted-printable
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(p)
		if strings.HasPrefix(p.Header.Get("Content-Type"), "text/plain") {
			text = string(b)
		} else {
			html = string(b)
		}
	}
	return subject, text, html
}

func TestSendDigest(t *testing.T) {
	m, out := newTestMailer(t)
	now := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	d := domain.NewDigest(domain.DigestDaily, []domain.NotificationItem{
		item("Sigur Rós", "Later", domain.PrimaryAlbum, "2026-10-10", domain.RoleMain),
		item("Sigur Rós", "Earlier", domain.PrimarySingle, "2026-10-01", domain.RoleMain),
		item("Björk", "Soon <EP>", domain.PrimaryEP, "2026-12-01", domain.RoleMain),
		item("Radiohead", "Guest", domain.PrimarySingle, "2026-10-05", domain.RoleFeatured),
	}, now)

	if err := m.SendDigest(context.Background(), "me@example.com", d); err != nil {
		t.Fatal(err)
	}
	if len(*out) != 1 || (*out)[0].from != "tracker@example.com" || (*out)[0].to != "me@example.com" {
		t.Fatalf("sent = %+v", *out)
	}
	subject, text, html := parts(t, (*out)[0].msg)

	if subject != "Новые релизы: Sigur Rós, Björk и ещё 1" {
		t.Errorf("subject = %q", subject)
	}
	for _, want := range []string{"== Вышло ==", "== Анонсы ==", "== Гостевое участие ==", "Soon <EP> (EP), 2026-12-01"} {
		if !strings.Contains(text, want) {
			t.Errorf("text has no %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "Earlier") > strings.Index(text, "Later") {
		t.Error("releases of an artist must be sorted by date")
	}
	if !strings.Contains(html, "Soon &lt;EP&gt;") || strings.Contains(html, "Soon <EP>") {
		t.Error("html must escape titles")
	}
	if !strings.Contains(html, "https://coverartarchive.org/release-group/62b427b8-1c52-34e7-b250-da2aa3e44860/front-250") {
		t.Error("html must show covers")
	}
}

func TestSubject(t *testing.T) {
	now := time.Now()
	one := domain.NewDigest(domain.DigestWeekly, []domain.NotificationItem{item("Björk", "X", domain.PrimaryAlbum, "", domain.RoleMain)}, now)
	if got := newDigestData(one, "").Subject; got != "Новые релизы за неделю: Björk" {
		t.Errorf("subject = %q", got)
	}
}

func TestSendTest(t *testing.T) {
	m, out := newTestMailer(t)
	if err := m.SendTest(context.Background(), "me@example.com"); err != nil {
		t.Fatal(err)
	}
	subject, text, html := parts(t, (*out)[0].msg)
	if subject != "Music Tracker: тестовое письмо" || !strings.Contains(text, "тестовое письмо") || !strings.Contains(html, "https://tracker.example") {
		t.Errorf("subject %q\ntext %q\nhtml %q", subject, text, html)
	}
}

func TestNotConfigured(t *testing.T) {
	m, err := NewMailer(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SendTest(context.Background(), "me@example.com"); !errors.Is(err, errNotConfigured) {
		t.Errorf("error = %v", err)
	}
	if _, err := NewMailer(Config{Host: "smtp.test", From: "not an address"}); err == nil {
		t.Error("a bad SMTP_FROM must fail at startup")
	}
}
