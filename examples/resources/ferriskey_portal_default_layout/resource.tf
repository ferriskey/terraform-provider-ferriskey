resource "ferriskey_portal_default_layout" "this" {
  realm     = "acme"
  layout_id = ferriskey_portal_layout.frame.layout_id
}
