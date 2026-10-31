resource "ferriskey_client_maintenance" "example" {
  realm            = "acme"
  client_uuid      = ferriskey_client.app.client_uuid
  enabled          = true
  reason           = "Planned database migration"
  session_strategy = "expire"
}
