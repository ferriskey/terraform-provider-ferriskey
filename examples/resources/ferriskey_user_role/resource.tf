# Grant a client's service account a built-in admin role (bootstrap pattern).
data "ferriskey_role" "staff" {
  realm = "master"
  name  = "staff"
}

resource "ferriskey_client" "runner" {
  realm                   = "master"
  client_id               = "terraform-runner"
  client_type             = "confidential"
  service_account_enabled = true
}

resource "ferriskey_user_role" "runner_staff" {
  realm   = "master"
  user_id = ferriskey_client.runner.service_account_user_id
  role_id = data.ferriskey_role.staff.id
}
