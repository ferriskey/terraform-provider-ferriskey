resource "ferriskey_portal_layout" "frame" {
  realm = "acme"
  name  = "frame"

  tree = jsonencode({
    type     = "root"
    children = []
  })
}
