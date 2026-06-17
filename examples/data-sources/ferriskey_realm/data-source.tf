data "ferriskey_realm" "master" {
  name = "master"
}

output "access_token_lifetime" {
  value = data.ferriskey_realm.master.settings.access_token_lifetime
}
