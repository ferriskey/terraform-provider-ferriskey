data "ferriskey_role" "staff" {
  realm = "master"
  name  = "staff"
}

output "staff_role_id" {
  value = data.ferriskey_role.staff.id
}
