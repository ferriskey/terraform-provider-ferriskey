// Package client is a thin, typed HTTP client for the FerrisKey REST API. It
// owns OAuth2 authentication (password and client_credentials grants), token
// caching and refresh, and request/response (de)serialization. It contains no
// Terraform-specific logic so it can be reused and unit-tested in isolation.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ErrNotFound is returned when the API responds with 404. Resource Read
// implementations use it to remove a resource from state cleanly.
var ErrNotFound = errors.New("ferriskey: resource not found")

// APIErrorResponse wraps an APIError with the HTTP status for richer messages.
type APIErrorResponse struct {
	StatusCode int
	Err        APIError
}

func (e *APIErrorResponse) Error() string {
	if len(e.Err.Errors) > 0 {
		parts := make([]string, 0, len(e.Err.Errors))
		for _, fe := range e.Err.Errors {
			if fe.Field != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", fe.Field, fe.Message))
			} else {
				parts = append(parts, fe.Message)
			}
		}
		return fmt.Sprintf("ferriskey API error %d: %s", e.StatusCode, strings.Join(parts, "; "))
	}
	if e.Err.Message != "" {
		return fmt.Sprintf("ferriskey API error %d (%s): %s", e.StatusCode, e.Err.Code, e.Err.Message)
	}
	return fmt.Sprintf("ferriskey API error %d", e.StatusCode)
}

// defaultMaxRetries is the number of times a retryable request is retried after
// the first attempt.
const defaultMaxRetries = 3

// Client talks to a single FerrisKey instance.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	tokens      *tokenManager
	userAgent   string
	maxRetries  int
	backoffBase time.Duration
}

// Config configures a Client.
type Config struct {
	// URL is the base URL of the FerrisKey instance (no trailing slash, no
	// /realms suffix), e.g. https://auth.example.com.
	URL        string
	Auth       AuthConfig
	HTTPClient *http.Client
	UserAgent  string
	// MaxRetries is the number of retries for transient failures (429/5xx and
	// connection errors on idempotent requests). Zero uses the default.
	MaxRetries int
	// BackoffBase is the base delay for exponential backoff. Zero uses a sane
	// default; tests set it small.
	BackoffBase time.Duration
}

// New builds a Client. It does not perform any network call; the first token
// is fetched lazily on the first authenticated request.
func New(cfg Config) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	baseURL := strings.TrimRight(cfg.URL, "/")
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	backoffBase := cfg.BackoffBase
	if backoffBase <= 0 {
		backoffBase = 500 * time.Millisecond
	}
	return &Client{
		baseURL:     baseURL,
		httpClient:  httpClient,
		tokens:      newTokenManager(baseURL, httpClient, cfg.Auth),
		userAgent:   cfg.UserAgent,
		maxRetries:  maxRetries,
		backoffBase: backoffBase,
	}
}

// do performs an authenticated JSON request against path (which must begin with
// "/"). When body is non-nil it is JSON-encoded. When out is non-nil the
// response body is JSON-decoded into it. A 404 yields ErrNotFound; other non-2xx
// statuses yield an *APIErrorResponse.
//
// Resilience:
//   - A single extra attempt is made on 401 after invalidating the cached token,
//     to survive token expiry our local clock missed (not counted as a retry).
//   - Transient failures are retried with exponential backoff (honoring
//     Retry-After): 429 for any method; connection errors and 5xx only for
//     idempotent methods (GET/HEAD/PUT/DELETE), never for POST/PATCH, to avoid
//     duplicating a mutation whose outcome is unknown.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
	}

	tflog.Trace(ctx, "ferriskey API request", map[string]any{"method": method, "path": path})

	var resp *http.Response
	attempt := 0
	refreshedOn401 := false

	for {
		var err error
		resp, err = c.send(ctx, method, path, encoded)
		if err != nil {
			// Connection-level error: no response was received.
			if isIdempotent(method) && attempt < c.maxRetries {
				attempt++
				tflog.Debug(ctx, "retrying ferriskey request after connection error",
					map[string]any{"method": method, "path": path, "attempt": attempt, "error": err.Error()})
				if waitErr := c.backoff(ctx, attempt, 0); waitErr != nil {
					return waitErr
				}
				continue
			}
			return err
		}

		status := resp.StatusCode

		// One-shot token refresh on 401, separate from transient-error retries.
		if status == http.StatusUnauthorized && !refreshedOn401 {
			resp.Body.Close()
			c.tokens.invalidate()
			refreshedOn401 = true
			tflog.Debug(ctx, "re-authenticating after 401", map[string]any{"method": method, "path": path})
			continue
		}

		if shouldRetryStatus(method, status) && attempt < c.maxRetries {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			resp.Body.Close()
			attempt++
			tflog.Debug(ctx, "retrying ferriskey request after transient status",
				map[string]any{"method": method, "path": path, "status": status, "attempt": attempt})
			if waitErr := c.backoff(ctx, attempt, retryAfter); waitErr != nil {
				return waitErr
			}
			continue
		}

		break
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	tflog.Trace(ctx, "ferriskey API response", map[string]any{"method": method, "path": path, "status": resp.StatusCode})

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIErrorResponse{StatusCode: resp.StatusCode}
		_ = json.Unmarshal(respBody, &apiErr.Err)
		return apiErr
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decoding response from %s %s: %w", method, path, err)
		}
	}
	return nil
}

// send issues a single authenticated request attempt.
func (c *Client) send(ctx context.Context, method, path string, encoded []byte) (*http.Response, error) {
	token, err := c.tokens.token(ctx)
	if err != nil {
		return nil, fmt.Errorf("authenticating: %w", err)
	}

	var reader io.Reader
	if encoded != nil {
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if encoded != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	return c.httpClient.Do(req)
}

// isIdempotent reports whether a method is safe to retry after an ambiguous
// failure (connection error or 5xx) without risking a duplicate mutation.
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	default: // POST, PATCH
		return false
	}
}

// shouldRetryStatus decides whether a received HTTP status warrants a retry.
// 429 is always retryable (the request was not processed); 5xx only for
// idempotent methods.
func shouldRetryStatus(method string, status int) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	if status >= 500 && status <= 599 {
		return isIdempotent(method)
	}
	return false
}

// parseRetryAfter parses a Retry-After header expressed in seconds. It ignores
// the HTTP-date form (rare for this API) and returns 0 when unset/unparseable.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

// backoff waits before the next attempt: max(exponential base*2^(attempt-1),
// Retry-After), aborting early if the context is cancelled.
func (c *Client) backoff(ctx context.Context, attempt int, retryAfter time.Duration) error {
	wait := c.backoffBase * time.Duration(1<<(attempt-1))
	if retryAfter > wait {
		wait = retryAfter
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Ping verifies connectivity and credentials by requesting a token and reading
// the configured auth realm. Used by the provider's Configure to fail fast with
// a clear message.
func (c *Client) Ping(ctx context.Context, realm string) error {
	if _, err := c.tokens.token(ctx); err != nil {
		return err
	}
	if realm != "" {
		if _, err := c.GetRealm(ctx, realm); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}
