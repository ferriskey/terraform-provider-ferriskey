package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

type orgExtCall struct {
	method string
	path   string
	query  string
	body   string
}

func orgExtServer(t *testing.T, status int, resp string, calls *[]orgExtCall) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*calls = append(*calls, orgExtCall{r.Method, r.URL.Path, r.URL.RawQuery, string(b)})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	})
}

func orgExtExpect(t *testing.T, got []orgExtCall, method, path, body string) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("expected 1 call, got %d", len(got))
	}
	if got[0].method != method || got[0].path != path {
		t.Errorf("got %s %s, want %s %s", got[0].method, got[0].path, method, path)
	}
	if body != "" && got[0].body != body {
		t.Errorf("got body %s, want %s", got[0].body, body)
	}
}

const orgExtPrefix = "/realms/r/organizations/o1"

func TestOrgExt_AttributeEndpoints(t *testing.T) {
	var calls []orgExtCall
	c, srv := newTestClient(t, orgExtServer(t, 200, `{"data":[{"id":"a1","organization_id":"o1","key":"k","value":"v"}]}`, &calls))
	defer srv.Close()
	ctx := context.Background()

	attrs, err := c.ListOrganizationAttributes(ctx, "r", "o1")
	if err != nil || len(attrs) != 1 || attrs[0].Key != "k" || attrs[0].Value != "v" {
		t.Fatalf("list: %v %+v", err, attrs)
	}
	orgExtExpect(t, calls, "GET", orgExtPrefix+"/attributes", "")

	calls = nil
	c2, srv2 := newTestClient(t, orgExtServer(t, 200, `{"id":"a1","organization_id":"o1","key":"k","value":"v2"}`, &calls))
	defer srv2.Close()
	attr, err := c2.UpsertOrganizationAttribute(ctx, "r", "o1", "k", "v2")
	if err != nil || attr.Value != "v2" {
		t.Fatalf("upsert: %v %+v", err, attr)
	}
	orgExtExpect(t, calls, "PUT", orgExtPrefix+"/attributes/k", `{"value":"v2"}`)
}

func TestOrgExt_DeleteEndpoints(t *testing.T) {
	cases := map[string]struct {
		path string
		call func(c *Client) error
	}{
		"attribute": {orgExtPrefix + "/attributes/k", func(c *Client) error {
			return c.DeleteOrganizationAttribute(context.Background(), "r", "o1", "k")
		}},
		"member role": {orgExtPrefix + "/members/u1/roles/r1", func(c *Client) error {
			return c.RemoveOrganizationMemberRole(context.Background(), "r", "o1", "u1", "r1")
		}},
		"group": {orgExtPrefix + "/groups/g1", func(c *Client) error {
			return c.DeleteOrganizationGroup(context.Background(), "r", "o1", "g1")
		}},
		"group attribute": {orgExtPrefix + "/groups/g1/attributes/k", func(c *Client) error {
			return c.DeleteOrganizationGroupAttribute(context.Background(), "r", "o1", "g1", "k")
		}},
		"group member": {orgExtPrefix + "/groups/g1/members/u1", func(c *Client) error {
			return c.RemoveOrganizationGroupMember(context.Background(), "r", "o1", "g1", "u1")
		}},
		"group role": {orgExtPrefix + "/groups/g1/roles/r1", func(c *Client) error {
			return c.RemoveOrganizationGroupRole(context.Background(), "r", "o1", "g1", "r1")
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var calls []orgExtCall
			c, srv := newTestClient(t, orgExtServer(t, 204, ``, &calls))
			defer srv.Close()
			if err := tc.call(c); err != nil {
				t.Fatal(err)
			}
			orgExtExpect(t, calls, "DELETE", tc.path, "")
		})
	}
}

func TestOrgExt_DeleteNotFound(t *testing.T) {
	var calls []orgExtCall
	c, srv := newTestClient(t, orgExtServer(t, 404, ``, &calls))
	defer srv.Close()
	err := c.RemoveOrganizationGroupMember(context.Background(), "r", "o1", "g1", "u1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestOrgExt_RoleEndpoints(t *testing.T) {
	roles := `[{"id":"r1","name":"admin","permissions":[]}]`
	t.Run("member list", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 200, roles, &calls))
		defer srv.Close()
		got, err := c.ListOrganizationMemberRoles(context.Background(), "r", "o1", "u1")
		if err != nil || len(got) != 1 || got[0].ID != "r1" {
			t.Fatalf("%v %+v", err, got)
		}
		orgExtExpect(t, calls, "GET", orgExtPrefix+"/members/u1/roles", "")
	})
	t.Run("member assign", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 204, ``, &calls))
		defer srv.Close()
		if err := c.AssignOrganizationMemberRole(context.Background(), "r", "o1", "u1", "r1"); err != nil {
			t.Fatal(err)
		}
		orgExtExpect(t, calls, "POST", orgExtPrefix+"/members/u1/roles", `{"role_id":"r1"}`)
	})
	t.Run("group list", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 200, roles, &calls))
		defer srv.Close()
		got, err := c.ListOrganizationGroupRoles(context.Background(), "r", "o1", "g1")
		if err != nil || len(got) != 1 || got[0].Name != "admin" {
			t.Fatalf("%v %+v", err, got)
		}
		orgExtExpect(t, calls, "GET", orgExtPrefix+"/groups/g1/roles", "")
	})
	t.Run("group assign", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 204, ``, &calls))
		defer srv.Close()
		if err := c.AssignOrganizationGroupRole(context.Background(), "r", "o1", "g1", "r1"); err != nil {
			t.Fatal(err)
		}
		orgExtExpect(t, calls, "POST", orgExtPrefix+"/groups/g1/roles", `{"role_id":"r1"}`)
	})
}

func TestOrgExt_GroupCRUD(t *testing.T) {
	grp := `{"id":"g1","organization_id":"o1","parent_group_id":"p1","name":"n","description":"d"}`
	ctx := context.Background()
	t.Run("create", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 201, grp, &calls))
		defer srv.Close()
		desc, parent := "d", "p1"
		g, err := c.CreateOrganizationGroup(ctx, "r", "o1", CreateOrgGroupRequest{Name: "n", Description: &desc, ParentGroupID: &parent})
		if err != nil || g.ID != "g1" || g.ParentGroupID == nil || *g.ParentGroupID != "p1" {
			t.Fatalf("%v %+v", err, g)
		}
		orgExtExpect(t, calls, "POST", orgExtPrefix+"/groups", `{"name":"n","description":"d","parent_group_id":"p1"}`)
	})
	t.Run("get", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 200, grp, &calls))
		defer srv.Close()
		g, err := c.GetOrganizationGroup(ctx, "r", "o1", "g1")
		if err != nil || g.Name != "n" {
			t.Fatalf("%v %+v", err, g)
		}
		orgExtExpect(t, calls, "GET", orgExtPrefix+"/groups/g1", "")
	})
	t.Run("update", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 200, grp, &calls))
		defer srv.Close()
		name, desc := "n2", ""
		if _, err := c.UpdateOrganizationGroup(ctx, "r", "o1", "g1", UpdateOrgGroupRequest{Name: &name, Description: &desc}); err != nil {
			t.Fatal(err)
		}
		orgExtExpect(t, calls, "PUT", orgExtPrefix+"/groups/g1", `{"name":"n2","description":""}`)
	})
}

func TestOrgExt_ListGroupsPaginates(t *testing.T) {
	var queries []string
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		page := map[string]string{
			"limit=100&page=1": `{"data":[{"id":"g1","name":"a"}],"metadata":{"total_pages":2}}`,
			"limit=100&page=2": `{"data":[{"id":"g2","name":"b"}],"metadata":{"total_pages":2}}`,
		}[r.URL.RawQuery]
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()
	got, err := c.ListOrganizationGroups(context.Background(), "r", "o1")
	if err != nil || len(got) != 2 || got[1].ID != "g2" {
		t.Fatalf("%v %+v", err, got)
	}
	if len(queries) != 2 {
		t.Errorf("expected 2 requests, got %v", queries)
	}
}

func TestOrgExt_GroupAttributeEndpoints(t *testing.T) {
	ctx := context.Background()
	t.Run("list bare array", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 200, `[{"id":"a","group_id":"g1","key":"k","value":"v"}]`, &calls))
		defer srv.Close()
		got, err := c.ListOrganizationGroupAttributes(ctx, "r", "o1", "g1")
		if err != nil || len(got) != 1 || got[0].GroupID != "g1" {
			t.Fatalf("%v %+v", err, got)
		}
		orgExtExpect(t, calls, "GET", orgExtPrefix+"/groups/g1/attributes", "")
	})
	t.Run("upsert", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 200, `{"id":"a","group_id":"g1","key":"k","value":"v"}`, &calls))
		defer srv.Close()
		got, err := c.UpsertOrganizationGroupAttribute(ctx, "r", "o1", "g1", "k", "v")
		if err != nil || got.Value != "v" {
			t.Fatalf("%v %+v", err, got)
		}
		orgExtExpect(t, calls, "PUT", orgExtPrefix+"/groups/g1/attributes/k", `{"value":"v"}`)
	})
}

func TestOrgExt_GroupMemberEndpoints(t *testing.T) {
	ctx := context.Background()
	t.Run("list paginated", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 200, `{"data":[{"id":"m","group_id":"g1","user_id":"u1","username":"alice"}],"metadata":{"total_pages":1}}`, &calls))
		defer srv.Close()
		got, err := c.ListOrganizationGroupMembers(ctx, "r", "o1", "g1")
		if err != nil || len(got) != 1 || got[0].UserID != "u1" {
			t.Fatalf("%v %+v", err, got)
		}
		orgExtExpect(t, calls, "GET", orgExtPrefix+"/groups/g1/members", "")
		if calls[0].query != "limit=100&page=1" {
			t.Errorf("query %q", calls[0].query)
		}
	})
	t.Run("add", func(t *testing.T) {
		var calls []orgExtCall
		c, srv := newTestClient(t, orgExtServer(t, 201, `{"id":"m","group_id":"g1","user_id":"u1"}`, &calls))
		defer srv.Close()
		if err := c.AddOrganizationGroupMember(ctx, "r", "o1", "g1", "u1"); err != nil {
			t.Fatal(err)
		}
		orgExtExpect(t, calls, "POST", orgExtPrefix+"/groups/g1/members", `{"user_id":"u1"}`)
	})
}

func TestOrgExt_RequestBodiesAreValidJSON(t *testing.T) {
	b, _ := json.Marshal(UpdateOrgGroupRequest{})
	if string(b) != `{}` {
		t.Errorf("empty update should marshal to {}, got %s", b)
	}
}
