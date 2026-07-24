# Manage the admin settings of an existing site collection (keyed by URL). Does not
# create or delete the site collection itself.
resource "spo_site" "marketing" {
  identity              = "https://contoso.sharepoint.com/sites/marketing"
  title                 = "Marketing"
  storage_maximum_level = 26214400
  sharing_capability    = "ExternalUserSharingOnly"
}
