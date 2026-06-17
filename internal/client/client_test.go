package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient builds a Client wired to srv with a token endpoint that always
// succeeds, so tests can focus on the resource endpoints.
func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/realms/master/protocol/openid-connect/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(JwtToken{AccessToken: "test-token", TokenType: "Bearer", ExpiresIn: 3600})
	})
	mux.Handle("/", handler)
	srv := httptest.NewServer(mux)
	c := New(Config{
		URL:         srv.URL,
		HTTPClient:  srv.Client(),
		Auth:        AuthConfig{Realm: "master", GrantType: GrantClientCredentials, ClientID: "svc", ClientSecret: "s"},
		BackoffBase: time.Millisecond, // keep retry tests fast
	})
	return c, srv
}

func TestDo_NotFoundMapsToErrNotFound(t *testing.T) {
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := c.GetRealm(context.Background(), "ghost")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDo_AuthorizationHeaderAndErrorEnvelope(t *testing.T) {
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("missing/incorrect auth header: %q", got)
		}
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(APIError{Code: "E_FORBIDDEN", Status: 403, Message: "nope"})
	}))
	defer srv.Close()

	_, err := c.GetRealm(context.Background(), "master")
	var apiErr *APIErrorResponse
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIErrorResponse, got %v", err)
	}
	if apiErr.StatusCode != 403 || apiErr.Err.Message != "nope" {
		t.Fatalf("unexpected error contents: %+v", apiErr)
	}
}

func TestDo_RetriesOnceOn401(t *testing.T) {
	var attempts int32
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(Realm{ID: "id", Name: "master"})
	}))
	defer srv.Close()

	realm, err := c.GetRealm(context.Background(), "master")
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if realm.Name != "master" {
		t.Fatalf("unexpected realm: %+v", realm)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected 2 attempts (401 then retry), got %d", attempts)
	}
}

func TestDo_RetriesOnTransientStatusForGET(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var attempts int32
			c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if atomic.AddInt32(&attempts, 1) == 1 {
					w.WriteHeader(status)
					return
				}
				_ = json.NewEncoder(w).Encode(Realm{ID: "id", Name: "master"})
			}))
			defer srv.Close()

			realm, err := c.GetRealm(context.Background(), "master")
			if err != nil {
				t.Fatalf("expected retry to succeed, got %v", err)
			}
			if realm.Name != "master" {
				t.Fatalf("unexpected realm %+v", realm)
			}
			if got := atomic.LoadInt32(&attempts); got != 2 {
				t.Fatalf("expected 2 attempts, got %d", got)
			}
		})
	}
}

func TestDo_DoesNotRetryPOSTOn5xx(t *testing.T) {
	var attempts int32
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	// CreateRealm issues a POST; a 5xx must not be retried (could double-create).
	_, err := c.CreateRealm(context.Background(), CreateRealmRequest{Name: "x"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("expected exactly 1 attempt for POST 5xx, got %d", got)
	}
}

func TestDo_RetriesPOSTOn429(t *testing.T) {
	var attempts int32
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(Realm{ID: "id", Name: "x"})
	}))
	defer srv.Close()

	// 429 means the request was not processed, so retrying a POST is safe.
	if _, err := c.CreateRealm(context.Background(), CreateRealmRequest{Name: "x"}); err != nil {
		t.Fatalf("expected 429 retry to succeed, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestSetRedirectURIs_ReconcilesAddAndDelete(t *testing.T) {
	// Server-side state: start with two URIs.
	existing := map[string]RedirectURI{
		"https://a.example/cb": {ID: "id-a", Value: "https://a.example/cb", Enabled: true},
		"https://b.example/cb": {ID: "id-b", Value: "https://b.example/cb", Enabled: true},
	}
	var created, deleted []string

	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			list := make([]RedirectURI, 0, len(existing))
			for _, u := range existing {
				list = append(list, u)
			}
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodPost:
			var req CreateRedirectURIRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			created = append(created, req.Value)
			u := RedirectURI{ID: "new-" + req.Value, Value: req.Value, Enabled: req.Enabled}
			existing[req.Value] = u
			_ = json.NewEncoder(w).Encode(u)
		case r.Method == http.MethodDelete:
			// Path: /realms/master/clients/<cid>/redirects/<uriID>
			_, id, _ := strings.Cut(r.URL.Path, "/redirects/")
			deleted = append(deleted, id)
			for v, u := range existing {
				if u.ID == id {
					delete(existing, v)
				}
			}
			w.WriteHeader(http.StatusOK)
		}
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	// Desired: keep a, drop b, add c.
	result, err := c.SetRedirectURIs(context.Background(), "master", "cid",
		[]string{"https://a.example/cb", "https://c.example/cb"})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if len(created) != 1 || created[0] != "https://c.example/cb" {
		t.Errorf("expected to create c only, created=%v", created)
	}
	if len(deleted) != 1 || deleted[0] != "id-b" {
		t.Errorf("expected to delete id-b only, deleted=%v", deleted)
	}
	values := map[string]bool{}
	for _, u := range result {
		values[u.Value] = true
	}
	if !values["https://a.example/cb"] || !values["https://c.example/cb"] || values["https://b.example/cb"] {
		t.Errorf("unexpected final set: %v", values)
	}
}
