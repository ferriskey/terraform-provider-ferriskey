data "ferriskey_email_template_variables" "reset" {
  email_type = "reset_password"
}

output "reset_variables" {
  value = [for v in data.ferriskey_email_template_variables.reset.variables : v.name]
}
