# Promote an existing site to a hub site (keyed by the site URL).
resource "spo_hub_site" "intranet" {
  identity               = "https://contoso.sharepoint.com/sites/intranet"
  title                  = "Contoso Intranet"
  requires_join_approval = true
}
