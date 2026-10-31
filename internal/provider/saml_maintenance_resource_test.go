package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func samlmAccNames() (string, string) {
	return "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum),
		"saml-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
}

func samlmAccBase(realm, clientID string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_client" "test" {
  realm       = ferriskey_realm.test.name
  client_id   = %q
  name        = "SAML test"
  client_type = "confidential"
  protocol    = "saml"
}
`, realm, clientID)
}

func samlmAccPrincipals() string {
	return `
resource "ferriskey_user" "test" {
  realm    = ferriskey_realm.test.name
  username = "wl-user"
}

resource "ferriskey_role" "test" {
  realm       = ferriskey_realm.test.name
  name        = "wl-role"
  permissions = []
}
`
}

func TestAccClientSamlConfigResource(t *testing.T) {
	realm, clientID := samlmAccNames()
	cfg := func(acs string, signDocs bool) string {
		return samlmAccBase(realm, clientID) + fmt.Sprintf(`
resource "ferriskey_client_saml_config" "test" {
  realm          = ferriskey_realm.test.name
  client_uuid    = ferriskey_client.test.client_uuid
  sp_entity_id   = "urn:example:sp"
  acs_url        = %q
  sign_documents = %t
}
`, acs, signDocs)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("https://sp.example.com/acs", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_saml_config.test", "sp_entity_id", "urn:example:sp"),
					resource.TestCheckResourceAttr("ferriskey_client_saml_config.test", "acs_url", "https://sp.example.com/acs"),
					resource.TestCheckResourceAttr("ferriskey_client_saml_config.test", "name_id_format", samlmNameIDEmail),
					resource.TestCheckResourceAttr("ferriskey_client_saml_config.test", "sign_assertions", "true"),
					resource.TestCheckResourceAttr("ferriskey_client_saml_config.test", "sign_documents", "false"),
				),
			},
			{
				ResourceName:      "ferriskey_client_saml_config.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_saml_config.test"),
				ImportStateVerify: true,
			},
			{
				Config: cfg("https://sp.example.com/acs2", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_saml_config.test", "acs_url", "https://sp.example.com/acs2"),
					resource.TestCheckResourceAttr("ferriskey_client_saml_config.test", "sign_documents", "true"),
				),
			},
		},
	})
}

func TestAccClientSamlAttributeMapperResource(t *testing.T) {
	realm, clientID := samlmAccNames()
	cfg := func(name, source string) string {
		return samlmAccBase(realm, clientID) + fmt.Sprintf(`
resource "ferriskey_client_saml_attribute_mapper" "test" {
  realm       = ferriskey_realm.test.name
  client_uuid = ferriskey_client.test.client_uuid
  name        = %q
  source      = %q
}
`, name, source)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("mail", "user:email"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_saml_attribute_mapper.test", "name", "mail"),
					resource.TestCheckResourceAttr("ferriskey_client_saml_attribute_mapper.test", "source", "user:email"),
					resource.TestCheckResourceAttr("ferriskey_client_saml_attribute_mapper.test", "name_format", samlmNameFormatBasic),
					resource.TestCheckResourceAttrSet("ferriskey_client_saml_attribute_mapper.test", "mapper_id"),
				),
			},
			{
				ResourceName:      "ferriskey_client_saml_attribute_mapper.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_saml_attribute_mapper.test"),
				ImportStateVerify: true,
			},
			{
				Config: cfg("department", "attribute:dept"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_saml_attribute_mapper.test", "name", "department"),
					resource.TestCheckResourceAttr("ferriskey_client_saml_attribute_mapper.test", "source", "attribute:dept"),
				),
			},
		},
	})
}

func TestAccClientMaintenanceResource(t *testing.T) {
	realm, clientID := samlmAccNames()
	cfg := func(enabled bool, reason, strategy string) string {
		return samlmAccBase(realm, clientID) + fmt.Sprintf(`
resource "ferriskey_client_maintenance" "test" {
  realm            = ferriskey_realm.test.name
  client_uuid      = ferriskey_client.test.client_uuid
  enabled          = %t
  reason           = %q
  session_strategy = %q
}
`, enabled, reason, strategy)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(true, "upgrade", "terminate"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_maintenance.test", "enabled", "true"),
					resource.TestCheckResourceAttr("ferriskey_client_maintenance.test", "reason", "upgrade"),
					resource.TestCheckResourceAttr("ferriskey_client_maintenance.test", "session_strategy", "terminate"),
				),
			},
			{
				ResourceName:      "ferriskey_client_maintenance.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_maintenance.test"),
				ImportStateVerify: true,
			},
			{
				Config: cfg(true, "database migration", "expire"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_client_maintenance.test", "reason", "database migration"),
					resource.TestCheckResourceAttr("ferriskey_client_maintenance.test", "session_strategy", "expire"),
				),
			},
		},
	})
}

func TestAccClientMaintenanceWhitelistEntryResource(t *testing.T) {
	realm, clientID := samlmAccNames()
	cfg := func(attr string) string {
		return samlmAccBase(realm, clientID) + samlmAccPrincipals() + fmt.Sprintf(`
resource "ferriskey_client_maintenance_whitelist_entry" "test" {
  realm       = ferriskey_realm.test.name
  client_uuid = ferriskey_client.test.client_uuid
  %s
}
`, attr)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("user_id = ferriskey_user.test.user_uuid"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ferriskey_client_maintenance_whitelist_entry.test", "entry_id"),
					resource.TestCheckResourceAttrPair("ferriskey_client_maintenance_whitelist_entry.test", "user_id", "ferriskey_user.test", "user_uuid"),
					resource.TestCheckNoResourceAttr("ferriskey_client_maintenance_whitelist_entry.test", "role_id"),
				),
			},
			{
				ResourceName:      "ferriskey_client_maintenance_whitelist_entry.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_client_maintenance_whitelist_entry.test"),
				ImportStateVerify: true,
			},
			{
				Config: cfg("role_id = ferriskey_role.test.role_uuid"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("ferriskey_client_maintenance_whitelist_entry.test", "role_id", "ferriskey_role.test", "role_uuid"),
					resource.TestCheckNoResourceAttr("ferriskey_client_maintenance_whitelist_entry.test", "user_id"),
				),
			},
		},
	})
}

func TestAccRealmMaintenanceWhitelistEntryResource(t *testing.T) {
	realm, clientID := samlmAccNames()
	cfg := func(attr string) string {
		return samlmAccBase(realm, clientID) + samlmAccPrincipals() + fmt.Sprintf(`
resource "ferriskey_realm_maintenance_whitelist_entry" "test" {
  realm = ferriskey_realm.test.name
  %s
}
`, attr)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("user_id = ferriskey_user.test.user_uuid"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("ferriskey_realm_maintenance_whitelist_entry.test", "entry_id"),
					resource.TestCheckResourceAttrPair("ferriskey_realm_maintenance_whitelist_entry.test", "user_id", "ferriskey_user.test", "user_uuid"),
				),
			},
			{
				ResourceName:      "ferriskey_realm_maintenance_whitelist_entry.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_realm_maintenance_whitelist_entry.test"),
				ImportStateVerify: true,
			},
			{
				Config: cfg("role_id = ferriskey_role.test.role_uuid"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("ferriskey_realm_maintenance_whitelist_entry.test", "role_id", "ferriskey_role.test", "role_uuid"),
					resource.TestCheckNoResourceAttr("ferriskey_realm_maintenance_whitelist_entry.test", "user_id"),
				),
			},
		},
	})
}
