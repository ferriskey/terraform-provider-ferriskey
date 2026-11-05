resource "ferriskey_organization_member_role" "example" {
  realm           = "acme"
  organization_id = ferriskey_organization.tenant.organization_uuid
  user_id         = ferriskey_organization_member.alice.user_id
  role_id         = ferriskey_role.editor.role_uuid
}
