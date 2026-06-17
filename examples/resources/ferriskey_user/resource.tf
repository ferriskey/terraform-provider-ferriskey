resource "ferriskey_user" "alice" {
  realm          = "acme"
  username       = "alice"
  email          = "alice@acme.example"
  email_verified = true
  firstname      = "Alice"
  lastname       = "Liddell"
  enabled        = true

  required_actions = ["verify_email"]
}
