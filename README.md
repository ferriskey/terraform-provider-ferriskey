# Terraform Provider for FerrisKey

[![Tests](https://github.com/ferriskey/terraform-provider-ferriskey/actions/workflows/test.yml/badge.svg)](https://github.com/ferriskey/terraform-provider-ferriskey/actions/workflows/test.yml)

Manage the configuration of a [FerrisKey](https://ferriskey.rs) CIAM/IAM
instance as Infrastructure-as-Code — realms, clients, users, roles and
organizations — in the spirit of the Keycloak and Auth0 providers.

This provider is **not** a deployment tool for FerrisKey itself (that remains
the role of the Helm chart). It is a declarative HTTP client of the FerrisKey
REST API.

- Built with the modern [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) (protocol v6).
- Two OAuth2 grant types: **password** (bootstrap) and **client credentials** (steady state).
- Automatic token caching and refresh — long `apply` runs that outlive a token are handled transparently.
- All configuration overridable via `FERRISKEY_*` environment variables (no secrets in `.tf`).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- A reachable FerrisKey instance (API >= 0.6)
- [Go](https://go.dev/dl/) >= 1.24 (to build from source)

## Using the provider

```terraform
terraform {
  required_providers {
    ferriskey = {
      source = "ferriskey/ferriskey"
    }
  }
}

provider "ferriskey" {
  url           = "https://auth.example.com"
  realm         = "master"
  client_id     = "terraform-runner"
  client_secret = var.ferriskey_client_secret # or FERRISKEY_CLIENT_SECRET
}

resource "ferriskey_realm" "acme" {
  name = "acme"
}

resource "ferriskey_client" "web" {
  realm         = ferriskey_realm.acme.name
  client_id     = "acme-web"
  name          = "Acme Web App"
  client_type   = "confidential"
  redirect_uris = ["https://app.acme.example/callback"]
}
```

See the [bootstrap guide](docs/guides/bootstrap.md) for the recommended
two-phase pattern when standing up a fresh instance.

## Resources & data sources

| Type          | Name                              | Notes                                                       |
|---------------|-----------------------------------|-------------------------------------------------------------|
| Resource      | `ferriskey_realm`                 | Root container; imported by name                            |
| Resource      | `ferriskey_realm_settings`        | Realm security settings (token lifetimes, registration, passkey, …) |
| Resource      | `ferriskey_client`                | Confidential/public/system; redirect URIs as a set; exposes `service_account_user_id` |
| Resource      | `ferriskey_user`                  | Username, email, required actions                           |
| Resource      | `ferriskey_role`                  | Realm role with a permission set                            |
| Resource      | `ferriskey_organization`          | Multi-tenancy unit                                          |
| Resource      | `ferriskey_organization_member`   | Adds a user to an organization                              |
| Resource      | `ferriskey_user_role`             | Assigns a role to a user / service account                  |
| Resource      | `ferriskey_password_policy`       | Realm password policy                                       |
| Resource      | `ferriskey_smtp_config`           | Realm SMTP settings (write-only password)                   |
| Resource      | `ferriskey_identity_provider`     | Federation broker (OIDC/SAML)                               |
| Data source   | `ferriskey_realm`                 | Read a realm and its settings                               |
| Data source   | `ferriskey_role`                  | Look up a role by name (e.g. built-in `staff`, `master-realm`) |
| Data source   | `ferriskey_openid_configuration`  | Read the realm's OIDC discovery document                    |

Realm-scoped resources use composite IDs of the form `{realm}/{uuid}` for
`terraform import`. Realms are imported by name.

```bash
terraform import ferriskey_realm.acme acme
terraform import ferriskey_client.web acme/3a8c6128-1111-2222-3333-444455556666
```

## Authentication

The provider authenticates against a realm's OIDC token endpoint. Provide
either:

- **Password grant** (bootstrap): `username` + `password` + a public `client_id`
  (e.g. `admin-cli`), or
- **Client credentials** (steady state): `client_id` + `client_secret`.

Every argument has a `FERRISKEY_*` environment-variable equivalent — see the
bootstrap guide for the full table.

### TLS and resilience

For on-prem instances behind a private CA or (in development) an untrusted
certificate, the provider accepts `ca_cert` (PEM, e.g. `file("ca.pem")`) and
`tls_insecure_skip_verify` (env: `FERRISKEY_CA_CERT` / `FERRISKEY_TLS_INSECURE`).
Transient failures (HTTP 429 and, for idempotent requests, 5xx and connection
errors) are retried with exponential backoff honoring `Retry-After`; the token
is cached and refreshed automatically so long `apply` runs don't fail on
expiry.

## Development

```bash
make build      # build the provider binary
make test       # unit tests (no network)
make testacc    # acceptance tests against a live instance (needs TF_ACC + FERRISKEY_*)
make docs       # regenerate docs/ with tfplugindocs
make fmt vet    # format and vet
```

### Local install (dev overrides)

Add a `~/.terraformrc` pointing at your `GOBIN`:

```hcl
provider_installation {
  dev_overrides {
    "ferriskey/ferriskey" = "/Users/you/go/bin"
  }
  direct {}
}
```

Then `make install` and use the provider without `terraform init`.

### Acceptance tests

Acceptance tests create and destroy real objects against a live FerrisKey
instance. They run only when `TF_ACC=1` and the `FERRISKEY_*` variables are set:

```bash
export TF_ACC=1
export FERRISKEY_URL=http://localhost:3333
export FERRISKEY_REALM=master
export FERRISKEY_CLIENT_ID=admin-cli
export FERRISKEY_USERNAME=admin
export FERRISKEY_PASSWORD=admin
make testacc
```

## Releasing

Releases are produced by [GoReleaser](https://goreleaser.com/) and published as
GPG-signed artifacts in the layout the Terraform Registry expects. Push a semver
tag to trigger the release workflow:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The repository must define the `GPG_PRIVATE_KEY` and `PASSPHRASE` secrets.

## License

Mozilla Public License 2.0. See [LICENSE](LICENSE).
