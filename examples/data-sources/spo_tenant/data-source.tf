data "spo_tenant" "current" {}

output "sharing_capability" {
  value = data.spo_tenant.current.sharing_capability
}
