resource "ferriskey_portal_active_theme" "this" {
  realm    = "acme"
  theme_id = ferriskey_portal_theme.brand.theme_id

  depends_on = [ferriskey_portal_theme_page.login]
}
