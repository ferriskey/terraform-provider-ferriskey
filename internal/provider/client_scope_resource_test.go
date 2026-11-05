package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func scopeAccBase(realm string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}
`, realm)
}

func TestAccClientScopeResource(t *testing.T) {
	realm := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	cfg := func(name, desc string, isDefault bool) string {
		return scopeAccBase(realm) + fmt.Sprintf(`
resource "ferriskey_client_scope" "test" {
  realm       = ferriskey_realm.test.name
  name        = %q
  description = %q
  is_default  = %t
}
`, name, desc, isDefault)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("tf-scope", "first", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_scope.test", "name", "tf-scope"),
					resource.TestCheckResourceAttr("ferriskey_client_scope.test", "protocol", "openid-connect"),
					resource.TestCheckResourceAttr("ferriskey_client_scope.test", "is_default", "false"),
					resource.TestMatchResourceAttr("ferriskey_client_scope.test", "id", regexp.MustCompile("^"+regexp.QuoteMeta(realm)+"/[0-9a-f-]+$")),
				),
			},
			{
				ResourceName:      "ferriskey_client_scope.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_scope.test"),
				ImportStateVerify: true,
			},
			{
				Config: cfg("tf-scope-renamed", "second", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_scope.test", "name", "tf-scope-renamed"),
					resource.TestCheckResourceAttr("ferriskey_client_scope.test", "description", "second"),
					resource.TestCheckResourceAttr("ferriskey_client_scope.test", "is_default", "true"),
				),
			},
		},
	})
}

func TestAccClientScopeProtocolMapperResource(t *testing.T) {
	realm := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	cfg := func(name, claim string) string {
		return scopeAccBase(realm) + fmt.Sprintf(`
resource "ferriskey_client_scope" "test" {
  realm = ferriskey_realm.test.name
  name  = "tf-mapper-scope"
}

resource "ferriskey_client_scope_protocol_mapper" "test" {
  realm       = ferriskey_realm.test.name
  scope_id    = ferriskey_client_scope.test.scope_id
  name        = %q
  mapper_type = "oidc-usermodel-attribute-mapper"
  config = {
    "claim.name"     = %q
    "user.attribute" = "email"
  }
}
`, name, claim)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("m1", "mail"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_scope_protocol_mapper.test", "name", "m1"),
					resource.TestCheckResourceAttr("ferriskey_client_scope_protocol_mapper.test", "config.claim.name", "mail"),
					resource.TestCheckResourceAttrSet("ferriskey_client_scope_protocol_mapper.test", "mapper_id"),
				),
			},
			{
				ResourceName:      "ferriskey_client_scope_protocol_mapper.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_scope_protocol_mapper.test"),
				ImportStateVerify: true,
			},
			{
				Config: cfg("m1-renamed", "email_claim"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_scope_protocol_mapper.test", "name", "m1-renamed"),
					resource.TestCheckResourceAttr("ferriskey_client_scope_protocol_mapper.test", "config.claim.name", "email_claim"),
				),
			},
		},
	})
}

func TestAccClientScopeAssignmentResource(t *testing.T) {
	realm := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	cfg := func(assignmentType string) string {
		return scopeAccBase(realm) + fmt.Sprintf(`
resource "ferriskey_client" "test" {
  realm       = ferriskey_realm.test.name
  client_id   = "tf-scope-app"
  name        = "tf-scope-app"
  client_type = "public"
}

resource "ferriskey_client_scope" "test" {
  realm = ferriskey_realm.test.name
  name  = "tf-assign-scope"
}

resource "ferriskey_client_scope_assignment" "test" {
  realm     = ferriskey_realm.test.name
  client_id = ferriskey_client.test.client_uuid
  scope_id  = ferriskey_client_scope.test.scope_id
  type      = %q
}
`, assignmentType)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("optional"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_scope_assignment.test", "type", "optional"),
					resource.TestCheckResourceAttrSet("ferriskey_client_scope_assignment.test", "id"),
				),
			},
			{
				ResourceName:      "ferriskey_client_scope_assignment.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_scope_assignment.test"),
				ImportStateVerify: true,
			},
			{
				Config: cfg("default"),
				Check:  resource.TestCheckResourceAttr("ferriskey_client_scope_assignment.test", "type", "default"),
			},
		},
	})
}
