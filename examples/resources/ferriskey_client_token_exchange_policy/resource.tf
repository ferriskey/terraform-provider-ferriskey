resource "ferriskey_client_token_exchange_policy" "app_to_api" {
  realm               = "acme"
  client_id           = ferriskey_client.app.client_uuid
  target_audience     = ferriskey_client.api.client_id
  allowed_scopes      = ["openid", "profile"]
  allow_delegation    = true
  allow_impersonation = false
}
