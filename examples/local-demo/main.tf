terraform {
  required_providers {
    ferriskey = {
      source = "ferriskey/ferriskey"
    }
  }
}

# All credentials come from FERRISKEY_* environment variables, so nothing
# secret lives in this file. Export them in your shell before running:
#
#   export FERRISKEY_URL=http://localhost:3333
#   export FERRISKEY_REALM=master
#   export FERRISKEY_CLIENT_ID=admin-cli
#   export FERRISKEY_USERNAME=admin
#   export FERRISKEY_PASSWORD=...        # the Helm bootstrap admin password
#
provider "ferriskey" {

}

resource "ferriskey_realm" "demo" {
  name = "tf-demo"
}

resource "ferriskey_client" "web" {
  realm       = ferriskey_realm.demo.name
  client_id   = "demo-web"
  name        = "Demo Web App"
  client_type = "confidential"

  redirect_uris = [
    "https://demo.example/callback",
  ]
}

output "client_uuid" {
  value = ferriskey_client.web.client_uuid
}

output "client_secret" {
  value     = ferriskey_client.web.secret
  sensitive = true
}
