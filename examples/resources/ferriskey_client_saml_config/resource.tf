resource "ferriskey_client_saml_config" "example" {
  realm          = "acme"
  client_uuid    = ferriskey_client.saml.client_uuid
  sp_entity_id   = "urn:example:sp"
  acs_url        = "https://sp.example.com/saml/acs"
  name_id_format = "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
  sign_documents = true
}
