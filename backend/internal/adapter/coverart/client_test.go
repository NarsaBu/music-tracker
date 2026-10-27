package coverart

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"music-tracker/backend/internal/domain"
)

const (
	userAgent              = "MusicTrackerTest/0.0 ( test@example.com )"
	okComputer domain.MBID = "b1392450-e666-3926-a536-22c65f834433"
)

func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewGray(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, string) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(Config{BaseURL: srv.URL, UserAgent: userAgent}), srv.URL
}

func TestFrontProbeAndOpen(t *testing.T) {
	big := pngOf(t, 1300, 1250)
	var srvURL string
	c, srvURL := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		switch r.URL.Path {
		case "/release-group/" + string(okComputer):
			_, _ = w.Write([]byte(`{"images": [
				{"front": false, "image": "` + srvURL + `/back.png"},
				{"front": true, "image": "` + srvURL + `/front.png"}]}`))
		case "/front.png":
			w.Header().Set("Content-Length", strconv.Itoa(len(big)))
			_, _ = w.Write(big)
		case "/release/00000000-0000-0000-0000-000000000001":
			_, _ = w.Write([]byte(`{"images": [{"front": false, "image": "x"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	ctx := context.Background()

	url, err := c.ReleaseGroupFront(ctx, okComputer)
	if err != nil || url != srvURL+"/front.png" {
		t.Fatalf("front = %q, %v", url, err)
	}
	img, err := c.Probe(ctx, url)
	if err != nil || img.Width != 1300 || img.Height != 1250 || img.ContentType != "image/png" || img.Size != int64(len(big)) {
		t.Errorf("probe = %+v, %v", img, err)
	}
	body, err := c.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(body)
	_ = body.Close()
	if !bytes.Equal(got, big) {
		t.Errorf("downloaded %d bytes, want the original %d", len(got), len(big))
	}

	if _, err := c.ReleaseFront(ctx, "00000000-0000-0000-0000-000000000001"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("no front image: %v, want ErrNotFound", err)
	}
	if _, err := c.ReleaseFront(ctx, "00000000-0000-0000-0000-000000000002"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("no cover art at all: %v, want ErrNotFound", err)
	}
}

func TestFailures(t *testing.T) {
	c, srvURL := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/not-an-image" {
			_, _ = w.Write([]byte("<html>"))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
	})
	if _, err := c.ReleaseGroupFront(context.Background(), okComputer); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("archive down: %v, want ErrUpstreamUnavailable", err)
	}
	if _, err := c.Probe(context.Background(), srvURL+"/not-an-image"); err == nil {
		t.Error("not an image must fail the probe")
	}
}
