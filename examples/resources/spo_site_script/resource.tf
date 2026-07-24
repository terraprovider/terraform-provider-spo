resource "spo_site_script" "brand" {
  title       = "Apply brand theme"
  description = "Applies the corporate theme to new sites."
  content = jsonencode({
    "$schema" = "schema.json"
    actions   = [{ verb = "applyTheme", themeName = "Contoso" }]
    bindata   = {}
    version   = 1
  })
}
