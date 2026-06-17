terraform {
  required_providers {
    ferriskey = {
      source = "ferriskey/ferriskey"
    }
  }
}

# Steady-state authentication: a confidential service account
# (client credentials grant). All arguments can also be supplied via
# FERRISKEY_* environment variables, which is the recommended approach in CI
# so no secret is committed to the .tf files.
provider "ferriskey" {
  url           = "https://auth.example.com"
  realm         = "master"
  client_id     = "terraform-runner"
  client_secret = var.ferriskey_client_secret # or FERRISKEY_CLIENT_SECRET
}

# Bootstrap authentication: the initial admin account (password grant) against
# the public admin-cli client. Use this only for the first phase that creates
# the terraform-runner service account.
#
# provider "ferriskey" {
#   url       = "https://auth.example.com"
#   realm     = "master"
#   client_id = "admin-cli"
#   username  = "admin"
#   password  = var.ferriskey_admin_password # or FERRISKEY_PASSWORD
# }
