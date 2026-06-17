resource "ferriskey_organization" "tenant" {
  realm       = "acme"
  name        = "Globex"
  alias       = "globex"
  description = "Globex Corporation tenant"
  domain      = "globex.example"
  enabled     = true
}
