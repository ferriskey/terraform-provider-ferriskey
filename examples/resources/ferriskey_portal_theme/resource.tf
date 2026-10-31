resource "ferriskey_portal_theme" "brand" {
  realm     = "acme"
  name      = "brand"
  layout_id = ferriskey_portal_layout.frame.layout_id

  config = jsonencode({
    colors = {
      primaryButton = "#0f766e"
      links         = "#0f766e"
    }
    spacing = {
      widgetPadding = 32
    }
  })
}
