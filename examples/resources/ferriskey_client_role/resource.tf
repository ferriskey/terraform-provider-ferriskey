resource "ferriskey_client_role" "reader" {
  realm       = "acme"
  client_id   = ferriskey_client.app.client_uuid
  name        = "reader"
  description = "Read-only access"

  permissions = ["view_users"]
}
