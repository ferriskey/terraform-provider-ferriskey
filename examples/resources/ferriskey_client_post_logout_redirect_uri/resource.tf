resource "ferriskey_client_post_logout_redirect_uri" "app" {
  realm     = "acme"
  client_id = ferriskey_client.app.client_uuid
  value     = "https://app.example.com/logged-out"
  enabled   = true
}
