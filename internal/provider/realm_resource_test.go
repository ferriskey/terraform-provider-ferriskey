package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRealmResource(t *testing.T) {
	name := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	renamed := name + "-renamed"

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
			{ // Rename in place
				Config: testAccRealmConfig(renamed),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_realm.test", "name", renamed),
				),
			},
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
