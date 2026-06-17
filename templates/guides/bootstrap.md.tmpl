---
page_title: "Bootstrapping a fresh FerrisKey instance"
subcategory: "Guides"
description: |-
  Two-phase pattern for bootstrapping authentication on a freshly deployed
  FerrisKey instance, from the Helm admin account to a dedicated Terraform
  service account.
---

# Bootstrapping a fresh FerrisKey instance

Authenticating against a freshly deployed FerrisKey instance is a
chicken-and-egg problem: the credentials Terraform should use day-to-day (a
confidential service account) do not exist yet on a brand-new instance. The
provider therefore supports two OAuth2 grant types, and the recommended
workflow uses both in sequence.

```text
helm install  (admin bootstrap stored in a Kubernetes Secret)
   └─> Terraform phase 1  [auth: password grant, initial admin account]
          └─> creates the service account "terraform-runner"
              (confidential client + secret + scoped admin roles)
                 └─> Terraform phase 2  [auth: client credentials, terraform-runner]
                        └─> creates realms, clients, users, roles, ...
```

Phase 1 and phase 2 **must use separate state files / backends**. If they shared
state, a `terraform destroy` of your business configuration could delete the
runner that performs the destroy.

## Phase 1 — password grant (bootstrap)

Use the initial admin account created by the Helm chart, authenticating against
the public `admin-cli` client. The only thing this phase creates is the
service account used by phase 2.

```terraform
provider "ferriskey" {
  url       = "https://auth.example.com"
  realm     = "master"
  client_id = "admin-cli"
  username  = "admin"
  password  = var.bootstrap_admin_password # FERRISKEY_PASSWORD in CI
}

# A confidential client that phase 2 will authenticate as.
resource "ferriskey_client" "terraform_runner" {
  realm                   = "master"
  client_id               = "terraform-runner"
  name                    = "Terraform Runner"
  client_type             = "confidential"
  service_account_enabled = true
}

output "terraform_runner_client_id" {
  value = ferriskey_client.terraform_runner.client_id
}

output "terraform_runner_secret" {
  value     = ferriskey_client.terraform_runner.secret
  sensitive = true
}
```

Grant the service account the admin roles it needs (scoped to least privilege)
using `ferriskey_role` and the role-assignment resources, then capture the
generated secret for phase 2 — ideally into a secrets manager rather than a
plaintext variable.

## Phase 2 — client credentials (steady state)

Configure the provider with the service account from phase 1. This is the
state/backend you use for all ongoing business configuration.

```terraform
provider "ferriskey" {
  url           = "https://auth.example.com"
  realm         = "master"
  client_id     = "terraform-runner"
  client_secret = var.terraform_runner_secret # FERRISKEY_CLIENT_SECRET in CI
}

resource "ferriskey_realm" "acme" {
  name = "acme"
}

resource "ferriskey_client" "web" {
  realm         = ferriskey_realm.acme.name
  client_id     = "acme-web"
  client_type   = "confidential"
  redirect_uris = ["https://app.acme.example/callback"]
}
```

## Environment variables

Every provider argument has an environment-variable equivalent, so no secret
needs to live in a `.tf` file — essential for CI:

| Argument        | Environment variable      |
|-----------------|---------------------------|
| `url`           | `FERRISKEY_URL`           |
| `realm`         | `FERRISKEY_REALM`         |
| `username`      | `FERRISKEY_USERNAME`      |
| `password`      | `FERRISKEY_PASSWORD`      |
| `client_id`     | `FERRISKEY_CLIENT_ID`     |
| `client_secret` | `FERRISKEY_CLIENT_SECRET` |
| `scope`         | `FERRISKEY_SCOPE`         |

## Token lifetime and long applies

The provider fetches an access token lazily, caches it, and refreshes it
automatically before it expires (using the refresh token when available, and
falling back to a fresh grant otherwise). A long `terraform apply` that outlives
the access token lifetime is handled transparently — you do not need to tune
token lifetimes for Terraform's sake.
