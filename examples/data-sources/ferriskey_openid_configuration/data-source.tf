data "ferriskey_openid_configuration" "master" {
  realm = "master"
}

output "token_endpoint" {
  value = data.ferriskey_openid_configuration.master.token_endpoint
}
