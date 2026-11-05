package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func rxRealmName() string {
	return "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
}

func TestAccRxWebhookResource(t *testing.T) {
	realm := rxRealmName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRxWebhookConfig(realm, "https://example.com/one", `["user.created"]`, "v1", `max_attempts = 3`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_webhook.test", "endpoint", "https://example.com/one"),
					resource.TestCheckResourceAttr("ferriskey_webhook.test", "subscribers.#", "1"),
					resource.TestCheckResourceAttr("ferriskey_webhook.test", "retry_policy.max_attempts", "3"),
					resource.TestMatchResourceAttr("ferriskey_webhook.test", "id", regexp.MustCompile("^"+regexp.QuoteMeta(realm)+"/[0-9a-f-]+$")),
					resource.TestCheckResourceAttrSet("ferriskey_webhook.test", "secret"),
				),
			},
			{
				ResourceName:            "ferriskey_webhook.test",
				ImportState:             true,
				ImportStateIdFunc:       importStateIDFunc("ferriskey_webhook.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"headers", "secret", "rotate_secret_trigger"},
			},
			{
				Config: testAccRxWebhookConfig(realm, "https://example.com/two", `["user.created", "user.updated"]`, "v1", `max_attempts = 4
    base_delay_ms = 100`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_webhook.test", "endpoint", "https://example.com/two"),
					resource.TestCheckResourceAttr("ferriskey_webhook.test", "subscribers.#", "2"),
					resource.TestCheckResourceAttr("ferriskey_webhook.test", "retry_policy.base_delay_ms", "100"),
				),
			},
			{
				Config: testAccRxWebhookConfig(realm, "https://example.com/two", `["user.created", "user.updated"]`, "v2", `max_attempts = 4
    base_delay_ms = 100`),
				Check: resource.TestCheckResourceAttrSet("ferriskey_webhook.test", "secret"),
			},
		},
	})
}

func testAccRxWebhookConfig(realm, endpoint, subscribers, trigger, retry string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_webhook" "test" {
  realm                 = ferriskey_realm.test.name
  name                  = "acc"
  endpoint              = %q
  subscribers           = %s
  headers               = { "X-Token" = "abc" }
  rotate_secret_trigger = %q

  retry_policy = {
    %s
  }
}
`, realm, endpoint, subscribers, trigger, retry)
}

func TestAccRxEmailTemplateResource(t *testing.T) {
	realm := rxRealmName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRxEmailTemplateConfig(realm, "acc", "hello"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_email_template.test", "name", "acc"),
					resource.TestCheckResourceAttr("ferriskey_email_template.test", "email_type", "reset_password"),
					resource.TestCheckResourceAttrSet("ferriskey_email_template.test", "mjml"),
					resource.TestCheckResourceAttrSet("ferriskey_email_template.test", "template_id"),
					resource.TestCheckResourceAttr("data.ferriskey_email_template_variables.test", "variables.0.name", "user.first_name"),
				),
			},
			{
				ResourceName:      "ferriskey_email_template.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_email_template.test"),
				ImportStateVerify: true,
			},
			{
				Config: testAccRxEmailTemplateConfig(realm, "acc2", "goodbye"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_email_template.test", "name", "acc2"),
					resource.TestMatchResourceAttr("ferriskey_email_template.test", "mjml", regexp.MustCompile("goodbye")),
				),
			},
		},
	})
}

func testAccRxEmailTemplateConfig(realm, name, text string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

data "ferriskey_email_template_variables" "test" {
  email_type = "reset_password"
}

resource "ferriskey_email_template" "test" {
  realm      = ferriskey_realm.test.name
  name       = %q
  email_type = "reset_password"
  structure = jsonencode({
    children = [{
      type     = "mj-section"
      children = [{ type = "mj-text", content = %q }]
    }]
  })
}
`, realm, name, text)
}

func TestAccRxFederationProviderResource(t *testing.T) {
	realm := rxRealmName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRxFederationConfig(realm, "ldap-acc", "ldap://ldap.example.com", "pw1", 1, "Import"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_federation_provider.test", "name", "ldap-acc"),
					resource.TestCheckResourceAttr("ferriskey_federation_provider.test", "provider_type", "Ldap"),
					resource.TestCheckResourceAttr("ferriskey_federation_provider.test", "sync_mode", "Import"),
					resource.TestCheckResourceAttr("ferriskey_federation_provider.test", "priority", "1"),
				),
			},
			{
				ResourceName:            "ferriskey_federation_provider.test",
				ImportState:             true,
				ImportStateIdFunc:       importStateIDFunc("ferriskey_federation_provider.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"bind_password"},
			},
			{
				Config: testAccRxFederationConfig(realm, "ldap-acc2", "ldap://ldap2.example.com", "pw2", 5, "Force"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_federation_provider.test", "name", "ldap-acc2"),
					resource.TestCheckResourceAttr("ferriskey_federation_provider.test", "sync_mode", "Force"),
					resource.TestCheckResourceAttr("ferriskey_federation_provider.test", "priority", "5"),
				),
			},
		},
	})
}

func testAccRxFederationConfig(realm, name, serverURL, password string, priority int, syncMode string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_federation_provider" "test" {
  realm         = ferriskey_realm.test.name
  name          = %q
  provider_type = "Ldap"
  priority      = %d
  sync_mode     = %q
  bind_password = %q

  config = jsonencode({
    connection = {
      server_url                 = %q
      port                       = 389
      use_tls                    = false
      use_starttls               = false
      connection_timeout_seconds = 10
    }
    bind       = { bind_dn = "cn=admin,dc=example,dc=com" }
    search     = { base_dn = "dc=example,dc=com", user_search_filter = "(uid={0})" }
    attributes = { username = "uid", email = "mail", first_name = "givenName", last_name = "sn" }
  })
}
`, realm, name, priority, syncMode, password, serverURL)
}

func TestAccRxUserAttributeResource(t *testing.T) {
	realm := rxRealmName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRxUserAttributeConfig(realm, "one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_user_attribute.test", "key", "department"),
					resource.TestCheckResourceAttr("ferriskey_user_attribute.test", "value", "one"),
					resource.TestCheckResourceAttr("ferriskey_user_attribute.other", "value", "keep"),
				),
			},
			{
				ResourceName:      "ferriskey_user_attribute.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_user_attribute.test"),
				ImportStateVerify: true,
			},
			{
				Config: testAccRxUserAttributeConfig(realm, "two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_user_attribute.test", "value", "two"),
					resource.TestCheckResourceAttr("ferriskey_user_attribute.other", "value", "keep"),
				),
			},
		},
	})
}

func testAccRxUserAttributeConfig(realm, value string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_user" "test" {
  realm    = ferriskey_realm.test.name
  username = "attruser"
  email    = "attr@example.com"
}

resource "ferriskey_user_attribute" "test" {
  realm   = ferriskey_realm.test.name
  user_id = ferriskey_user.test.user_uuid
  key     = "department"
  value   = %q
}

resource "ferriskey_user_attribute" "other" {
  realm   = ferriskey_realm.test.name
  user_id = ferriskey_user.test.user_uuid
  key     = "other"
  value   = "keep"
}
`, realm, value)
}
