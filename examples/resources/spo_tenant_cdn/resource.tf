resource "spo_tenant_cdn" "public" {
  identity = "Public" # "Public" or "Private"
  enabled  = true
  origins  = ["*/MASTERPAGE", "*/STYLE LIBRARY"]
}
