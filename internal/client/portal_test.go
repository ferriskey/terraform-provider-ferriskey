package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type portalCall struct {
	method string
	uri    string
	body   string
}

func portalServer(t *testing.T, status int, resp string) (*Client, *portalCall, func()) {
	t.Helper()
	got := &portalCall{}
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*got = portalCall{method: r.Method, uri: r.URL.RequestURI(), body: string(b)}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	}))
	return c, got, srv.Close
}

func portalExpect(t *testing.T, got *portalCall, method, uri, body string) {
	t.Helper()
	if got.method != method || got.uri != uri {
		t.Fatalf("got %s %s, want %s %s", got.method, got.uri, method, uri)
	}
	if body != "" && got.body != body {
		t.Fatalf("got body %s, want %s", got.body, body)
	}
}

const portalLayoutJSON = `{"data":{"id":"l1","name":"n","tree":{"a":1},"is_default":true}}`

func TestPortalLayoutCRUD(t *testing.T) {
	ctx := context.Background()
	c, got, done := portalServer(t, 200, portalLayoutJSON)
	defer done()

	l, err := c.CreatePortalLayout(ctx, "r", PortalLayoutRequest{Name: "n", Tree: json.RawMessage(`{"a":1}`)})
	if err != nil || l.ID != "l1" || !l.IsDefault || string(l.Tree) != `{"a":1}` {
		t.Fatalf("create: %+v %v", l, err)
	}
	portalExpect(t, got, "POST", "/realms/r/portal-layouts", `{"name":"n","tree":{"a":1}}`)

	if _, err = c.GetPortalLayout(ctx, "r", "l1"); err != nil {
		t.Fatal(err)
	}
	portalExpect(t, got, "GET", "/realms/r/portal-layouts/l1", "")

	if _, err = c.UpdatePortalLayout(ctx, "r", "l1", PortalLayoutRequest{Name: "m", Tree: json.RawMessage(`[]`)}); err != nil {
		t.Fatal(err)
	}
	portalExpect(t, got, "PUT", "/realms/r/portal-layouts/l1", `{"name":"m","tree":[]}`)

	if _, err = c.SetDefaultPortalLayout(ctx, "r", "l1"); err != nil {
		t.Fatal(err)
	}
	portalExpect(t, got, "PUT", "/realms/r/portal-layouts/l1/default", "")

	if err = c.DeletePortalLayout(ctx, "r", "l1"); err != nil {
		t.Fatal(err)
	}
	portalExpect(t, got, "DELETE", "/realms/r/portal-layouts/l1", "")
}

func TestPortalDefaultLayout(t *testing.T) {
	c, got, done := portalServer(t, 200, `{"data":[{"id":"l1","name":"n","tree":{},"is_default":true}],"metadata":{}}`)
	defer done()
	l, err := c.GetDefaultPortalLayout(context.Background(), "r")
	if err != nil || l.ID != "l1" {
		t.Fatalf("%+v %v", l, err)
	}
	portalExpect(t, got, "GET", "/realms/r/portal-layouts?is_default=true&limit=1", "")

	c2, _, done2 := portalServer(t, 200, `{"data":[],"metadata":{}}`)
	defer done2()
	if _, err = c2.GetDefaultPortalLayout(context.Background(), "r"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

const portalThemeJSON = `{"data":{"id":"t1","name":"n","layout_id":null,"config":{"colors":{}},"pages":{"login":[],"forgotPassword":[1]}}}`

func TestPortalThemeCRUD(t *testing.T) {
	ctx := context.Background()
	c, got, done := portalServer(t, 200, portalThemeJSON)
	defer done()

	lid := "l1"
	th, err := c.CreatePortalTheme(ctx, "r", PortalThemeRequest{Name: "n", LayoutID: &lid, Config: json.RawMessage(`{"colors":{}}`)})
	if err != nil || th.ID != "t1" || th.LayoutID != nil || string(th.Config) != `{"colors":{}}` {
		t.Fatalf("create: %+v %v", th, err)
	}
	portalExpect(t, got, "POST", "/realms/r/portal/themes", `{"name":"n","layout_id":"l1","config":{"colors":{}}}`)

	if _, err = c.GetPortalTheme(ctx, "r", "t1"); err != nil {
		t.Fatal(err)
	}
	portalExpect(t, got, "GET", "/realms/r/portal/themes/t1", "")

	if _, err = c.UpdatePortalTheme(ctx, "r", "t1", PortalThemeRequest{Name: "m"}); err != nil {
		t.Fatal(err)
	}
	portalExpect(t, got, "PUT", "/realms/r/portal/themes/t1", `{"name":"m","layout_id":null}`)

	if err = c.DeletePortalTheme(ctx, "r", "t1"); err != nil {
		t.Fatal(err)
	}
	portalExpect(t, got, "DELETE", "/realms/r/portal/themes/t1", "")
}

func TestPortalThemePageAndActivate(t *testing.T) {
	ctx := context.Background()
	c, got, done := portalServer(t, 200, portalThemeJSON)
	defer done()

	if _, err := c.UpdatePortalThemePage(ctx, "r", "t1", "forgot_password", json.RawMessage(`[1]`)); err != nil {
		t.Fatal(err)
	}
	portalExpect(t, got, "PUT", "/realms/r/portal/themes/t1/pages/forgot_password", `{"tree":[1]}`)

	if err := c.ActivatePortalTheme(ctx, "r", "t1"); err != nil {
		t.Fatal(err)
	}
	portalExpect(t, got, "POST", "/realms/r/portal/themes/t1/activate", "")
}

func TestPortalActiveThemeID(t *testing.T) {
	c, got, done := portalServer(t, 200, `{"theme_id":"t1","design_tokens":{},"layout_id":null,"page_tree":[]}`)
	defer done()
	id, err := c.GetActivePortalThemeID(context.Background(), "r")
	if err != nil || id != "t1" {
		t.Fatalf("%q %v", id, err)
	}
	portalExpect(t, got, "GET", "/realms/r/portal/active?page_type=login", "")

	c2, _, done2 := portalServer(t, 200, `{"theme_id":null,"design_tokens":{},"page_tree":[]}`)
	defer done2()
	id, err = c2.GetActivePortalThemeID(context.Background(), "r")
	if err != nil || id != "" {
		t.Fatalf("%q %v", id, err)
	}
}
