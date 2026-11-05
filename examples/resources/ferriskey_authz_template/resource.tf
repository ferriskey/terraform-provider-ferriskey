resource "ferriskey_authz_template" "role_can_read" {
  realm       = ferriskey_authz_schema.main.realm
  name        = "role-can-read"
  description = "Lets a role read documents"
  source      = "permit(principal in ?principal, action == App::Action::\"read\", resource);"
}
