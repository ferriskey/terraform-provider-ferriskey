resource "ferriskey_webhook" "audit" {
  realm       = "acme"
  name        = "audit"
  endpoint    = "https://hooks.example.com/ferriskey"
  subscribers = ["user.created", "user.deleted"]

  headers = {
    "X-Token" = "s3cret"
  }

  retry_policy = {
    max_attempts = 5
  }

  rotate_secret_trigger = "2024-01"
}

output "audit_webhook_secret" {
  value     = ferriskey_webhook.audit.secret
  sensitive = true
}
