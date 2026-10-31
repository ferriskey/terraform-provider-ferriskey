package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccClientResource(t *testing.T) {
	realm := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	clientID := "app-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // Create a confidential client with two redirect URIs
				Config: testAccClientConfig(realm, clientID, "Initial", []string{
					"https://app.example.com/callback",
					"https://app.example.com/silent",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client.test", "client_id", clientID),
					resource.TestCheckResourceAttr("ferriskey_client.test", "name", "Initial"),
					resource.TestCheckResourceAttr("ferriskey_client.test", "client_type", "confidential"),
					resource.TestMatchResourceAttr("ferriskey_client.test", "id", regexp.MustCompile("^"+regexp.QuoteMeta(realm)+"/[0-9a-f-]+$")),
					resource.TestCheckResourceAttrSet("ferriskey_client.test", "client_uuid"),
					resource.TestCheckResourceAttr("ferriskey_client.test", "redirect_uris.#", "2"),
					resource.TestCheckResourceAttrSet("ferriskey_client.test", "secret"),
				),
			},
			{ // Import (composite ID {realm}/{uuid}); secret is not returned on import GET
				ResourceName:            "ferriskey_client.test",
				ImportState:             true,
				ImportStateIdFunc:       importStateIDFunc("ferriskey_client.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
			{ // Update name + change redirect URIs (drop one, add one)
				Config: testAccClientConfig(realm, clientID, "Updated", []string{
					"https://app.example.com/callback",
					"https://app.example.com/new",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client.test", "name", "Updated"),
					resource.TestCheckResourceAttr("ferriskey_client.test", "redirect_uris.#", "2"),
					resource.TestCheckTypeSetElemAttr("ferriskey_client.test", "redirect_uris.*", "https://app.example.com/new"),
				),
			},
		},
	})
}

func testAccClientConfig(realm, clientID, name string, redirects []string) string {
	uris := "["
	for i, u := range redirects {
		if i > 0 {
			uris += ", "
		}
		uris += fmt.Sprintf("%q", u)
	}
	uris += "]"

	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_client" "test" {
  realm         = ferriskey_realm.test.name
  client_id     = %q
  name          = %q
  client_type   = "confidential"
  redirect_uris = %s
}
`, realm, clientID, name, uris)
}

// importStateIDFunc returns the composite import ID for a client by reading the
// resource's "id" attribute from the Terraform state.
func importStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		return rs.Primary.Attributes["id"], nil
	}
}

func TestAccClientResourceFlowsAndBackchannel(t *testing.T) {
	realm := "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	clientID := "app-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientFlowsConfig(realm, clientID, true, `"https://app.example.com/backchannel"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client.test", "require_pkce", "true"),
					resource.TestCheckResourceAttr("ferriskey_client.test", "consent_required", "true"),
					resource.TestCheckResourceAttr("ferriskey_client.test", "oauth_device_code_grant_enabled", "true"),
					resource.TestCheckResourceAttr("ferriskey_client.test", "token_exchange_enabled", "true"),
					resource.TestCheckResourceAttr("ferriskey_client.test", "backchannel_logout_uri", "https://app.example.com/backchannel"),
					resource.TestCheckResourceAttr("ferriskey_client.test", "backchannel_logout_session_required", "false"),
					resource.TestCheckResourceAttr("ferriskey_client.test", "registration_source", "admin"),
				),
			},
			{
				ResourceName:            "ferriskey_client.test",
				ImportState:             true,
				ImportStateIdFunc:       importStateIDFunc("ferriskey_client.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
			{ // Flip the flags and clear the back-channel URI
				Config: testAccClientFlowsConfig(realm, clientID, false, "null"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client.test", "require_pkce", "false"),
					resource.TestCheckResourceAttr("ferriskey_client.test", "consent_required", "false"),
					resource.TestCheckNoResourceAttr("ferriskey_client.test", "backchannel_logout_uri"),
				),
			},
		},
	})
}

func testAccClientFlowsConfig(realm, clientID string, on bool, backchannel string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_client" "test" {
  realm       = ferriskey_realm.test.name
  client_id   = %q
  client_type = "confidential"

  require_pkce                        = %[3]t
  consent_required                    = %[3]t
  oauth_device_code_grant_enabled     = %[3]t
  token_exchange_enabled              = %[3]t
  backchannel_logout_uri              = %[4]s
  backchannel_logout_session_required = false
}
`, realm, clientID, on, backchannel)
}
