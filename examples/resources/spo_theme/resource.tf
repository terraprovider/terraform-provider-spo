resource "spo_theme" "contoso" {
  identity = "Contoso"
  theme_json = jsonencode({
    palette = {
      themePrimary   = "#0078d4"
      themeSecondary = "#106ebe"
    }
  })
}
