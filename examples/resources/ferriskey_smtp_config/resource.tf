resource "ferriskey_smtp_config" "example" {
  realm      = "acme"
  host       = "smtp.example.com"
  port       = 587
  username   = "mailer"
  password   = var.smtp_password # write-only; never returned by the API
  from_email = "no-reply@acme.example"
  from_name  = "Acme"
  encryption = "tls"
}
