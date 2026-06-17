resource "ferriskey_identity_provider" "google" {
  realm        = "acme"
  alias        = "google"
  provider_id  = "oidc"
  display_name = "Google"
  enabled      = true

  # Only the keys you set are managed; keys the server adds are ignored.
  config = {
    clientId     = var.google_client_id
    clientSecret = var.google_client_secret
  }
}
