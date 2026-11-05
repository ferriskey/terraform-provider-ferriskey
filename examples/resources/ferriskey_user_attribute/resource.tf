resource "ferriskey_user_attribute" "department" {
  realm   = "acme"
  user_id = ferriskey_user.alice.user_uuid
  key     = "department"
  value   = "engineering"
}
