resource "ferriskey_client_scope" "billing" {
  realm       = ferriskey_realm.acme.name
  name        = "billing"
  description = "Access to billing data"
  is_default  = false
}
