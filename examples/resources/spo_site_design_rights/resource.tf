resource "spo_site_design_rights" "team" {
  identity   = spo_site_design.team.id
  principals = ["group@contoso.com", "user@contoso.com"]
}
