# Requires a SharePoint Premium license.
resource "spo_restricted_search" "this" {
  mode         = "Enabled"
  allowed_list = ["https://contoso.sharepoint.com/sites/marketing"]
}
