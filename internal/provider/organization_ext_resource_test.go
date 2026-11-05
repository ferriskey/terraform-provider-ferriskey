package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func orgExtBaseConfig(realm string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_organization" "test" {
  realm = ferriskey_realm.test.name
  name  = "Acme"
  alias = "acme"
}

resource "ferriskey_user" "test" {
  realm    = ferriskey_realm.test.name
  username = "alice"
  email    = "alice@example.com"
}

resource "ferriskey_role" "test" {
  realm       = ferriskey_realm.test.name
  name        = "org-role"
  permissions = ["view_users"]
}

resource "ferriskey_organization_member" "test" {
  realm           = ferriskey_realm.test.name
  organization_id = ferriskey_organization.test.organization_uuid
  user_id         = ferriskey_user.test.user_uuid
}

resource "ferriskey_organization_group" "test" {
  realm           = ferriskey_realm.test.name
  organization_id = ferriskey_organization.test.organization_uuid
  name            = "engineering"
}
`, realm)
}

func orgExtRealmName() string {
	return "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
}

func TestAccOrganizationAttributeResource(t *testing.T) {
	realm := orgExtRealmName()
	config := func(value string) string {
		return orgExtBaseConfig(realm) + fmt.Sprintf(`
resource "ferriskey_organization_attribute" "test" {
  realm           = ferriskey_realm.test.name
  organization_id = ferriskey_organization.test.organization_uuid
  key             = "tier"
  value           = %q
}
`, value)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("gold"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_organization_attribute.test", "value", "gold"),
					resource.TestCheckResourceAttrSet("ferriskey_organization_attribute.test", "id"),
				),
			},
			{
				ResourceName:      "ferriskey_organization_attribute.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_organization_attribute.test"),
				ImportStateVerify: true,
			},
			{
				Config: config("platinum"),
				Check:  resource.TestCheckResourceAttr("ferriskey_organization_attribute.test", "value", "platinum"),
			},
		},
	})
}

func TestAccOrganizationMemberRoleResource(t *testing.T) {
	realm := orgExtRealmName()
	config := orgExtBaseConfig(realm) + `
resource "ferriskey_organization_member_role" "test" {
  realm           = ferriskey_realm.test.name
  organization_id = ferriskey_organization.test.organization_uuid
  user_id         = ferriskey_organization_member.test.user_id
  role_id         = ferriskey_role.test.role_uuid
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttrSet("ferriskey_organization_member_role.test", "id"),
			},
			{
				ResourceName:      "ferriskey_organization_member_role.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_organization_member_role.test"),
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func TestAccOrganizationGroupResource(t *testing.T) {
	realm := orgExtRealmName()
	config := func(name, description string) string {
		return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_organization" "test" {
  realm = ferriskey_realm.test.name
  name  = "Acme"
  alias = "acme"
}

resource "ferriskey_organization_group" "parent" {
  realm           = ferriskey_realm.test.name
  organization_id = ferriskey_organization.test.organization_uuid
  name            = %q
  description     = %q
}

resource "ferriskey_organization_group" "child" {
  realm           = ferriskey_realm.test.name
  organization_id = ferriskey_organization.test.organization_uuid
  name            = "child"
  parent_group_id = ferriskey_organization_group.parent.group_uuid
}
`, realm, name, description)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("parent", "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_organization_group.parent", "name", "parent"),
					resource.TestCheckResourceAttrSet("ferriskey_organization_group.parent", "group_uuid"),
					resource.TestCheckResourceAttrPair("ferriskey_organization_group.child", "parent_group_id", "ferriskey_organization_group.parent", "group_uuid"),
				),
			},
			{
				ResourceName:      "ferriskey_organization_group.child",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_organization_group.child"),
				ImportStateVerify: true,
			},
			{
				Config: config("parent-renamed", "second"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_organization_group.parent", "name", "parent-renamed"),
					resource.TestCheckResourceAttr("ferriskey_organization_group.parent", "description", "second"),
				),
			},
		},
	})
}

func TestAccOrganizationGroupAttributeResource(t *testing.T) {
	realm := orgExtRealmName()
	config := func(value string) string {
		return orgExtBaseConfig(realm) + fmt.Sprintf(`
resource "ferriskey_organization_group_attribute" "test" {
  realm           = ferriskey_realm.test.name
  organization_id = ferriskey_organization.test.organization_uuid
  group_id        = ferriskey_organization_group.test.group_uuid
  key             = "cost-center"
  value           = %q
}
`, value)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("cc-1"),
				Check:  resource.TestCheckResourceAttr("ferriskey_organization_group_attribute.test", "value", "cc-1"),
			},
			{
				ResourceName:      "ferriskey_organization_group_attribute.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_organization_group_attribute.test"),
				ImportStateVerify: true,
			},
			{
				Config: config("cc-2"),
				Check:  resource.TestCheckResourceAttr("ferriskey_organization_group_attribute.test", "value", "cc-2"),
			},
		},
	})
}

func TestAccOrganizationGroupMemberResource(t *testing.T) {
	realm := orgExtRealmName()
	config := orgExtBaseConfig(realm) + `
resource "ferriskey_organization_group_member" "test" {
  realm           = ferriskey_realm.test.name
  organization_id = ferriskey_organization.test.organization_uuid
  group_id        = ferriskey_organization_group.test.group_uuid
  user_id         = ferriskey_user.test.user_uuid
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttrSet("ferriskey_organization_group_member.test", "id"),
			},
			{
				ResourceName:      "ferriskey_organization_group_member.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_organization_group_member.test"),
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

func TestAccOrganizationGroupRoleResource(t *testing.T) {
	realm := orgExtRealmName()
	config := orgExtBaseConfig(realm) + `
resource "ferriskey_organization_group_role" "test" {
  realm           = ferriskey_realm.test.name
  organization_id = ferriskey_organization.test.organization_uuid
  group_id        = ferriskey_organization_group.test.group_uuid
  role_id         = ferriskey_role.test.role_uuid
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttrSet("ferriskey_organization_group_role.test", "id"),
			},
			{
				ResourceName:      "ferriskey_organization_group_role.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_organization_group_role.test"),
				ImportStateVerify: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}
