package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestPortalJSONContains(t *testing.T) {
	prior := map[string]any{"colors": map[string]any{"links": "#111"}}
	server := map[string]any{"colors": map[string]any{"links": "#111", "error": "#f00"}, "fonts": map[string]any{}}
	if !portalJSONContains(prior, server, true) {
		t.Fatal("subset should match")
	}
	if portalJSONContains(prior, server, false) {
		t.Fatal("exact should not match")
	}
	if portalJSONContains(map[string]any{"colors": map[string]any{"links": "#222"}}, server, true) {
		t.Fatal("differing value should not match")
	}
}

func TestPortalParseID(t *testing.T) {
	parts, err := parseCompositeID("r/t/login", "realm", "theme_id", "page_type")
	if err != nil || len(parts) != 3 || parts[2] != "login" {
		t.Fatalf("%v %v", parts, err)
	}
	if _, err = parseCompositeID("r/t", "realm", "theme_id", "page_type"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPortalPageKey(t *testing.T) {
	if got := portalPageKey("magic_link_verify"); got != "magicLinkVerify" {
		t.Fatalf("got %s", got)
	}
}

func testAccPortalRealm() string {
	return "tfacc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
}

func testAccPortalLayoutConfig(realm, name, tree string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_portal_layout" "test" {
  realm = ferriskey_realm.test.name
  name  = %q
  tree  = jsonencode(%s)
}
`, realm, name, tree)
}

func TestAccPortalLayoutResource(t *testing.T) {
	realm := testAccPortalRealm()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortalLayoutConfig(realm, "main", `{ type = "root", children = [] }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_portal_layout.test", "name", "main"),
					resource.TestCheckResourceAttr("ferriskey_portal_layout.test", "is_default", "true"),
					resource.TestMatchResourceAttr("ferriskey_portal_layout.test", "id", regexp.MustCompile("^"+regexp.QuoteMeta(realm)+"/[0-9a-f-]+$")),
					resource.TestCheckResourceAttrSet("ferriskey_portal_layout.test", "layout_id"),
				),
			},
			{
				ResourceName:      "ferriskey_portal_layout.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_portal_layout.test"),
				ImportStateVerify: true,
			},
			{
				Config: testAccPortalLayoutConfig(realm, "renamed", `{ type = "root", children = [{ type = "slot" }] }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_portal_layout.test", "name", "renamed"),
					resource.TestCheckResourceAttr("ferriskey_portal_layout.test", "tree", `{"children":[{"type":"slot"}],"type":"root"}`),
				),
			},
		},
	})
}

func testAccPortalDefaultLayoutConfig(realm, target string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_portal_layout" "a" {
  realm = ferriskey_realm.test.name
  name  = "a"
  tree  = jsonencode({ type = "root" })
}

resource "ferriskey_portal_layout" "b" {
  realm      = ferriskey_realm.test.name
  name       = "b"
  tree       = jsonencode({ type = "root" })
  depends_on = [ferriskey_portal_layout.a]
}

resource "ferriskey_portal_default_layout" "test" {
  realm     = ferriskey_realm.test.name
  layout_id = ferriskey_portal_layout.%s.layout_id
}
`, realm, target)
}

func TestAccPortalDefaultLayoutResource(t *testing.T) {
	realm := testAccPortalRealm()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortalDefaultLayoutConfig(realm, "b"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("ferriskey_portal_default_layout.test", "layout_id", "ferriskey_portal_layout.b", "layout_id"),
					resource.TestCheckResourceAttr("ferriskey_portal_default_layout.test", "id", realm),
				),
			},
			{
				ResourceName:      "ferriskey_portal_default_layout.test",
				ImportState:       true,
				ImportStateId:     realm,
				ImportStateVerify: true,
			},
			{
				Config: testAccPortalDefaultLayoutConfig(realm, "a"),
				Check:  resource.TestCheckResourceAttrPair("ferriskey_portal_default_layout.test", "layout_id", "ferriskey_portal_layout.a", "layout_id"),
			},
		},
	})
}

func testAccPortalThemeConfig(realm, name, colors string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_portal_layout" "test" {
  realm = ferriskey_realm.test.name
  name  = "frame"
  tree  = jsonencode({ type = "root" })
}

resource "ferriskey_portal_theme" "test" {
  realm     = ferriskey_realm.test.name
  name      = %q
  layout_id = ferriskey_portal_layout.test.layout_id
  config    = jsonencode({ colors = { primaryButton = %q } })
}
`, realm, name, colors)
}

func TestAccPortalThemeResource(t *testing.T) {
	realm := testAccPortalRealm()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortalThemeConfig(realm, "brand", "#111111"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_portal_theme.test", "name", "brand"),
					resource.TestCheckResourceAttrPair("ferriskey_portal_theme.test", "layout_id", "ferriskey_portal_layout.test", "layout_id"),
					resource.TestMatchResourceAttr("ferriskey_portal_theme.test", "id", regexp.MustCompile("^"+regexp.QuoteMeta(realm)+"/[0-9a-f-]+$")),
					resource.TestCheckResourceAttrSet("ferriskey_portal_theme.test", "theme_id"),
				),
			},
			{
				ResourceName:            "ferriskey_portal_theme.test",
				ImportState:             true,
				ImportStateIdFunc:       importStateIDFunc("ferriskey_portal_theme.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config"},
			},
			{
				Config: testAccPortalThemeConfig(realm, "brand2", "#222222"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_portal_theme.test", "name", "brand2"),
					resource.TestCheckResourceAttr("ferriskey_portal_theme.test", "config", `{"colors":{"primaryButton":"#222222"}}`),
				),
			},
		},
	})
}

func testAccPortalThemePageConfig(realm, tree string) string {
	return fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}

resource "ferriskey_portal_theme" "test" {
  realm = ferriskey_realm.test.name
  name  = "draft"
}

resource "ferriskey_portal_theme_page" "test" {
  realm     = ferriskey_realm.test.name
  theme_id  = ferriskey_portal_theme.test.theme_id
  page_type = "forgot_password"
  tree      = jsonencode(%s)
}
`, realm, tree)
}

func TestAccPortalThemePageResource(t *testing.T) {
	realm := testAccPortalRealm()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortalThemePageConfig(realm, `[{ type = "email_input" }]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ferriskey_portal_theme_page.test", "page_type", "forgot_password"),
					resource.TestMatchResourceAttr("ferriskey_portal_theme_page.test", "id", regexp.MustCompile("^"+regexp.QuoteMeta(realm)+"/[0-9a-f-]+/forgot_password$")),
				),
			},
			{
				ResourceName:      "ferriskey_portal_theme_page.test",
				ImportState:       true,
				ImportStateIdFunc: importStateIDFunc("ferriskey_portal_theme_page.test"),
				ImportStateVerify: true,
			},
			{
				Config: testAccPortalThemePageConfig(realm, `[{ type = "email_input" }, { type = "submit_button" }]`),
				Check:  resource.TestCheckResourceAttr("ferriskey_portal_theme_page.test", "tree", `[{"type":"email_input"},{"type":"submit_button"}]`),
			},
		},
	})
}

var testAccPortalRequiredBlocks = []struct {
	page   string
	blocks []string
}{
	{"login", []string{"email_input", "password_input", "submit_button", "identity_providers"}},
	{"register", []string{"email_input", "password_input", "submit_button"}},
	{"totp", []string{"totp_input", "submit_button"}},
	{"forgot_password", []string{"email_input", "submit_button"}},
	{"reset_password", []string{"password_input", "submit_button"}},
	{"magic_link_verify", []string{"submit_button"}},
	{"magic_link_request", []string{"email_input", "submit_button"}},
	{"verify_email", []string{"submit_button"}},
	{"totp_setup", []string{"totp_qr_code", "totp_input", "submit_button"}},
	{"device_verify", []string{"user_code_input", "device_approve_button", "device_deny_button"}},
	{"consent", []string{"consent_scopes", "consent_allow_button", "consent_deny_button"}},
}

func testAccPortalActiveThemeConfig(realm, target string) string {
	cfg := fmt.Sprintf(`
resource "ferriskey_realm" "test" {
  name = %q
}
`, realm)
	var deps []string
	for _, theme := range []string{"a", "b"} {
		cfg += fmt.Sprintf(`
resource "ferriskey_portal_theme" %q {
  realm = ferriskey_realm.test.name
  name  = %q
}
`, theme, theme)
		for _, p := range testAccPortalRequiredBlocks {
			tree := "["
			for i, b := range p.blocks {
				if i > 0 {
					tree += ", "
				}
				tree += fmt.Sprintf("{ type = %q }", b)
			}
			tree += "]"
			name := theme + "_" + p.page
			deps = append(deps, "ferriskey_portal_theme_page."+name)
			cfg += fmt.Sprintf(`
resource "ferriskey_portal_theme_page" %q {
  realm     = ferriskey_realm.test.name
  theme_id  = ferriskey_portal_theme.%s.theme_id
  page_type = %q
  tree      = jsonencode(%s)
}
`, name, theme, p.page, tree)
		}
	}
	cfg += fmt.Sprintf(`
resource "ferriskey_portal_active_theme" "test" {
  realm      = ferriskey_realm.test.name
  theme_id   = ferriskey_portal_theme.%s.theme_id
  depends_on = [%s]
}
`, target, strings.Join(deps, ", "))
	return cfg
}

func TestAccPortalActiveThemeResource(t *testing.T) {
	realm := testAccPortalRealm()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortalActiveThemeConfig(realm, "a"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("ferriskey_portal_active_theme.test", "theme_id", "ferriskey_portal_theme.a", "theme_id"),
					resource.TestCheckResourceAttr("ferriskey_portal_active_theme.test", "id", realm),
				),
			},
			{
				ResourceName:      "ferriskey_portal_active_theme.test",
				ImportState:       true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) { return realm, nil },
				ImportStateVerify: true,
			},
			{
				Config: testAccPortalActiveThemeConfig(realm, "b"),
				Check:  resource.TestCheckResourceAttrPair("ferriskey_portal_active_theme.test", "theme_id", "ferriskey_portal_theme.b", "theme_id"),
			},
		},
	})
}
