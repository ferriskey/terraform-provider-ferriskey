package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock returns a controllable time source.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }
func (c *fakeClock) advance(d time.Duration) {
	c.t = c.t.Add(d)
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func TestTokenManager_CachesAndRefreshesOnExpiry(t *testing.T) {
	var tokenCalls int32
	var lastForm map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected form content type, got %q", ct)
		}
		_ = r.ParseForm()
		lastForm = map[string]string{}
		for k := range r.Form {
			lastForm[k] = r.Form.Get(k)
		}
		n := atomic.AddInt32(&tokenCalls, 1)
		resp := JwtToken{
			AccessToken:      "access-" + string(rune('0'+n)),
			TokenType:        "Bearer",
			RefreshToken:     "refresh-token",
			ExpiresIn:        300,
			RefreshExpiresIn: 1800,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	clock := newFakeClock()
	m := newTokenManager(srv.URL, srv.Client(), AuthConfig{
		Realm:        "master",
		GrantType:    GrantClientCredentials,
		ClientID:     "svc",
		ClientSecret: "secret",
	})
	m.now = clock.now

	ctx := context.Background()

	tok1, err := m.token(ctx)
	if err != nil {
		t.Fatalf("first token: %v", err)
	}
	if tok1 != "access-1" {
		t.Fatalf("expected access-1, got %q", tok1)
	}
	if lastForm["grant_type"] != "client_credentials" || lastForm["client_id"] != "svc" || lastForm["client_secret"] != "secret" {
		t.Fatalf("unexpected initial grant form: %+v", lastForm)
	}

	// Second call within the lifetime must reuse the cached token (no new HTTP call).
	clock.advance(100 * time.Second)
	tok2, _ := m.token(ctx)
	if tok2 != "access-1" {
		t.Fatalf("expected cached access-1, got %q", tok2)
	}
	if got := atomic.LoadInt32(&tokenCalls); got != 1 {
		t.Fatalf("expected 1 token call, got %d", got)
	}

	// After the access token expires (minus margin) but while the refresh
	// token is valid, it must refresh via refresh_token.
	clock.advance(250 * time.Second) // total 350s > 300s lifetime
	tok3, err := m.token(ctx)
	if err != nil {
		t.Fatalf("refresh token: %v", err)
	}
	if tok3 == "access-1" {
		t.Fatalf("expected a refreshed token, still got %q", tok3)
	}
	if lastForm["grant_type"] != "refresh_token" || lastForm["refresh_token"] != "refresh-token" {
		t.Fatalf("expected refresh_token grant, got %+v", lastForm)
	}
	if got := atomic.LoadInt32(&tokenCalls); got != 2 {
		t.Fatalf("expected 2 token calls, got %d", got)
	}
}

func TestTokenManager_PasswordGrantForm(t *testing.T) {
	var lastForm map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		lastForm = map[string]string{}
		for k := range r.Form {
			lastForm[k] = r.Form.Get(k)
		}
		_ = json.NewEncoder(w).Encode(JwtToken{AccessToken: "a", TokenType: "Bearer", ExpiresIn: 300})
	}))
	defer srv.Close()

	m := newTokenManager(srv.URL, srv.Client(), AuthConfig{
		Realm:     "master",
		GrantType: GrantPassword,
		Username:  "admin",
		Password:  "pw",
		ClientID:  "admin-cli",
	})
	m.now = newFakeClock().now

	if _, err := m.token(context.Background()); err != nil {
		t.Fatalf("token: %v", err)
	}
	for k, want := range map[string]string{
		"grant_type": "password",
		"username":   "admin",
		"password":   "pw",
		"client_id":  "admin-cli",
	} {
		if lastForm[k] != want {
			t.Errorf("form[%q] = %q, want %q", k, lastForm[k], want)
		}
	}
}

func TestTokenManager_RefreshFallsBackToFreshGrant(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		n := atomic.AddInt32(&calls, 1)
		// Reject the refresh-token attempt (call #2) to exercise the fallback.
		if r.Form.Get("grant_type") == "refresh_token" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(APIError{Code: "invalid_grant", Status: 400, Message: "expired"})
			return
		}
		_ = json.NewEncoder(w).Encode(JwtToken{
			AccessToken:      "fresh-" + string(rune('0'+n)),
			TokenType:        "Bearer",
			ExpiresIn:        300,
			RefreshToken:     "rt",
			RefreshExpiresIn: 1800,
		})
	}))
	defer srv.Close()

	clock := newFakeClock()
	m := newTokenManager(srv.URL, srv.Client(), AuthConfig{Realm: "master", GrantType: GrantClientCredentials, ClientID: "svc", ClientSecret: "s"})
	m.now = clock.now

	if _, err := m.token(context.Background()); err != nil {
		t.Fatalf("initial: %v", err)
	}
	clock.advance(400 * time.Second) // force expiry
	tok, err := m.token(context.Background())
	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if tok == "" {
		t.Fatal("expected a non-empty token after fallback")
	}
}
