resource "ferriskey_organization_group_attribute" "cost_center" {
  realm           = "acme"
  organization_id = ferriskey_organization.tenant.organization_uuid
  group_id        = ferriskey_organization_group.engineering.group_uuid
  key             = "cost-center"
  value           = "cc-1234"
}
