// Package email implements usecase.DigestMailer over SMTP: html/text templates
// rendered into a multipart/alternative message (docs/08-notifications.md §2.1).
package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"time"

	"music-tracker/backend/internal/domain"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string // "Music Tracker <tracker@example.com>"
	// AppURL is linked from emails.
	AppURL string
}

type Mailer struct {
	cfg  Config
	from *mail.Address
	now  func() time.Time
	// send delivers a raw message; replaced in tests.
	send func(ctx context.Context, from string, to string, msg []byte) error
}

func NewMailer(cfg Config) (*Mailer, error) {
	m := &Mailer{cfg: cfg, now: time.Now}
	if cfg.Host != "" {
		from, err := mail.ParseAddress(cfg.From)
		if err != nil {
			return nil, fmt.Errorf("SMTP_FROM %q: %w", cfg.From, err)
		}
		m.from = from
	}
	m.send = m.smtpSend
	return m, nil
}

var errNotConfigured = errors.New("SMTP is not configured (SMTP_HOST is empty)")

func (m *Mailer) SendDigest(ctx context.Context, to string, d domain.Digest) error {
	if m.from == nil {
		return errNotConfigured
	}
	data := newDigestData(d, m.cfg.AppURL)
	var text, html bytes.Buffer
	if err := textTemplates.ExecuteTemplate(&text, "digest.txt.tmpl", data); err != nil {
		return fmt.Errorf("render digest text: %w", err)
	}
	if err := htmlTemplates.ExecuteTemplate(&html, "digest.html.tmpl", data); err != nil {
		return fmt.Errorf("render digest html: %w", err)
	}
	return m.deliver(ctx, to, data.Subject, text.Bytes(), html.Bytes())
}

func (m *Mailer) SendTest(ctx context.Context, to string) error {
	if m.from == nil {
		return errNotConfigured
	}
	data := struct{ AppURL string }{m.cfg.AppURL}
	var text, html bytes.Buffer
	if err := textTemplates.ExecuteTemplate(&text, "test.txt.tmpl", data); err != nil {
		return fmt.Errorf("render test text: %w", err)
	}
	if err := htmlTemplates.ExecuteTemplate(&html, "test.html.tmpl", data); err != nil {
		return fmt.Errorf("render test html: %w", err)
	}
	return m.deliver(ctx, to, "Music Tracker: тестовое письмо", text.Bytes(), html.Bytes())
}

func (m *Mailer) deliver(ctx context.Context, to, subject string, text, html []byte) error {
	msg, err := m.buildMessage(to, subject, text, html)
	if err != nil {
		return err
	}
	if err := m.send(ctx, m.from.Address, to, msg); err != nil {
		return fmt.Errorf("smtp: %w", err)
	}
	return nil
}

// buildMessage renders an RFC 5322 multipart/alternative message: text first, HTML preferred.
func (m *Mailer) buildMessage(to, subject string, text, html []byte) ([]byte, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, part := range []struct {
		contentType string
		content     []byte
	}{{"text/plain; charset=utf-8", text}, {"text/html; charset=utf-8", html}} {
		w, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {part.contentType},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return nil, err
		}
		qp := quotedprintable.NewWriter(w)
		if _, err := qp.Write(part.content); err != nil {
			return nil, err
		}
		if err := qp.Close(); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	id := make([]byte, 12)
	_, _ = rand.Read(id)
	var msg bytes.Buffer
	for _, h := range [][2]string{
		{"From", m.from.String()},
		{"To", to},
		{"Subject", mime.QEncoding.Encode("utf-8", subject)},
		{"Date", m.now().Format(time.RFC1123Z)},
		{"Message-ID", "<" + hex.EncodeToString(id) + "@music-tracker>"},
		{"MIME-Version", "1.0"},
		{"Content-Type", "multipart/alternative; boundary=" + mw.Boundary()},
	} {
		fmt.Fprintf(&msg, "%s: %s\r\n", h[0], h[1])
	}
	msg.WriteString("\r\n")
	msg.Write(body.Bytes())
	return msg.Bytes(), nil
}

// smtpSend talks SMTP with a deadline: implicit TLS on port 465, otherwise STARTTLS
// when the server offers it; AUTH PLAIN only when a user is configured.
func (m *Mailer) smtpSend(ctx context.Context, from, to string, msg []byte) error {
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	tlsConfig := &tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}
	if m.cfg.Port == 465 {
		conn = tls.Client(conn, tlsConfig)
	}
	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = c.Close() }()

	if ok, _ := c.Extension("STARTTLS"); ok && m.cfg.Port != 465 {
		if err := c.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if m.cfg.User != "" {
		if err := c.Auth(smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
