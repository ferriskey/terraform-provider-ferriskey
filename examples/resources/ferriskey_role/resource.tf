resource "ferriskey_role" "editor" {
  realm       = "acme"
  name        = "editor"
  description = "Can edit content"

  permissions = [
    "content:read",
    "content:write",
  ]
}
