resource "ferriskey_realm" "example" {
  name = "acme"
}

resource "ferriskey_realm_settings" "example" {
  realm = ferriskey_realm.example.name

  access_token_lifetime     = 600
  refresh_token_lifetime    = 43200
  user_registration_enabled = true
  passkey_enabled           = true
  magic_link_enabled        = false

  # Users may sign in with their email first, then their username.
  login_aliases     = ["email", "username"]
  default_locale    = "en"
  supported_locales = ["en", "fr"]

  require_mfa              = true
  lockout_threshold        = 5
  lockout_duration_seconds = 900
}
