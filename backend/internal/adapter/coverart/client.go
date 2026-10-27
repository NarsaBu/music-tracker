// Package coverart implements usecase.CoverArchive on top of the Cover Art
// Archive: front cover originals of releases and release groups.
package coverart

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif" // registers the decoder for DecodeConfig
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"music-tracker/backend/internal/domain"
)

// headerBytes is how much of an image is read to learn its size; JPEG headers
// can sit behind large EXIF blocks.
const headerBytes = 1 << 20

type Config struct {
	BaseURL   string // default https://coverartarchive.org
	UserAgent string
	// Timeout of metadata requests and probes, default 20s. Downloads are bounded
	// by their context only: originals reach tens of megabytes.
	Timeout time.Duration
}

// Client is safe for concurrent use.
type Client struct {
	meta      *http.Client
	download  *http.Client
	baseURL   string
	userAgent string
}

func NewClient(cfg Config) *Client {
	c := &Client{
		meta:      &http.Client{Timeout: cfg.Timeout},
		download:  &http.Client{},
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		userAgent: cfg.UserAgent,
	}
	if c.meta.Timeout == 0 {
		c.meta.Timeout = 20 * time.Second
	}
	if c.baseURL == "" {
		c.baseURL = "https://coverartarchive.org"
	}
	return c
}

type imagesJSON struct {
	Images []struct {
		Front bool   `json:"front"`
		Image string `json:"image"`
	} `json:"images"`
}

// ReleaseGroupFront returns the original URL of the release group's front cover.
func (c *Client) ReleaseGroupFront(ctx context.Context, rg domain.MBID) (string, error) {
	return c.front(ctx, "/release-group/"+string(rg))
}

// ReleaseFront returns the original URL of a release's front cover.
func (c *Client) ReleaseFront(ctx context.Context, release domain.MBID) (string, error) {
	return c.front(ctx, "/release/"+string(release))
}

// front returns domain.ErrNotFound when there is no front image.
func (c *Client) front(ctx context.Context, path string) (string, error) {
	resp, err := c.get(ctx, c.meta, c.baseURL+path)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	var body imagesJSON
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("%w: coverart: decode %s: %w", domain.ErrUpstreamUnavailable, path, err)
	}
	for _, img := range body.Images {
		if img.Front && img.Image != "" {
			return img.Image, nil
		}
	}
	return "", fmt.Errorf("coverart %s: no front image: %w", path, domain.ErrNotFound)
}

// Probe reads the start of an image to learn its size and type.
func (c *Client) Probe(ctx context.Context, url string) (domain.CoverImage, error) {
	resp, err := c.get(ctx, c.meta, url)
	if err != nil {
		return domain.CoverImage{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	// A connection cut after the header still tells the size.
	head, readErr := io.ReadAll(io.LimitReader(resp.Body, headerBytes))
	cfg, format, err := image.DecodeConfig(bytes.NewReader(head))
	if err != nil && readErr != nil {
		return domain.CoverImage{}, fmt.Errorf("%w: coverart: read %s: %w", domain.ErrUpstreamUnavailable, url, readErr)
	}
	if err != nil {
		return domain.CoverImage{}, fmt.Errorf("coverart %s: unknown image format: %w", url, err)
	}
	return domain.CoverImage{
		URL: url, Width: cfg.Width, Height: cfg.Height, ContentType: "image/" + format, Size: max(resp.ContentLength, 0),
	}, nil
}

// Open starts the download of an image; the caller closes the body.
func (c *Client) Open(ctx context.Context, url string) (io.ReadCloser, error) {
	resp, err := c.get(ctx, c.download, url)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// get returns a 200 response; 404 → domain.ErrNotFound, other failures → ErrUpstreamUnavailable.
func (c *Client) get(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: coverart: %w", domain.ErrUpstreamUnavailable, err)
	}
	switch {
	case resp.StatusCode == http.StatusOK:
		return resp, nil
	case resp.StatusCode == http.StatusNotFound:
		_ = resp.Body.Close()
		return nil, fmt.Errorf("coverart %s: %w", url, domain.ErrNotFound)
	default:
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%w: coverart %s: status %d", domain.ErrUpstreamUnavailable, url, resp.StatusCode)
	}
}
