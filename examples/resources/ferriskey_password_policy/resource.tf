resource "ferriskey_password_policy" "example" {
  realm             = "acme"
  min_length        = 12
  require_uppercase = true
  require_number    = true
  require_special   = true
}
