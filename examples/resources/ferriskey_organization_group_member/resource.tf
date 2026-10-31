resource "ferriskey_organization_group_member" "example" {
  realm           = "acme"
  organization_id = ferriskey_organization.tenant.organization_uuid
  group_id        = ferriskey_organization_group.engineering.group_uuid
  user_id         = ferriskey_user.alice.user_uuid
}
