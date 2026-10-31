resource "ferriskey_client_maintenance_whitelist_entry" "ops" {
  realm       = "acme"
  client_uuid = ferriskey_client.app.client_uuid
  role_id     = ferriskey_role.ops.role_uuid
}
