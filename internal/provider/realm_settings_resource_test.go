package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRealmSettingsResource(t *testing.T) {
	realm := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRealmSettingsConfig(realm, 5, `["email", "username"]`, `["en", "fr"]`, `["https://api.example.com"]`, 3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "lockout_threshold", "5"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "login_aliases.0", "email"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "login_aliases.1", "username"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "supported_locales.#", "2"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "default_locale", "fr"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "require_mfa", "true"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "seawatch_pii_mode", "pseudonymise"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "allowed_resources.#", "1"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "webhook_retry_max_attempts", "3"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "dcr_enabled", "true"),
				),
			},
			{
				ResourceName:            "ferriskey_realm_settings.test",
				ImportState:             true,
				ImportStateId:           realm,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"seawatch_pseudo_key", "updated_at"},
			},
			{ // Reorder aliases, shrink lists, clear allowed_resources
				Config: testAccRealmSettingsConfig(realm, 7, `["username"]`, `["fr"]`, `[]`, 5),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "lockout_threshold", "7"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "login_aliases.#", "1"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "supported_locales.#", "1"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "allowed_resources.#", "0"),
					resource.TestCheckResourceAttr("ferriskey_realm_settings.test", "webhook_retry_max_attempts", "5"),
				),
			},
		},
	})
}

func testAccRealmSettingsConfig(realm string, lockout int, aliases, locales, resources string, retries int) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_realm_settings" "test" {
  realm                      = ferriskey_realm.test.name
  lockout_threshold          = %d
  lockout_duration_seconds   = 600
  login_aliases              = %s
  default_locale             = "fr"
  supported_locales          = %s
  require_mfa                = true
  edit_username_enabled      = true
  seawatch_pii_mode          = "pseudonymise"
  seawatch_pseudo_key        = "an-example-pseudonymisation-key-0123456789"
  allowed_resources          = %s
  webhook_retry_max_attempts = %d
  dcr_enabled                = true
  cimd_enabled               = false
}
`, realm, lockout, aliases, locales, resources, retries)
}
