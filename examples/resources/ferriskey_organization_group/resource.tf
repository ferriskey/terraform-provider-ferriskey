resource "ferriskey_organization_group" "engineering" {
  realm           = "acme"
  organization_id = ferriskey_organization.tenant.organization_uuid
  name            = "engineering"
  description     = "Engineering department"
}

resource "ferriskey_organization_group" "platform" {
  realm           = "acme"
  organization_id = ferriskey_organization.tenant.organization_uuid
  name            = "platform"
  parent_group_id = ferriskey_organization_group.engineering.group_uuid
}
