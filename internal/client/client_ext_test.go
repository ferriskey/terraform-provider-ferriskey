package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

type clientExtCall struct {
	method string
	path   string
	body   string
}

func clientExtServer(t *testing.T, status int, resp string, got *clientExtCall) (*Client, func()) {
	t.Helper()
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*got = clientExtCall{method: r.Method, path: r.URL.Path, body: string(b)}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	}))
	return c, srv.Close
}

func clientExtExpect(t *testing.T, got clientExtCall, method, path, body string) {
	t.Helper()
	if got.method != method || got.path != path {
		t.Fatalf("got %s %s, want %s %s", got.method, got.path, method, path)
	}
	if body != "" {
		var a, b any
		_ = json.Unmarshal([]byte(got.body), &a)
		_ = json.Unmarshal([]byte(body), &b)
		ja, _ := json.Marshal(a)
		jb, _ := json.Marshal(b)
		if string(ja) != string(jb) {
			t.Fatalf("body %s, want %s", got.body, body)
		}
	}
}

func TestClientExt_CreateClientRole(t *testing.T) {
	var got clientExtCall
	c, done := clientExtServer(t, 201, `{"id":"r1","name":"n","client_id":"c1","permissions":["a"]}`, &got)
	defer done()
	d := "d"
	role, err := c.CreateClientRole(context.Background(), "re", "c1", CreateRoleRequest{Name: "n", Description: &d, Permissions: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "POST", "/realms/re/clients/c1/roles", `{"name":"n","description":"d","permissions":["a"]}`)
	if role.ID != "r1" || *role.ClientID != "c1" || role.Permissions[0] != "a" {
		t.Fatalf("bad decode: %+v", role)
	}
}

func TestClientExt_WebOrigins(t *testing.T) {
	var got clientExtCall
	c, done := clientExtServer(t, 201, `{"id":"w1","client_id":"c1","value":"https://a"}`, &got)
	w, err := c.CreateClientWebOrigin(context.Background(), "re", "c1", CreateClientWebOriginRequest{Value: "https://a"})
	done()
	if err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "POST", "/realms/re/clients/c1/web-origins", `{"value":"https://a"}`)
	if w.ID != "w1" || w.Value != "https://a" {
		t.Fatalf("bad decode: %+v", w)
	}

	c, done = clientExtServer(t, 200, `[{"id":"w0","value":"x"},{"id":"w1","value":"https://a"}]`, &got)
	w, err = c.GetClientWebOrigin(context.Background(), "re", "c1", "w1")
	if err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "GET", "/realms/re/clients/c1/web-origins", "")
	if w.Value != "https://a" {
		t.Fatalf("bad find: %+v", w)
	}
	if _, err = c.GetClientWebOrigin(context.Background(), "re", "c1", "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	done()

	c, done = clientExtServer(t, 200, `null`, &got)
	defer done()
	if err := c.DeleteClientWebOrigin(context.Background(), "re", "c1", "w1"); err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "DELETE", "/realms/re/clients/c1/web-origins/w1", "")
}

func TestClientExt_PostLogoutRedirectURIs(t *testing.T) {
	var got clientExtCall
	c, done := clientExtServer(t, 201, `{"id":"u1","client_id":"c1","value":"https://a/out","enabled":true}`, &got)
	u, err := c.CreateClientPostLogoutRedirectURI(context.Background(), "re", "c1", CreateClientPostLogoutRedirectURIRequest{Value: "https://a/out", Enabled: true})
	done()
	if err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "POST", "/realms/re/clients/c1/post-logout-redirects", `{"value":"https://a/out","enabled":true}`)
	if u.ID != "u1" || !u.Enabled {
		t.Fatalf("bad decode: %+v", u)
	}

	c, done = clientExtServer(t, 200, `{"data":{"id":"u1","value":"https://a/out","enabled":false}}`, &got)
	u, err = c.UpdateClientPostLogoutRedirectURI(context.Background(), "re", "c1", "u1", UpdateClientPostLogoutRedirectURIRequest{Enabled: false})
	done()
	if err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "PUT", "/realms/re/clients/c1/post-logout-redirects/u1", `{"enabled":false}`)
	if u.Enabled || u.ID != "u1" {
		t.Fatalf("bad envelope decode: %+v", u)
	}

	c, done = clientExtServer(t, 200, `[{"id":"u1","value":"v","enabled":true}]`, &got)
	u, err = c.GetClientPostLogoutRedirectURI(context.Background(), "re", "c1", "u1")
	done()
	if err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "GET", "/realms/re/clients/c1/post-logout-redirects", "")
	if !u.Enabled {
		t.Fatalf("bad find: %+v", u)
	}

	c, done = clientExtServer(t, 200, `null`, &got)
	defer done()
	if err := c.DeleteClientPostLogoutRedirectURI(context.Background(), "re", "c1", "u1"); err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "DELETE", "/realms/re/clients/c1/post-logout-redirects/u1", "")
}

func TestClientExt_TokenExchangePolicies(t *testing.T) {
	var got clientExtCall
	c, done := clientExtServer(t, 201, `{"id":"p1","client_id":"c1","target_audience":"c2","allowed_scopes":["openid"],"allow_impersonation":false,"allow_delegation":true}`, &got)
	p, err := c.CreateClientTokenExchangePolicy(context.Background(), "re", "c1", CreateClientTokenExchangePolicyRequest{
		TargetAudience: "c2", AllowedScopes: []string{"openid"}, AllowDelegation: true,
	})
	done()
	if err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "POST", "/realms/re/clients/c1/token-exchange-policies",
		`{"target_audience":"c2","allowed_scopes":["openid"],"allow_impersonation":false,"allow_delegation":true}`)
	if p.ID != "p1" || !p.AllowDelegation || p.AllowedScopes[0] != "openid" {
		t.Fatalf("bad decode: %+v", p)
	}

	c, done = clientExtServer(t, 200, `[{"id":"p1","target_audience":"c2","allowed_scopes":null}]`, &got)
	p, err = c.GetClientTokenExchangePolicy(context.Background(), "re", "c1", "p1")
	done()
	if err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "GET", "/realms/re/clients/c1/token-exchange-policies", "")
	if p.AllowedScopes != nil {
		t.Fatalf("expected nil scopes: %+v", p)
	}

	c, done = clientExtServer(t, 200, `null`, &got)
	defer done()
	if err := c.DeleteClientTokenExchangePolicy(context.Background(), "re", "c1", "p1"); err != nil {
		t.Fatal(err)
	}
	clientExtExpect(t, got, "DELETE", "/realms/re/clients/c1/token-exchange-policies/p1", "")
}
