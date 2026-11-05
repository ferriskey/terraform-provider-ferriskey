resource "ferriskey_realm_maintenance_whitelist_entry" "admin" {
  realm   = "acme"
  user_id = ferriskey_user.admin.user_uuid
}
