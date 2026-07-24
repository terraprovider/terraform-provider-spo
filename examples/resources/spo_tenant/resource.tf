# Organization-wide SharePoint Online tenant settings (singleton). Only the
# attributes you declare are written; everything else is left as-is.
resource "spo_tenant" "this" {
  sharing_capability              = "ExternalUserAndGuestSharing"
  one_drive_storage_quota         = 1048576
  comments_on_site_pages_disabled = true
  legacy_auth_protocols_enabled   = false
}
