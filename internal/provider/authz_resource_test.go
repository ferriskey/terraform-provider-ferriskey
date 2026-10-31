package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func authzRealmName() string {
	return "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
}

func authzSchemaSource(extraAction string) string {
	return fmt.Sprintf(`namespace App {
  entity Doc;
  action read appliesTo { principal: [FerrisKey::User], resource: [Doc] };%s
}`, extraAction)
}

func TestAccAuthzStateResource(t *testing.T) {
	realm := authzRealmName()
	cfg := func(enabled bool) string {
		return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_authz_state" "test" {
  realm   = ferriskey_realm.test.name
  enabled = %t
}
`, realm, enabled)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_authz_state.test", "enabled", "true"),
					resource.TestCheckResourceAttr("ferriskey_authz_state.test", "id", realm),
					resource.TestCheckResourceAttrSet("ferriskey_authz_state.test", "version"),
				),
			},
			{
				ResourceName:      "ferriskey_authz_state.test",
				ImportState:       true,
				ImportStateId:     realm,
				ImportStateVerify: true,
			},
			{
				Config: cfg(false),
				Check:  resource.TestCheckResourceAttr("ferriskey_authz_state.test", "enabled", "false"),
			},
		},
	})
}

func TestAccAuthzSchemaResource(t *testing.T) {
	realm := authzRealmName()
	cfg := func(source string) string {
		return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_authz_schema" "test" {
  realm  = ferriskey_realm.test.name
  source = %q
}

data "ferriskey_authz_builtin_schema" "test" {
  realm = ferriskey_authz_schema.test.realm
}
`, realm, source)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(authzSchemaSource("")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_authz_schema.test", "namespace", "App"),
					resource.TestCheckResourceAttr("ferriskey_authz_schema.test", "source", authzSchemaSource("")),
					resource.TestCheckResourceAttrSet("ferriskey_authz_schema.test", "updated_at"),
					resource.TestCheckResourceAttr("data.ferriskey_authz_builtin_schema.test", "namespace", "FerrisKey"),
					resource.TestMatchResourceAttr("data.ferriskey_authz_builtin_schema.test", "source", regexp.MustCompile(`namespace FerrisKey`)),
				),
			},
			{
				ResourceName:      "ferriskey_authz_schema.test",
				ImportState:       true,
				ImportStateId:     realm,
				ImportStateVerify: true,
			},
			{
				Config: cfg(authzSchemaSource("\n  action write appliesTo { principal: [FerrisKey::User], resource: [Doc] };")),
				Check:  resource.TestMatchResourceAttr("ferriskey_authz_schema.test", "source", regexp.MustCompile(`action write`)),
			},
		},
	})
}

func TestAccAuthzTemplateResource(t *testing.T) {
	realm := authzRealmName()
	cfg := func(description, action string) string {
		return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_authz_schema" "test" {
  realm  = ferriskey_realm.test.name
  source = %q
}

resource "ferriskey_authz_template" "test" {
  realm       = ferriskey_authz_schema.test.realm
  name        = "role-can-read"
  description = %q
  source      = "permit(principal == ?principal, action == App::Action::\"%s\", resource);"
}
`, realm, authzSchemaSource(""), description, action)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("first", "read"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_authz_template.test", "name", "role-can-read"),
					resource.TestCheckResourceAttr("ferriskey_authz_template.test", "description", "first"),
					resource.TestMatchResourceAttr("ferriskey_authz_template.test", "id", regexp.MustCompile("^"+regexp.QuoteMeta(realm)+"/[0-9a-f-]+$")),
					resource.TestCheckResourceAttrSet("ferriskey_authz_template.test", "template_id"),
				),
			},
			{
				ResourceName:      "ferriskey_authz_template.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_authz_template.test"),
				ImportStateVerify: true,
			},
			{
				Config: cfg("second", "read"),
				Check:  resource.TestCheckResourceAttr("ferriskey_authz_template.test", "description", "second"),
			},
		},
	})
}

func TestAccAuthzPolicyResource(t *testing.T) {
	realm := authzRealmName()
	cfg := func(enabled bool, description string) string {
		return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_authz_schema" "test" {
  realm  = ferriskey_realm.test.name
  source = %q
}

resource "ferriskey_authz_template" "test" {
  realm  = ferriskey_authz_schema.test.realm
  name   = "role-can-read"
  source = "permit(principal == ?principal, action == App::Action::\"read\", resource);"
}

resource "ferriskey_authz_policy" "static" {
  realm       = ferriskey_authz_schema.test.realm
  name        = "allow-read"
  description = %q
  enabled     = %t
  source      = "permit(principal, action == App::Action::\"read\", resource);   // keep spacing"
}

resource "ferriskey_authz_policy" "linked" {
  realm       = ferriskey_authz_schema.test.realm
  name        = "editors-read"
  template_id = ferriskey_authz_template.test.template_id
  links = {
    "?principal" = {
      type = "FerrisKey::Role"
      id   = "editor"
    }
  }
}

resource "ferriskey_authz_policy" "labelled" {
  realm      = ferriskey_authz_schema.test.realm
  name       = "labelled"
  managed_by = "terraform"
  source     = "permit(principal, action == App::Action::\"read\", resource);"
}
`, realm, authzSchemaSource(""), description, enabled)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(true, "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_authz_policy.static", "origin", "custom"),
					resource.TestCheckResourceAttr("ferriskey_authz_policy.static", "enabled", "true"),
					resource.TestCheckResourceAttr("ferriskey_authz_policy.static", "source", `permit(principal, action == App::Action::"read", resource);   // keep spacing`),
					resource.TestCheckResourceAttr("ferriskey_authz_policy.linked", "origin", "generated"),
					resource.TestCheckResourceAttr("ferriskey_authz_policy.linked", "links.%", "1"),
					resource.TestCheckResourceAttr("ferriskey_authz_policy.linked", "links.?principal.id", "editor"),
					resource.TestCheckResourceAttrPair("ferriskey_authz_policy.linked", "template_id", "ferriskey_authz_template.test", "template_id"),
					resource.TestCheckResourceAttr("ferriskey_authz_policy.labelled", "managed_by", "terraform"),
				),
			},
			{
				ResourceName:      "ferriskey_authz_policy.static",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_authz_policy.static"),
				ImportStateVerify: true,
			},
			{
				ResourceName:      "ferriskey_authz_policy.linked",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_authz_policy.linked"),
				ImportStateVerify: true,
			},
			{
				Config: cfg(false, "second"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_authz_policy.static", "enabled", "false"),
					resource.TestCheckResourceAttr("ferriskey_authz_policy.static", "description", "second"),
				),
			},
		},
	})
}
