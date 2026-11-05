resource "ferriskey_federation_provider" "ldap" {
  realm         = "acme"
  name          = "corporate-ldap"
  provider_type = "Ldap"
  sync_mode     = "Import"
  bind_password = var.ldap_bind_password

  config = jsonencode({
    connection = {
      server_url                 = "ldap://ldap.example.com"
      port                       = 389
      use_tls                    = false
      use_starttls               = false
      connection_timeout_seconds = 10
    }
    bind       = { bind_dn = "cn=admin,dc=example,dc=com" }
    search     = { base_dn = "dc=example,dc=com", user_search_filter = "(uid={0})" }
    attributes = { username = "uid", email = "mail", first_name = "givenName", last_name = "sn" }
  })
}
