# Tenant storage entities (farm property-bag values) are written on the app
# catalog site. The identity is "<site-url>|<key>".
resource "spo_storage_entity" "brand_color" {
  identity    = "https://contoso.sharepoint.com/sites/appcatalog|BrandPrimaryColor"
  value       = "#0078d4"
  description = "Corporate primary brand colour"
  comment     = "Managed by Terraform"
}
