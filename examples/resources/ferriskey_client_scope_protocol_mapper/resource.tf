resource "ferriskey_client_scope_protocol_mapper" "email_claim" {
  realm       = ferriskey_realm.acme.name
  scope_id    = ferriskey_client_scope.billing.scope_id
  name        = "email-claim"
  mapper_type = "oidc-usermodel-attribute-mapper"

  config = {
    "claim.name"         = "email"
    "user.attribute"     = "email"
    "id.token.claim"     = "true"
    "access.token.claim" = "true"
  }
}
