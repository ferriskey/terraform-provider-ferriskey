resource "ferriskey_client_saml_attribute_mapper" "mail" {
  realm       = "acme"
  client_uuid = ferriskey_client.saml.client_uuid
  name        = "mail"
  source      = "user:email"
}
