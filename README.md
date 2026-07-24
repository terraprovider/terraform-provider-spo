# terraform-provider-spo

A Terraform/OpenTofu provider for **SharePoint Online tenant/site admin**
configuration — the **CSOM** admin surface behind the SharePoint Online Management
Shell (`Microsoft.Online.SharePoint.PowerShell`). Resources are **generated** from a
derived catalog via [`go-spo`](https://github.com/terraprovider/go-spo); the runtime
builds on [`tf-msadmin`](https://github.com/terraprovider/tf-msadmin).

> Not affiliated with or endorsed by Microsoft.

## Status

Sixteen resources cover the SharePoint Online Management Shell admin surface (CSOM).
Each also exposes a read-only data source of the same name.

**Configuration (adopt existing; not deleted on destroy)**

| Resource | Cmdlets | Notes |
|---|---|---|
| `spo_tenant` | `Get-/Set-SPOTenant` | org-wide singleton, 184 typed knobs |
| `spo_site` | `Get-/Set-SPOSite` | per-site collection settings, keyed by URL |
| `spo_tenant_cdn` | `Get-/Set-SPOTenantCdnEnabled` (+ origins) | Public/Private CDN |
| `spo_knowledge_hub_site` | `Set-/Get-/Remove-SPOKnowledgeHubSite` | singleton |
| `spo_home_site` | `Set-/Get-/Remove-SPOHomeSite` | singleton |
| `spo_restricted_search` | `Set-/Get-SPORestrictedSearchMode` | singleton; needs SharePoint Premium |

**Objects (full create/read/update/delete)**

| Resource | Cmdlets | Notes |
|---|---|---|
| `spo_site_script` | `Add-/Get-/Set-/Remove-SPOSiteScript` | write-only `content` |
| `spo_site_design` | `Add-/Get-/Set-/Remove-SPOSiteDesign` | references site scripts |
| `spo_list_design` | `Add-/Get-/Remove-SPOListDesign` | immutable (change → replace) |
| `spo_theme` | `Add-/Get-/Set-/Remove-SPOTheme` | keyed by name |
| `spo_hub_site` | `Register-/Get-/Set-/Unregister-SPOHubSite` | promote a site to a hub |
| `spo_org_news_site` | `Set-/Get-/Remove-SPOOrgNewsSite` | designate a comms site |
| `spo_org_assets_library` | `Add-/Get-/Set-/Remove-SPOOrgAssetsLibrary` | needs the CDN enabled |
| `spo_storage_entity` | `Set-/Get-/Remove-SPOStorageEntity` | tenant property; keyed `"<site>\|<key>"`, on the app catalog |
| `spo_site_design_rights` | `Grant-/Revoke-/Get-SPOSiteDesignRights` | View grants on a design |
| `spo_blocked_page_content_type` | `Set-/Get-/Remove-SPOBlockedPageCreationContentType` | block a page type |

**Deferred.** After a full sweep of the `Microsoft.Online.SharePoint.PowerShell`
cmdlet surface, the following are intentionally out of the initial scope:

- **Graph-first** — identities, sites, containers, SPNs and unified groups that have a
  first-class Microsoft Graph API are managed there, not via CSOM (original scope call).
- **Complex config object** — `content_security_policy` (`Add-/Get-SPOContentSecurityPolicy`).
- **Require tenant capabilities not present on the reference tenant** — `data_encryption`/BYOK
  (key vault), multi-geo / `geo_admin` / identity mapping, `font_package` (a font file),
  `portal_launch_waves`, and `site_collection_app_catalog` (a provisioned tenant app catalog).
- **Need an un-validatable wire shape** (implementable, but their exact CSOM encoding can't be
  exercised on the reference tenant, so they are held back rather than shipped unverified):
  `hub_site_rights` (`Grant-/Revoke-SPOHubSiteRights` — reading the hub `Permissions` child
  collection needs a live hub site) and `browser_idle_sign_out`
  (`Set-SPOBrowserIdleSignOut` — needs `TimeSpan` parameter support in the transport).

`storage_entity` is shipped but only fully round-trips where a tenant app catalog exists; its
wire is byte-identical to `Set-/Get-/Remove-SPOStorageEntity` and reaches the CSOM method live.

```hcl
provider "spo" {
  tenant_name = "contoso" # → https://contoso-admin.sharepoint.com (or set admin_url)
  # Auth mirrors AzureAD/AzureRM: ARM_*/AZURE_* env. App-only needs SharePoint
  # Sites.FullControl.All + a certificate credential (client secrets are rejected).
}

resource "spo_tenant" "this" {
  sharing_capability              = "ExternalUserSharingOnly"
  one_drive_storage_quota         = 1048576
  comments_on_site_pages_disabled = true
}
```

`spo_tenant` is a **singleton**: it adopts the existing tenant configuration, applies
only the settings you declare (sparse writes, like `Set-SPOTenant`), and is not
deleted on destroy.

## Configuration

| Attribute | Env | Notes |
|---|---|---|
| `tenant_name` | `SPO_TENANT_NAME` | the `{tenant}` in `{tenant}-admin.sharepoint.com` |
| `admin_url` | `SPO_ADMIN_URL` | full admin URL; required for national clouds; overrides `tenant_name` |
| auth block (`tenant_id`, `client_id`, `client_certificate_path`, …) | `ARM_*`/`AZURE_*` | aligned field-for-field with `hashicorp/azuread` |

The SharePoint token audience is the admin URL itself.

> ⚠️ **Client-secret app-only auth is not supported.** SharePoint's CSOM endpoint
> rejects secret-based app tokens with `401 Unsupported app only token` — it requires
> a **certificate** credential (`client_certificate` / `client_certificate_path`, or
> `ARM_CLIENT_CERTIFICATE*`). The secret is refused before the permission check, so a
> certificate is mandatory. Grant the app SharePoint `Sites.FullControl.All` and add a
> certificate. (Confirmed live: a client secret returns 401 regardless of permissions.)

## Architecture

```
spo-powershell-api-re (private)  reflect module → catalog + golden ProcessQuery fixtures
        │ publish
        ▼
go-spo   spoapi (CSOM ProcessQuery transport) + generated spo bindings
        │ go get
        ▼
terraform-provider-spo   cmd/gen-tf → genframework → internal/provider/*_resource.go
```

Regenerate resources after a catalog bump: `go generate .` (runs `cmd/gen-tf`).

## Development

```bash
go build ./... && go test ./...          # unit + schema-validation test
go run ./cmd/gen-tf                       # regenerate internal/provider from the catalog
```

Validate against real HCL with a dev override (see `examples/tenant`):

```bash
go install .
cat > dev.tfrc <<'EOF'
provider_installation { dev_overrides { "terraprovider/spo" = "<GOBIN>" }; direct {} }
EOF
TF_CLI_CONFIG_FILE=dev.tfrc terraform -chdir=examples/tenant validate
```

## License

[MIT](LICENSE) © glueckanja AG
