resource "ferriskey_client_scope_assignment" "billing" {
  realm     = ferriskey_realm.acme.name
  client_id = ferriskey_client.web.client_uuid
  scope_id  = ferriskey_client_scope.billing.scope_id
  type      = "optional"
}
