terraform {
  required_providers {
    spo = { source = "terraprovider/spo" }
  }
}

# Auth resolves ARM_*/AZURE_* env vars (aligned with hashicorp/azuread). App-only
# over CSOM requires SharePoint Sites.FullControl.All AND a certificate credential —
# client secrets are rejected by SharePoint.
provider "spo" {
  tenant_name = "contoso" # -> https://contoso-admin.sharepoint.com (or set admin_url)
}
