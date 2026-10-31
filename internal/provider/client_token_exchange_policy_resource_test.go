package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientTokenExchangePolicyResource(t *testing.T) {
	realm := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientTokenExchangePolicyConfig(realm, "false", `["openid"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_token_exchange_policy.test", "target_audience", "target"),
					resource.TestCheckResourceAttr("ferriskey_client_token_exchange_policy.test", "allow_delegation", "false"),
					resource.TestCheckResourceAttr("ferriskey_client_token_exchange_policy.test", "allowed_scopes.#", "1"),
					resource.TestCheckResourceAttrSet("ferriskey_client_token_exchange_policy.test", "policy_id"),
				),
			},
			{
				ResourceName:      "ferriskey_client_token_exchange_policy.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_token_exchange_policy.test"),
				ImportStateVerify: true,
			},
			{
				Config: testAccClientTokenExchangePolicyConfig(realm, "true", `["openid", "profile"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_token_exchange_policy.test", "allow_delegation", "true"),
					resource.TestCheckResourceAttr("ferriskey_client_token_exchange_policy.test", "allowed_scopes.#", "2"),
				),
			},
		},
	})
}

func testAccClientTokenExchangePolicyConfig(realm, delegation, scopes string) string {
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

resource "ferriskey_client" "target" {
  realm       = ferriskey_realm.test.name
  client_id   = "target"
  name        = "Target"
  client_type = "confidential"
}

resource "ferriskey_client_token_exchange_policy" "test" {
  realm            = ferriskey_realm.test.name
  client_id        = ferriskey_client.test.client_uuid
  target_audience  = ferriskey_client.target.client_id
  allow_delegation = %s
  allowed_scopes   = %s
}
`, realm, delegation, scopes)
}
