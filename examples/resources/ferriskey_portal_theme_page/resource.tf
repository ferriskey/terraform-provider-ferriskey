resource "ferriskey_portal_theme_page" "login" {
  realm     = "acme"
  theme_id  = ferriskey_portal_theme.brand.theme_id
  page_type = "login"

  tree = jsonencode([
    { type = "email_input" },
    { type = "password_input" },
    { type = "submit_button" },
    { type = "identity_providers" },
  ])
}
