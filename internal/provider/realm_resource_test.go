package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRealmResource(t *testing.T) {
	name := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // Create
				Config: testAccRealmConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_realm.test", "name", name),
					resource.TestCheckResourceAttr("ferriskey_realm.test", "id", name),
					resource.TestCheckResourceAttrSet("ferriskey_realm.test", "realm_id"),
					resource.TestCheckResourceAttrSet("ferriskey_realm.test", "created_at"),
				),
			},
			{ // Import
				ResourceName:      "ferriskey_realm.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
			// No rename step: the API answers 403 to the admin on a realm once it is renamed.
		},
	})
}

func testAccRealmConfig(name string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}
`, name)
}
