package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

type samlmCall struct {
	method string
	path   string
	body   map[string]any
}

func samlmServer(t *testing.T, status int, response string, got *samlmCall) (*Client, func()) {
	t.Helper()
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method = r.Method
		got.path = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		if len(raw) > 0 {
			got.body = map[string]any{}
			_ = json.Unmarshal(raw, &got.body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	return c, srv.Close
}

func samlmExpect(t *testing.T, got samlmCall, method, path string) {
	t.Helper()
	if got.method != method || got.path != path {
		t.Fatalf("expected %s %s, got %s %s", method, path, got.method, got.path)
	}
}

func TestSamlConfig(t *testing.T) {
	var got samlmCall
	c, done := samlmServer(t, 200, `{"client_id":"c1","sp_entity_id":"urn:sp","acs_url":"https://sp/acs","name_id_format":"f","sign_assertions":true,"sign_documents":false,"want_authn_requests_signed":true}`, &got)
	defer done()

	cfg, err := c.GetClientSamlConfig(context.Background(), "r", "c1")
	if err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "GET", "/realms/r/clients/c1/saml-config")
	if cfg.SpEntityID != "urn:sp" || !cfg.SignAssertions || !cfg.WantAuthnRequestsSigned {
		t.Fatalf("unexpected config %+v", cfg)
	}

	cfg, err = c.SetClientSamlConfig(context.Background(), "r", "c1", SetClientSamlConfigRequest{
		SpEntityID: "urn:sp", AcsURL: "https://sp/acs", NameIDFormat: "f", SignAssertions: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "PUT", "/realms/r/clients/c1/saml-config")
	if got.body["sp_entity_id"] != "urn:sp" || got.body["sign_documents"] != false || got.body["sign_assertions"] != true {
		t.Fatalf("unexpected body %v", got.body)
	}
	if cfg.AcsURL != "https://sp/acs" {
		t.Fatalf("unexpected config %+v", cfg)
	}
}

func TestSamlConfig_NotFound(t *testing.T) {
	var got samlmCall
	c, done := samlmServer(t, 404, `{"code":"E_NOT_FOUND"}`, &got)
	defer done()
	if _, err := c.GetClientSamlConfig(context.Background(), "r", "c1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSamlAttributeMappers(t *testing.T) {
	var got samlmCall
	c, done := samlmServer(t, 200, `[{"id":"m1","client_id":"c1","name":"mail","name_format":"f","source":"user:email"}]`, &got)
	defer done()

	list, err := c.ListSamlAttributeMappers(context.Background(), "r", "c1")
	if err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "GET", "/realms/r/clients/c1/saml-attribute-mappers")
	if len(list) != 1 || list[0].ID != "m1" || list[0].Source != "user:email" {
		t.Fatalf("unexpected list %+v", list)
	}

	c2, done2 := samlmServer(t, 201, `{"id":"m2","name":"mail","name_format":"f","source":"user:email"}`, &got)
	defer done2()
	m, err := c2.CreateSamlAttributeMapper(context.Background(), "r", "c1", CreateSamlAttributeMapperRequest{Name: "mail", Source: "user:email"})
	if err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "POST", "/realms/r/clients/c1/saml-attribute-mappers")
	if got.body["name"] != "mail" || got.body["source"] != "user:email" {
		t.Fatalf("unexpected body %v", got.body)
	}
	if _, ok := got.body["name_format"]; ok {
		t.Fatalf("empty name_format must be omitted: %v", got.body)
	}
	if m.ID != "m2" {
		t.Fatalf("unexpected mapper %+v", m)
	}

	c3, done3 := samlmServer(t, 200, `null`, &got)
	defer done3()
	if err := c3.DeleteSamlAttributeMapper(context.Background(), "r", "c1", "m2"); err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "DELETE", "/realms/r/clients/c1/saml-attribute-mappers/m2")
}

func TestClientMaintenance(t *testing.T) {
	var got samlmCall
	c, done := samlmServer(t, 200, `{"message":"Maintenance mode enabled"}`, &got)
	defer done()
	reason, strategy := "upgrade", "terminate"
	err := c.ToggleClientMaintenance(context.Background(), "r", "c1", ToggleClientMaintenanceRequest{Enabled: true, Reason: &reason, SessionStrategy: &strategy})
	if err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "PUT", "/realms/r/clients/c1/maintenance")
	if got.body["enabled"] != true || got.body["reason"] != "upgrade" || got.body["session_strategy"] != "terminate" {
		t.Fatalf("unexpected body %v", got.body)
	}

	c2, done2 := samlmServer(t, 200, `{"data":{"id":"c1","maintenance_enabled":true,"maintenance_reason":"upgrade","maintenance_session_strategy":"terminate"}}`, &got)
	defer done2()
	st, err := c2.GetClientMaintenance(context.Background(), "r", "c1")
	if err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "GET", "/realms/r/clients/c1")
	if !st.Enabled || st.Reason == nil || *st.Reason != "upgrade" || st.SessionStrategy != "terminate" {
		t.Fatalf("unexpected state %+v", st)
	}
}

func TestClientMaintenanceWhitelist(t *testing.T) {
	var got samlmCall
	c, done := samlmServer(t, 200, `{"data":[{"id":"e1","client_id":"c1","user_id":"u1","role_id":null}]}`, &got)
	defer done()
	list, err := c.ListClientMaintenanceWhitelist(context.Background(), "r", "c1")
	if err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "GET", "/realms/r/clients/c1/maintenance/whitelist")
	if len(list) != 1 || list[0].UserID == nil || *list[0].UserID != "u1" || list[0].RoleID != nil {
		t.Fatalf("unexpected list %+v", list)
	}

	c2, done2 := samlmServer(t, 201, `{"data":{"id":"e2","client_id":"c1","user_id":null,"role_id":"r1"}}`, &got)
	defer done2()
	role := "r1"
	e, err := c2.AddClientMaintenanceWhitelistEntry(context.Background(), "r", "c1", AddMaintenanceWhitelistEntryRequest{RoleID: &role})
	if err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "POST", "/realms/r/clients/c1/maintenance/whitelist")
	if got.body["role_id"] != "r1" {
		t.Fatalf("unexpected body %v", got.body)
	}
	if _, ok := got.body["user_id"]; ok {
		t.Fatalf("nil user_id must be omitted: %v", got.body)
	}
	if e.ID != "e2" || e.RoleID == nil {
		t.Fatalf("unexpected entry %+v", e)
	}

	c3, done3 := samlmServer(t, 200, `{"message":"Entry removed"}`, &got)
	defer done3()
	if err := c3.RemoveClientMaintenanceWhitelistEntry(context.Background(), "r", "c1", "e2"); err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "DELETE", "/realms/r/clients/c1/maintenance/whitelist/e2")
}

func TestRealmMaintenanceWhitelist(t *testing.T) {
	var got samlmCall
	c, done := samlmServer(t, 200, `{"data":[{"id":"e1","realm_id":"rid","user_id":"u1","role_id":null}]}`, &got)
	defer done()
	list, err := c.ListRealmMaintenanceWhitelist(context.Background(), "r")
	if err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "GET", "/realms/r/settings/maintenance/whitelist")
	if len(list) != 1 || list[0].RealmID != "rid" {
		t.Fatalf("unexpected list %+v", list)
	}

	c2, done2 := samlmServer(t, 201, `{"data":{"id":"e2","realm_id":"rid","user_id":"u1","role_id":null}}`, &got)
	defer done2()
	user := "u1"
	e, err := c2.AddRealmMaintenanceWhitelistEntry(context.Background(), "r", AddMaintenanceWhitelistEntryRequest{UserID: &user})
	if err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "POST", "/realms/r/settings/maintenance/whitelist")
	if got.body["user_id"] != "u1" || e.ID != "e2" {
		t.Fatalf("unexpected body %v entry %+v", got.body, e)
	}

	c3, done3 := samlmServer(t, 200, `{"message":"Entry removed"}`, &got)
	defer done3()
	if err := c3.RemoveRealmMaintenanceWhitelistEntry(context.Background(), "r", "e2"); err != nil {
		t.Fatal(err)
	}
	samlmExpect(t, got, "DELETE", "/realms/r/settings/maintenance/whitelist/e2")

	c4, done4 := samlmServer(t, 404, ``, &got)
	defer done4()
	if err := c4.RemoveRealmMaintenanceWhitelistEntry(context.Background(), "r", "e2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
