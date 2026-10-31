package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

type scopeCall struct {
	method string
	path   string
	body   string
}

func scopeRecorder(status int, resp string, calls *[]scopeCall) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*calls = append(*calls, scopeCall{r.Method, r.URL.Path, string(b)})
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	})
}

func TestClientScope_CRUD(t *testing.T) {
	var calls []scopeCall
	c, srv := newTestClient(t, scopeRecorder(200, `{"id":"s1","name":"n","protocol":"openid-connect","default_scope_type":"DEFAULT","dynamic_registration_allowed":true,"description":"d"}`, &calls))
	defer srv.Close()
	ctx := context.Background()

	got, err := c.CreateClientScope(ctx, "r", CreateClientScopeRequest{Name: "n", Protocol: "openid-connect", IsDefault: true})
	if err != nil || got.ID != "s1" || got.DefaultScopeType != "DEFAULT" || !got.DynamicRegistrationAllowed {
		t.Fatalf("create: %+v %v", got, err)
	}
	name := "n2"
	if _, err := c.UpdateClientScope(ctx, "r", "s1", UpdateClientScopeRequest{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetClientScope(ctx, "r", "s1"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteClientScope(ctx, "r", "s1"); err != nil {
		t.Fatal(err)
	}

	want := []scopeCall{
		{"POST", "/realms/r/client-scopes", `{"name":"n","protocol":"openid-connect","is_default":true,"dynamic_registration_allowed":false}`},
		{"PATCH", "/realms/r/client-scopes/s1", `{"name":"n2"}`},
		{"GET", "/realms/r/client-scopes/s1", ""},
		{"DELETE", "/realms/r/client-scopes/s1", ""},
	}
	for i, w := range want {
		if calls[i] != w {
			t.Errorf("call %d: got %+v want %+v", i, calls[i], w)
		}
	}
}

func TestGetClientScope_NotFound(t *testing.T) {
	var calls []scopeCall
	c, srv := newTestClient(t, scopeRecorder(404, ``, &calls))
	defer srv.Close()
	if _, err := c.GetClientScope(context.Background(), "r", "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestProtocolMapper_GetFindsEmbedded(t *testing.T) {
	var calls []scopeCall
	c, srv := newTestClient(t, scopeRecorder(200, `{"id":"s1","name":"n","protocol":"p","default_scope_type":"NONE","dynamic_registration_allowed":false,"protocol_mappers":[{"id":"m1","client_scope_id":"s1","name":"a","mapper_type":"t","config":{"k":"v","n":5}}]}`, &calls))
	defer srv.Close()

	m, err := c.GetProtocolMapper(context.Background(), "r", "s1", "m1")
	if err != nil || m.Name != "a" || m.Config["k"] != "v" {
		t.Fatalf("got %+v %v", m, err)
	}
	if _, err := c.GetProtocolMapper(context.Background(), "r", "s1", "zz"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProtocolMapper_Writes(t *testing.T) {
	var calls []scopeCall
	c, srv := newTestClient(t, scopeRecorder(200, `{"id":"m1","client_scope_id":"s1","name":"a","mapper_type":"t","config":{}}`, &calls))
	defer srv.Close()
	ctx := context.Background()

	if m, err := c.CreateProtocolMapper(ctx, "r", "s1", CreateProtocolMapperRequest{Name: "a", MapperType: "t", Config: map[string]any{"k": "v"}}); err != nil || m.ID != "m1" {
		t.Fatalf("create: %+v %v", m, err)
	}
	name := "b"
	if _, err := c.UpdateProtocolMapper(ctx, "r", "s1", "m1", UpdateProtocolMapperRequest{Name: &name, Config: map[string]any{"k": "w"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteProtocolMapper(ctx, "r", "s1", "m1"); err != nil {
		t.Fatal(err)
	}

	want := []scopeCall{
		{"POST", "/realms/r/client-scopes/s1/protocol-mappers", `{"name":"a","mapper_type":"t","config":{"k":"v"}}`},
		{"PATCH", "/realms/r/client-scopes/s1/protocol-mappers/m1", `{"name":"b","config":{"k":"w"}}`},
		{"DELETE", "/realms/r/client-scopes/s1/protocol-mappers/m1", ""},
	}
	for i, w := range want {
		if calls[i] != w {
			t.Errorf("call %d: got %+v want %+v", i, calls[i], w)
		}
	}
}

func TestClientScopeAssignment(t *testing.T) {
	var calls []scopeCall
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, scopeCall{r.Method, r.URL.Path, ""})
		switch r.Method {
		case "GET":
			_ = json.NewEncoder(w).Encode([]ClientScope{{ID: "s1", DefaultScopeType: "OPTIONAL"}})
		case "PUT":
			_, _ = w.Write([]byte(`{"client_id":"c1","scope_id":"s1","default_scope_type":"OPTIONAL"}`))
		default:
			_, _ = w.Write([]byte(`null`))
		}
	}))
	defer srv.Close()
	ctx := context.Background()

	m, err := c.AssignClientScope(ctx, "r", "c1", "s1", ScopeAssignmentOptional)
	if err != nil || m.DefaultScopeType != "OPTIONAL" {
		t.Fatalf("assign: %+v %v", m, err)
	}
	typ, err := c.GetClientScopeAssignment(ctx, "r", "c1", "s1")
	if err != nil || typ != "OPTIONAL" {
		t.Fatalf("get: %q %v", typ, err)
	}
	if _, err := c.GetClientScopeAssignment(ctx, "r", "c1", "other"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := c.UnassignClientScope(ctx, "r", "c1", "s1", ScopeAssignmentDefault); err != nil {
		t.Fatal(err)
	}

	if calls[0] != (scopeCall{"PUT", "/realms/r/clients/c1/optional-client-scopes/s1", ""}) {
		t.Errorf("assign call: %+v", calls[0])
	}
	if calls[1].path != "/realms/r/clients/c1/client-scopes" {
		t.Errorf("list call: %+v", calls[1])
	}
	if calls[3] != (scopeCall{"DELETE", "/realms/r/clients/c1/default-client-scopes/s1", ""}) {
		t.Errorf("unassign call: %+v", calls[3])
	}
}
