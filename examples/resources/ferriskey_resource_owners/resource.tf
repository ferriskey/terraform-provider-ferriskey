resource "ferriskey_resource_owners" "main" {
  realm = "acme"

  owners = [
    {
      uri       = "https://mcp.example.com"
      client_id = ferriskey_client.mcp.client_uuid
    },
  ]
}
