resource "ferriskey_organization_attribute" "tier" {
  realm           = "acme"
  organization_id = ferriskey_organization.tenant.organization_uuid
  key             = "tier"
  value           = "gold"
}
