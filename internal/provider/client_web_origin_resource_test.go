package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientWebOriginResource(t *testing.T) {
	realm := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientWebOriginConfig(realm, "https://a.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_web_origin.test", "value", "https://a.example.com"),
					resource.TestCheckResourceAttrSet("ferriskey_client_web_origin.test", "web_origin_id"),
				),
			},
			{
				ResourceName:      "ferriskey_client_web_origin.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_web_origin.test"),
				ImportStateVerify: true,
			},
			{
				Config: testAccClientWebOriginConfig(realm, "https://b.example.com"),
				Check:  resource.TestCheckResourceAttr("ferriskey_client_web_origin.test", "value", "https://b.example.com"),
			},
		},
	})
}

func testAccClientWebOriginConfig(realm, value string) string {
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

resource "ferriskey_client_web_origin" "test" {
  realm     = ferriskey_realm.test.name
  client_id = ferriskey_client.test.client_uuid
  value     = %q
}
`, realm, value)
}
