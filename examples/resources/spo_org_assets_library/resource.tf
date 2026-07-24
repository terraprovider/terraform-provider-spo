# Register a document library as an organization assets library so its
# images / templates / fonts surface across the org. The tenant Office 365 CDN
# must be enabled for the chosen cdn_type (Public or Private, default Private).
resource "spo_org_assets_library" "brand_images" {
  identity       = "https://contoso.sharepoint.com/sites/brand/SiteAssets"
  org_asset_type = "ImageDocumentLibrary" # or OfficeTemplateLibrary / OfficeFontLibrary
  cdn_type       = "Private"
}
