resource "ferriskey_client_web_origin" "app" {
  realm     = "acme"
  client_id = ferriskey_client.app.client_uuid
  value     = "https://app.example.com"
}
