package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientRoleResource(t *testing.T) {
	realm := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientRoleConfig(realm, "reader", "first", `["manage_users"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_role.test", "name", "reader"),
					resource.TestCheckResourceAttr("ferriskey_client_role.test", "permissions.#", "1"),
					resource.TestCheckResourceAttrSet("ferriskey_client_role.test", "role_id"),
					resource.TestMatchResourceAttr("ferriskey_client_role.test", "id", regexp.MustCompile("^"+regexp.QuoteMeta(realm)+"/[0-9a-f-]+/[0-9a-f-]+$")),
				),
			},
			{
				ResourceName:      "ferriskey_client_role.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_role.test"),
				ImportStateVerify: true,
			},
			{
				Config: testAccClientRoleConfig(realm, "reader2", "second", `["manage_users", "view_users"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_role.test", "name", "reader2"),
					resource.TestCheckResourceAttr("ferriskey_client_role.test", "description", "second"),
					resource.TestCheckResourceAttr("ferriskey_client_role.test", "permissions.#", "2"),
				),
			},
		},
	})
}

func testAccClientRoleConfig(realm, name, description, permissions string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_client" "test" {
  realm       = ferriskey_realm.test.name
  client_id   = "app"
  name        = "App"
  client_type = "confidential"
}

resource "ferriskey_client_role" "test" {
  realm       = ferriskey_realm.test.name
  client_id   = ferriskey_client.test.client_uuid
  name        = %q
  description = %q
  permissions = %s
}
`, realm, name, description, permissions)
}
