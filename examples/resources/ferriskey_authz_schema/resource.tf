resource "ferriskey_authz_schema" "main" {
  realm     = "acme"
  namespace = "App"

  source = <<-EOT
    namespace App {
      entity Doc;
      action read appliesTo { principal: [FerrisKey::User], resource: [Doc] };
    }
  EOT
}
