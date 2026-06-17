resource "ferriskey_organization_member" "example" {
  realm           = "acme"
  organization_id = ferriskey_organization.tenant.organization_uuid
  user_id         = ferriskey_user.alice.user_uuid
}
