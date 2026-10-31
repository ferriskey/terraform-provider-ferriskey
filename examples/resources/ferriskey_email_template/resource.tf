resource "ferriskey_email_template" "reset" {
  realm      = "acme"
  name       = "Password reset"
  email_type = "reset_password"

  structure = jsonencode({
    children = [{
      type = "mj-section"
      children = [{
        type    = "mj-text"
        content = "Hello {{user.first_name}}, reset your password: {{reset_link}}"
      }]
    }]
  })
}
