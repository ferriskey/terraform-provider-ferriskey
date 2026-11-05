resource "ferriskey_organization_group_role" "example" {
  realm           = "acme"
  organization_id = ferriskey_organization.tenant.organization_uuid
  group_id        = ferriskey_organization_group.engineering.group_uuid
  role_id         = ferriskey_role.editor.role_uuid
}
