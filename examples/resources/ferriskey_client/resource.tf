resource "ferriskey_realm" "example" {
  name = "acme"
}

resource "ferriskey_client" "web" {
  realm       = ferriskey_realm.example.name
  client_id   = "acme-web"
  name        = "Acme Web App"
  client_type = "confidential"

  # Order is not significant; managed as a set.
  redirect_uris = [
    "https://app.acme.example/callback",
    "https://app.acme.example/silent-renew",
  ]

  # Optional token lifetime overrides (seconds).
  access_token_lifetime = 300
}

# A public SPA client.
resource "ferriskey_client" "spa" {
  realm                        = ferriskey_realm.example.name
  client_id                    = "acme-spa"
  name                         = "Acme SPA"
  client_type                  = "public"
  public_client                = true
  direct_access_grants_enabled = false
  redirect_uris                = ["https://spa.acme.example/callback"]
}
