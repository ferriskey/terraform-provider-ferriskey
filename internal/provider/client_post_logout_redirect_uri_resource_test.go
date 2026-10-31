package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientPostLogoutRedirectURIResource(t *testing.T) {
	realm := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientPostLogoutRedirectURIConfig(realm, "https://a.example.com/out", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_post_logout_redirect_uri.test", "value", "https://a.example.com/out"),
					resource.TestCheckResourceAttr("ferriskey_client_post_logout_redirect_uri.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("ferriskey_client_post_logout_redirect_uri.test", "uri_id"),
				),
			},
			{
				ResourceName:      "ferriskey_client_post_logout_redirect_uri.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_post_logout_redirect_uri.test"),
				ImportStateVerify: true,
			},
			{
				Config: testAccClientPostLogoutRedirectURIConfig(realm, "https://a.example.com/out", false),
				Check:  resource.TestCheckResourceAttr("ferriskey_client_post_logout_redirect_uri.test", "enabled", "false"),
			},
		},
	})
}

func testAccClientPostLogoutRedirectURIConfig(realm, value string, enabled bool) string {
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

resource "ferriskey_client_post_logout_redirect_uri" "test" {
  realm     = ferriskey_realm.test.name
  client_id = ferriskey_client.test.client_uuid
  value     = %q
  enabled   = %t
}
`, realm, value, enabled)
}
