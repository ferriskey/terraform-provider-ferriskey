package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type authzCall struct {
	method string
	uri    string
	body   map[string]any
}

func authzRecorder(t *testing.T, status int, response string) (http.Handler, *authzCall) {
	t.Helper()
	call := &authzCall{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call.method = r.Method
		call.uri = r.URL.RequestURI()
		raw, _ := io.ReadAll(r.Body)
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &call.body); err != nil {
				t.Errorf("invalid JSON body %q: %v", raw, err)
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}), call
}

func authzExpect(t *testing.T, call *authzCall, method, uri string) {
	t.Helper()
	if call.method != method || call.uri != uri {
		t.Fatalf("got %s %s, want %s %s", call.method, call.uri, method, uri)
	}
}

func TestAuthzState(t *testing.T) {
	h, call := authzRecorder(t, 200, `{"enabled":true,"version":3}`)
	c, srv := newTestClient(t, h)
	defer srv.Close()

	s, err := c.SetAuthzState(context.Background(), "acme", true)
	if err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "PUT", "/realms/acme/authz/state")
	if call.body["enabled"] != true || !s.Enabled || s.Version != 3 {
		t.Fatalf("unexpected: body=%v state=%+v", call.body, s)
	}

	s, err = c.GetAuthzState(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "GET", "/realms/acme/authz/state")
	if !s.Enabled {
		t.Fatalf("unexpected state %+v", s)
	}
}

func TestAuthzSchema(t *testing.T) {
	h, call := authzRecorder(t, 200, `{"namespace":"App","source":"namespace App {}","updated_at":"2026-01-01T00:00:00Z"}`)
	c, srv := newTestClient(t, h)
	defer srv.Close()

	ns := "App"
	s, err := c.PutAuthzSchema(context.Background(), "acme", ReplaceAuthzSchemaRequest{Namespace: &ns, Source: "namespace App {}"})
	if err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "PUT", "/realms/acme/authz/schema")
	if call.body["namespace"] != "App" || call.body["source"] != "namespace App {}" || s.Namespace != "App" {
		t.Fatalf("unexpected: body=%v schema=%+v", call.body, s)
	}

	if _, err := c.GetAuthzSchema(context.Background(), "acme"); err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "GET", "/realms/acme/authz/schema")

	b, err := c.GetAuthzBuiltinSchema(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "GET", "/realms/acme/authz/schema/builtin")
	if b.Namespace != "App" {
		t.Fatalf("unexpected builtin %+v", b)
	}
}

func TestAuthzPolicy(t *testing.T) {
	resp := `{"id":"p1","name":"n","description":null,"origin":"generated","enabled":true,"managed_by":"tf","body":{"type":"linked","template_id":"t1","links":{"?principal":{"type":"FerrisKey::Role","id":"r1"}}}}`
	h, call := authzRecorder(t, 201, resp)
	c, srv := newTestClient(t, h)
	defer srv.Close()

	tid := "t1"
	req := AuthzPolicyRequest{
		Name:    "n",
		Enabled: true,
		Body: AuthzPolicyBody{
			Type:       "linked",
			TemplateID: &tid,
			Links:      map[string]AuthzEntityRef{"?principal": {Type: "FerrisKey::Role", ID: "r1"}},
		},
	}
	p, err := c.CreateAuthzPolicy(context.Background(), "acme", "tf", req)
	if err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "POST", "/realms/acme/authz/policies?managed_by=tf")
	body := call.body["body"].(map[string]any)
	if body["type"] != "linked" || body["template_id"] != "t1" || body["source"] != nil {
		t.Fatalf("unexpected body %v", call.body)
	}
	if _, ok := call.body["description"]; ok {
		t.Fatalf("nil description must be omitted: %v", call.body)
	}
	if p.ID != "p1" || p.Description != nil || p.Body.Links["?principal"].ID != "r1" || *p.ManagedBy != "tf" {
		t.Fatalf("unexpected policy %+v", p)
	}

	if _, err := c.GetAuthzPolicy(context.Background(), "acme", "p1"); err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "GET", "/realms/acme/authz/policies/p1")

	if _, err := c.UpdateAuthzPolicy(context.Background(), "acme", "p1", "", req); err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "PUT", "/realms/acme/authz/policies/p1")

	if err := c.DeleteAuthzPolicy(context.Background(), "acme", "p1"); err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "DELETE", "/realms/acme/authz/policies/p1")
}

func TestAuthzPolicy_StaticBody(t *testing.T) {
	h, call := authzRecorder(t, 201, `{"id":"p1","name":"n","origin":"custom","enabled":false,"body":{"type":"static","source":"permit(principal,action,resource);"}}`)
	c, srv := newTestClient(t, h)
	defer srv.Close()

	src := "permit(principal,action,resource);"
	p, err := c.CreateAuthzPolicy(context.Background(), "acme", "", AuthzPolicyRequest{Name: "n", Body: AuthzPolicyBody{Type: "static", Source: &src}})
	if err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "POST", "/realms/acme/authz/policies")
	if call.body["enabled"] != false || p.Body.Source == nil || *p.Body.Source != src {
		t.Fatalf("unexpected: body=%v policy=%+v", call.body, p)
	}
}

func TestAuthzTemplate(t *testing.T) {
	h, call := authzRecorder(t, 201, `{"id":"t1","name":"n","description":"d","source":"permit(principal == ?principal, action, resource);"}`)
	c, srv := newTestClient(t, h)
	defer srv.Close()

	d := "d"
	req := AuthzTemplateRequest{Name: "n", Description: &d, Source: "permit(principal == ?principal, action, resource);"}
	tpl, err := c.CreateAuthzTemplate(context.Background(), "acme", req)
	if err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "POST", "/realms/acme/authz/templates")
	if call.body["name"] != "n" || call.body["description"] != "d" || tpl.ID != "t1" {
		t.Fatalf("unexpected: body=%v template=%+v", call.body, tpl)
	}

	if _, err := c.GetAuthzTemplate(context.Background(), "acme", "t1"); err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "GET", "/realms/acme/authz/templates/t1")

	if _, err := c.UpdateAuthzTemplate(context.Background(), "acme", "t1", req); err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "PUT", "/realms/acme/authz/templates/t1")

	if err := c.DeleteAuthzTemplate(context.Background(), "acme", "t1"); err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "DELETE", "/realms/acme/authz/templates/t1")
}

func TestResourceOwners(t *testing.T) {
	h, call := authzRecorder(t, 200, `{"data":[{"uri":"https://mcp.example.com","client_id":"c1"}]}`)
	c, srv := newTestClient(t, h)
	defer srv.Close()

	owners, err := c.GetResourceOwners(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "GET", "/realms/acme/resource-owners")
	if len(owners) != 1 || owners[0].URI != "https://mcp.example.com" || owners[0].ClientID != "c1" {
		t.Fatalf("unexpected owners %+v", owners)
	}

	if _, err := c.ReplaceResourceOwners(context.Background(), "acme", nil); err != nil {
		t.Fatal(err)
	}
	authzExpect(t, call, "PUT", "/realms/acme/resource-owners")
	if list, ok := call.body["owners"].([]any); !ok || len(list) != 0 {
		t.Fatalf("empty owners must be sent as [], got %v", call.body)
	}
}
