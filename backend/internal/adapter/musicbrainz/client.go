// Package musicbrainz implements usecase.MusicCatalog on top of the MusicBrainz
// Web Service v2 (docs/07-musicbrainz.md): rate limiting, retries, JSON → domain.
package musicbrainz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"music-tracker/backend/internal/domain"
)

type Config struct {
	BaseURL   string
	UserAgent string
	// RateLimit is requests per second; MusicBrainz allows 1.
	RateLimit float64
	// Zero values below take the defaults.
	Timeout     time.Duration // per request, default 15s
	MaxAttempts int           // default 4
	BaseBackoff time.Duration // default 2s, doubled per attempt
	// MaxRetryAfter caps a server-requested pause, default 30s.
	MaxRetryAfter time.Duration
}

// Client is safe for concurrent use. Create one per process: the limiter must be shared (ADR-6).
type Client struct {
	http          *http.Client
	limiter       *rate.Limiter
	baseURL       string
	userAgent     string
	maxAttempts   int
	baseBackoff   time.Duration
	maxRetryAfter time.Duration
}

func NewClient(cfg Config) *Client {
	c := &Client{
		http:          &http.Client{Timeout: cfg.Timeout},
		limiter:       rate.NewLimiter(rate.Limit(cfg.RateLimit), 1),
		baseURL:       strings.TrimRight(cfg.BaseURL, "/"),
		userAgent:     cfg.UserAgent,
		maxAttempts:   cfg.MaxAttempts,
		baseBackoff:   cfg.BaseBackoff,
		maxRetryAfter: cfg.MaxRetryAfter,
	}
	if c.http.Timeout == 0 {
		c.http.Timeout = 15 * time.Second
	}
	if c.maxAttempts == 0 {
		c.maxAttempts = 4
	}
	if c.baseBackoff == 0 {
		c.baseBackoff = 2 * time.Second
	}
	if c.maxRetryAfter == 0 {
		c.maxRetryAfter = 30 * time.Second
	}
	return c
}

// retryableError is a failure worth repeating; wait is the server-requested pause, if any.
type retryableError struct {
	err  error
	wait time.Duration
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

// statusError is a non-retryable unexpected HTTP status.
type statusError struct{ code int }

func (e *statusError) Error() string { return fmt.Sprintf("musicbrainz: unexpected status %d", e.code) }

// get performs GET {baseURL}{path}?{query}&fmt=json and decodes the body into out.
// 404 → domain.ErrNotFound; retries exhausted → domain.ErrUpstreamUnavailable.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	if query == nil {
		query = url.Values{}
	}
	query.Set("fmt", "json")
	target := c.baseURL + path + "?" + query.Encode()

	var lastErr error
	for attempt := range c.maxAttempts {
		if attempt > 0 {
			if err := sleep(ctx, c.backoff(attempt, lastErr)); err != nil {
				return err
			}
		}
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}
		err := c.do(ctx, target, out)
		var retry *retryableError
		if !errors.As(err, &retry) {
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("%w: musicbrainz %s: %d attempts failed, last: %w",
		domain.ErrUpstreamUnavailable, path, c.maxAttempts, lastErr)
}

func (c *Client) do(ctx context.Context, target string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &retryableError{err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusOK:
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("musicbrainz: decode response: %w", err)
		}
		return nil
	case resp.StatusCode == http.StatusNotFound:
		return domain.ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError:
		return &retryableError{
			err:  fmt.Errorf("status %d", resp.StatusCode),
			wait: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	default:
		return &statusError{code: resp.StatusCode}
	}
}

// backoff is BaseBackoff·2^(attempt-1) with ±25% jitter, or Retry-After when the server asked for longer.
func (c *Client) backoff(attempt int, lastErr error) time.Duration {
	d := c.baseBackoff << (attempt - 1)
	d += time.Duration((rand.Float64() - 0.5) * 0.5 * float64(d))
	var retry *retryableError
	if errors.As(lastErr, &retry) && retry.wait > d {
		d = min(retry.wait, c.maxRetryAfter)
	}
	return d
}

// parseRetryAfter supports the delay-seconds form, the one MusicBrainz uses.
func parseRetryAfter(v string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || secs < 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
