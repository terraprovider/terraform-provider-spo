resource "spo_site_design" "team" {
  title        = "Contoso team site"
  web_template = "64" # 64 = team site, 68 = communication site, 1 = group
  description  = "Standard configuration for new team sites."
  site_scripts = [spo_site_script.brand.id]
  is_default   = false
}
