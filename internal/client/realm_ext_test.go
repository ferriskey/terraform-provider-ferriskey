package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

type rxCall struct {
	method string
	path   string
	body   string
}

func rxServer(t *testing.T, status int, response string, calls *[]rxCall) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*calls = append(*calls, rxCall{method: r.Method, path: r.URL.EscapedPath(), body: string(b)})
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	})
}

func rxExpect(t *testing.T, got rxCall, method, path, body string) {
	t.Helper()
	if got.method != method || got.path != path {
		t.Fatalf("got %s %s, want %s %s", got.method, got.path, method, path)
	}
	if body == "" {
		return
	}
	var a, b any
	if err := json.Unmarshal([]byte(got.body), &a); err != nil {
		t.Fatalf("body %q: %v", got.body, err)
	}
	_ = json.Unmarshal([]byte(body), &b)
	ga, _ := json.Marshal(a)
	gb, _ := json.Marshal(b)
	if string(ga) != string(gb) {
		t.Fatalf("body = %s, want %s", ga, gb)
	}
}

const rxWebhookJSON = `{"id":"w1","endpoint":"https://e.com","name":"n","description":null,"subscribers":[{"id":"s","name":"user.created","webhook_id":"w1"}],"retry_policy":{"max_attempts":3,"base_delay_ms":null,"max_delay_ms":null,"max_total_delay_ms":null}}`

func TestWebhook_GetBareAndNullIsNotFound(t *testing.T) {
	var calls []rxCall
	c, srv := newTestClient(t, rxServer(t, 200, rxWebhookJSON, &calls))
	defer srv.Close()

	w, err := c.GetWebhook(context.Background(), "r", "w1")
	if err != nil {
		t.Fatal(err)
	}
	rxExpect(t, calls[0], "GET", "/realms/r/webhooks/w1", "")
	if w.ID != "w1" || w.Endpoint != "https://e.com" || len(w.Subscribers) != 1 || *w.RetryPolicy.MaxAttempts != 3 {
		t.Fatalf("unexpected webhook %+v", w)
	}

	calls = nil
	c2, srv2 := newTestClient(t, rxServer(t, 200, "null", &calls))
	defer srv2.Close()
	if _, err := c2.GetWebhook(context.Background(), "r", "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestWebhook_CreateUpdateDeleteRotate(t *testing.T) {
	var calls []rxCall
	c, srv := newTestClient(t, rxServer(t, 200, `{"data":`+rxWebhookJSON+`}`, &calls))
	defer srv.Close()
	ctx := context.Background()

	name := "n"
	w, err := c.CreateWebhook(ctx, "r", CreateWebhookRequest{
		Endpoint: "https://e.com", Name: &name, Headers: map[string]string{"a": "b"}, Subscribers: []string{"user.created"},
	})
	if err != nil || w.ID != "w1" {
		t.Fatalf("create: %v %+v", err, w)
	}
	rxExpect(t, calls[0], "POST", "/realms/r/webhooks", `{"endpoint":"https://e.com","name":"n","headers":{"a":"b"},"subscribers":["user.created"]}`)

	calls = nil
	w, err = c.UpdateWebhook(ctx, "r", "w1", UpdateWebhookRequest{Endpoint: "https://e.com", Subscribers: []string{}, RetryPolicy: &WebhookRetryPolicy{}})
	if err != nil || w.ID != "w1" {
		t.Fatalf("update: %v %+v", err, w)
	}
	rxExpect(t, calls[0], "PUT", "/realms/r/webhooks/w1",
		`{"endpoint":"https://e.com","name":null,"description":null,"subscribers":[],"retry_policy":{"max_attempts":null,"base_delay_ms":null,"max_delay_ms":null,"max_total_delay_ms":null}}`)

	calls = nil
	if err := c.DeleteWebhook(ctx, "r", "w1"); err != nil {
		t.Fatal(err)
	}
	rxExpect(t, calls[0], "DELETE", "/realms/r/webhooks/w1", "")

	calls = nil
	c2, srv2 := newTestClient(t, rxServer(t, 200, `{"secret":"abc"}`, &calls))
	defer srv2.Close()
	secret, err := c2.RotateWebhookSecret(ctx, "r", "w1")
	if err != nil || secret != "abc" {
		t.Fatalf("rotate: %v %q", err, secret)
	}
	rxExpect(t, calls[0], "POST", "/realms/r/webhooks/w1/secret/rotate", "")
}

const rxTemplateJSON = `{"id":"t1","realm_id":"rid","name":"n","email_type":"reset_password","structure":{"children":[]},"mjml":"<mjml/>"}`

func TestEmailTemplate_CRUD(t *testing.T) {
	var calls []rxCall
	c, srv := newTestClient(t, rxServer(t, 200, `{"data":`+rxTemplateJSON+`}`, &calls))
	defer srv.Close()
	ctx := context.Background()

	tpl, err := c.CreateEmailTemplate(ctx, "r", CreateEmailTemplateRequest{Name: "n", EmailType: "reset_password", Structure: json.RawMessage(`{"children":[]}`)})
	if err != nil || tpl.ID != "t1" || string(tpl.Structure) != `{"children":[]}` || tpl.MJML != "<mjml/>" {
		t.Fatalf("create: %v %+v", err, tpl)
	}
	rxExpect(t, calls[0], "POST", "/realms/r/email-templates", `{"name":"n","email_type":"reset_password","structure":{"children":[]}}`)

	calls = nil
	if _, err := c.GetEmailTemplate(ctx, "r", "t1"); err != nil {
		t.Fatal(err)
	}
	rxExpect(t, calls[0], "GET", "/realms/r/email-templates/t1", "")

	calls = nil
	if _, err := c.UpdateEmailTemplate(ctx, "r", "t1", UpdateEmailTemplateRequest{Name: "m", Structure: json.RawMessage(`{"children":[]}`)}); err != nil {
		t.Fatal(err)
	}
	rxExpect(t, calls[0], "PUT", "/realms/r/email-templates/t1", `{"name":"m","structure":{"children":[]}}`)

	calls = nil
	if err := c.DeleteEmailTemplate(ctx, "r", "t1"); err != nil {
		t.Fatal(err)
	}
	rxExpect(t, calls[0], "DELETE", "/realms/r/email-templates/t1", "")
}

func TestEmailTemplate_Variables(t *testing.T) {
	var calls []rxCall
	c, srv := newTestClient(t, rxServer(t, 200, `{"data":[{"name":"user.email","description":"mail"}]}`, &calls))
	defer srv.Close()

	vars, err := c.GetEmailTemplateVariables(context.Background(), "reset_password")
	if err != nil || len(vars) != 1 || vars[0].Name != "user.email" || vars[0].Description != "mail" {
		t.Fatalf("variables: %v %+v", err, vars)
	}
	rxExpect(t, calls[0], "GET", "/email-templates/variables/reset_password", "")
}

const rxProviderJSON = `{"id":"p1","realm_id":"rid","name":"ldap","provider_type":"Ldap","enabled":true,"priority":2,"config":{"a":1},"sync_enabled":true,"sync_mode":"Force","sync_interval_minutes":15}`

func TestFederationProvider_CRUD(t *testing.T) {
	var calls []rxCall
	c, srv := newTestClient(t, rxServer(t, 201, rxProviderJSON, &calls))
	defer srv.Close()
	ctx := context.Background()

	interval := int64(15)
	p, err := c.CreateFederationProvider(ctx, "r", CreateFederationProviderRequest{
		Name: "ldap", ProviderType: "Ldap", Enabled: true, Priority: 2, Config: json.RawMessage(`{"a":1}`),
		SyncEnabled: true, SyncMode: "Force", SyncIntervalMinutes: &interval,
	})
	if err != nil || p.ID != "p1" || p.Priority != 2 || *p.SyncIntervalMinutes != 15 || string(p.Config) != `{"a":1}` {
		t.Fatalf("create: %v %+v", err, p)
	}
	rxExpect(t, calls[0], "POST", "/realms/r/federation/providers",
		`{"name":"ldap","provider_type":"Ldap","enabled":true,"priority":2,"config":{"a":1},"sync_enabled":true,"sync_mode":"Force","sync_interval_minutes":15}`)

	calls = nil
	c2, srv2 := newTestClient(t, rxServer(t, 200, rxProviderJSON, &calls))
	defer srv2.Close()
	if p, err = c2.GetFederationProvider(ctx, "r", "p1"); err != nil || p.Name != "ldap" {
		t.Fatalf("get: %v %+v", err, p)
	}
	rxExpect(t, calls[0], "GET", "/realms/r/federation/providers/p1", "")

	calls = nil
	c3, srv3 := newTestClient(t, rxServer(t, 200, `{"data":`+rxProviderJSON+`}`, &calls))
	defer srv3.Close()
	p, err = c3.UpdateFederationProvider(ctx, "r", "p1", UpdateFederationProviderRequest{
		Name: "ldap", ProviderType: "Ldap", Config: json.RawMessage(`{}`), SyncMode: "Import",
	})
	if err != nil || p.ID != "p1" {
		t.Fatalf("update: %v %+v", err, p)
	}
	rxExpect(t, calls[0], "PUT", "/realms/r/federation/providers/p1",
		`{"name":"ldap","provider_type":"Ldap","enabled":false,"priority":0,"config":{},"sync_enabled":false,"sync_mode":"Import","sync_interval_minutes":null}`)

	calls = nil
	if err := c3.DeleteFederationProvider(ctx, "r", "p1"); err != nil {
		t.Fatal(err)
	}
	rxExpect(t, calls[0], "DELETE", "/realms/r/federation/providers/p1", "")
}

func TestUserAttributes(t *testing.T) {
	var calls []rxCall
	list := `{"data":[{"id":"i","user_id":"u","key":"k","value":"v"}]}`
	c, srv := newTestClient(t, rxServer(t, 200, list, &calls))
	defer srv.Close()
	ctx := context.Background()

	got, err := c.ListUserAttributes(ctx, "r", "u")
	if err != nil || len(got) != 1 || got[0].Key != "k" || got[0].Value != "v" {
		t.Fatalf("list: %v %+v", err, got)
	}
	rxExpect(t, calls[0], "GET", "/realms/r/users/u/attributes", "")

	calls = nil
	if got, err = c.SetUserAttributes(ctx, "r", "u", map[string]string{"k": "v"}); err != nil || len(got) != 1 {
		t.Fatalf("set: %v %+v", err, got)
	}
	rxExpect(t, calls[0], "PUT", "/realms/r/users/u/attributes", `{"attributes":{"k":"v"}}`)

	calls = nil
	if err := c.DeleteUserAttribute(ctx, "r", "u", "a/b c"); err != nil {
		t.Fatal(err)
	}
	rxExpect(t, calls[0], "DELETE", "/realms/r/users/u/attributes/a%2Fb%20c", "")
}
