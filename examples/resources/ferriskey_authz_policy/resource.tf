resource "ferriskey_authz_policy" "allow_read" {
  realm  = ferriskey_authz_schema.main.realm
  name   = "allow-read"
  source = "permit(principal, action == App::Action::\"read\", resource);"
}

resource "ferriskey_authz_policy" "editors_read" {
  realm       = ferriskey_authz_schema.main.realm
  name        = "editors-read"
  template_id = ferriskey_authz_template.role_can_read.template_id

  links = {
    "?principal" = {
      type = "FerrisKey::Role"
      id   = "editor"
    }
  }
}
