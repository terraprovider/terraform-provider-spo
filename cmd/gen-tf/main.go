// Command gen-tf generates the terraform-plugin-framework resource(s) for the
// SharePoint Online provider from the go-spo catalog. It is the SPO-specific
// frontend to the reusable tf-msadmin/genframework engine.
//
// The first slice is the tenant-settings singleton: Get-/Set-SPOTenant → the
// spo_tenant Config+Singleton resource, one attribute per settable admin knob.
// Attribute typing mirrors go-spo's gen-go classification so the emitted resource
// references only fields that exist on spo.SetSPOTenantParams.
//
//	go run ./cmd/gen-tf
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/terraprovider/go-spo/spec"
	"github.com/terraprovider/tf-msadmin/genframework"
)

func main() {
	out := flag.String("out", "internal/provider", "output directory")
	flag.Parse()

	tenant, err := spec.Tenant()
	check(err)
	site, err := spec.Site()
	check(err)

	cfg := genframework.Config{
		Package:        "provider",
		ClientsImport:  "github.com/terraprovider/terraform-provider-spo/internal/clients",
		ClientField:    "SPO",
		BindingsImport: "github.com/terraprovider/go-spo/spo",
		BindingsPkg:    "spo",
	}

	tenantAttrs, tSkip := buildAttrs(tenant)
	tenantRes := genframework.Resource{
		Noun:        "Tenant",
		TFName:      "tenant",
		Description: "Manages SharePoint Online organization-wide tenant settings via Get-/Set-SPOTenant (CSOM). This is a singleton: it adopts the existing tenant configuration and is not deleted on destroy.",
		Attributes:  tenantAttrs,
		Read:        genframework.Op{Method: "GetSPOTenant", Params: "GetSPOTenantParams"},
		Update:      genframework.Op{Method: "SetSPOTenant", Params: "SetSPOTenantParams"},
		Config:      true,
		Singleton:   true,
		SparseWrite: true,
	}

	siteAttrs, sSkip := buildAttrs(site)
	siteRes := genframework.Resource{
		Noun:        "Site",
		TFName:      "site",
		Description: "Manages the admin settings of an existing SharePoint Online site collection via Get-/Set-SPOSite (CSOM), keyed by the site URL (identity). It adopts an existing site and is not deleted on destroy (it does not create or remove the site collection itself).",
		Attributes:  siteAttrs,
		// Config + non-singleton: identity (the site URL) is a required, replace-forcing key.
		Read:        genframework.Op{Method: "GetSPOSite", Params: "GetSPOSiteParams", IdentityField: "Identity"},
		Update:      genframework.Op{Method: "SetSPOSite", Params: "SetSPOSiteParams", IdentityField: "Identity"},
		Config:      true,
		Singleton:   false,
		SparseWrite: true,
	}

	// spo_site_script — full method-CRUD (create/read/update/delete real objects),
	// hand-defined attributes (its inputs are a creation-info type, not SiteProperties).
	siteScriptRes := genframework.Resource{
		Noun:        "SiteScript",
		TFName:      "site_script",
		Description: "Manages a SharePoint site script (Add/Get/Set/Remove-SPOSiteScript, CSOM) — a JSON document consumed by site designs.",
		Attributes: []genframework.Attribute{
			{TFName: "title", Field: "Title", APIName: "Title", Type: genframework.TypeString, Required: true, InCreate: true, InUpdate: true, Description: "The site script display name."},
			{TFName: "description", Field: "Description", APIName: "Description", Type: genframework.TypeString, Computed: true, InCreate: true, InUpdate: true, Description: "The site script description."},
			{TFName: "content", Field: "Content", APIName: "Content", Type: genframework.TypeString, Required: true, WriteOnly: true, InCreate: true, InUpdate: true, Description: "The site script JSON (the actions document)."},
		},
		Create: genframework.Op{Method: "NewSPOSiteScript", Params: "NewSPOSiteScriptParams"},
		Read:   genframework.Op{Method: "GetSPOSiteScript", Params: "GetSPOSiteScriptParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "SetSPOSiteScript", Params: "SetSPOSiteScriptParams", IdentityField: "Identity"},
		Delete: genframework.Op{Method: "RemoveSPOSiteScript", Params: "RemoveSPOSiteScriptParams", IdentityField: "Identity"},
	}

	// spo_site_design — method-CRUD; references site scripts by GUID.
	siteDesignRes := genframework.Resource{
		Noun:        "SiteDesign",
		TFName:      "site_design",
		Description: "Manages a SharePoint site design (Add/Get/Set/Remove-SPOSiteDesign, CSOM) — applies one or more site scripts when a site is created or the design is invoked.",
		Attributes: []genframework.Attribute{
			{TFName: "title", Field: "Title", APIName: "Title", Type: genframework.TypeString, Required: true, InCreate: true, InUpdate: true, Description: "The site design display name."},
			{TFName: "web_template", Field: "WebTemplate", APIName: "WebTemplate", Type: genframework.TypeString, Required: true, InCreate: true, InUpdate: true, Description: "Site template the design applies to: 64 (team site), 68 (communication site), 1 (group)."},
			{TFName: "site_scripts", Field: "SiteScripts", APIName: "SiteScriptIds", Type: genframework.TypeStringSet, Computed: true, InCreate: true, InUpdate: true, Description: "Ordered site-script GUIDs the design runs."},
			{TFName: "description", Field: "Description", APIName: "Description", Type: genframework.TypeString, Computed: true, InCreate: true, InUpdate: true, Description: "The site design description."},
			{TFName: "preview_image_url", Field: "PreviewImageUrl", APIName: "PreviewImageUrl", Type: genframework.TypeString, Computed: true, InCreate: true, InUpdate: true, Description: "Preview image URL."},
			{TFName: "preview_image_alt_text", Field: "PreviewImageAltText", APIName: "PreviewImageAltText", Type: genframework.TypeString, Computed: true, InCreate: true, InUpdate: true, Description: "Preview image alt text."},
			{TFName: "is_default", Field: "IsDefault", APIName: "IsDefault", Type: genframework.TypeBool, Computed: true, InCreate: true, InUpdate: true, Description: "Whether the design applies to all new sites of the web template."},
		},
		Create: genframework.Op{Method: "NewSPOSiteDesign", Params: "NewSPOSiteDesignParams"},
		Read:   genframework.Op{Method: "GetSPOSiteDesign", Params: "GetSPOSiteDesignParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "SetSPOSiteDesign", Params: "SetSPOSiteDesignParams", IdentityField: "Identity"},
		Delete: genframework.Op{Method: "RemoveSPOSiteDesign", Params: "RemoveSPOSiteDesignParams", IdentityField: "Identity"},
	}

	// spo_theme — CRUD keyed by the theme name (IdentityIsName). The palette JSON is
	// write-only (the API returns a parsed palette, not the source JSON).
	themeRes := genframework.Resource{
		Noun:           "Theme",
		TFName:         "theme",
		Description:    "Manages a SharePoint tenant theme (Add/Get/Set/Remove-SPOTheme, CSOM), keyed by the theme name.",
		IdentityIsName: true,
		Attributes: []genframework.Attribute{
			{TFName: "theme_json", Field: "ThemeJson", APIName: "Palette", Type: genframework.TypeString, Required: true, WriteOnly: true, InCreate: true, InUpdate: true, Description: "The theme palette as a JSON string (e.g. {\"palette\":{\"themePrimary\":\"#0078d4\"}})."},
			{TFName: "is_inverted", Field: "IsInverted", APIName: "IsInverted", Type: genframework.TypeBool, Computed: true, Description: "Whether the theme is a dark (inverted) theme."},
		},
		Create: genframework.Op{Method: "NewSPOTheme", Params: "NewSPOThemeParams"},
		Read:   genframework.Op{Method: "GetSPOTheme", Params: "GetSPOThemeParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "SetSPOTheme", Params: "SetSPOThemeParams", IdentityField: "Identity"},
		Delete: genframework.Op{Method: "RemoveSPOTheme", Params: "RemoveSPOThemeParams", IdentityField: "Identity"},
	}

	// spo_hub_site — promote an existing site to a hub (CRUD keyed by site URL).
	hubSiteRes := genframework.Resource{
		Noun:           "HubSite",
		TFName:         "hub_site",
		Description:    "Registers an existing SharePoint site as a hub site and manages its hub settings (Register/Get/Set/Unregister-SPOHubSite, CSOM), keyed by the site URL.",
		IdentityIsName: true,
		Attributes: []genframework.Attribute{
			{TFName: "title", Field: "Title", APIName: "Title", Type: genframework.TypeString, Computed: true, InCreate: true, InUpdate: true, Description: "The hub site display name."},
			{TFName: "description", Field: "Description", APIName: "Description", Type: genframework.TypeString, Computed: true, InCreate: true, InUpdate: true, Description: "The hub site description."},
			{TFName: "logo_url", Field: "LogoUrl", APIName: "LogoUrl", Type: genframework.TypeString, Computed: true, InCreate: true, InUpdate: true, Description: "The hub site logo URL."},
			{TFName: "requires_join_approval", Field: "RequiresJoinApproval", APIName: "RequiresJoinApproval", Type: genframework.TypeBool, Computed: true, InCreate: true, InUpdate: true, Description: "Whether joining the hub requires approval."},
			{TFName: "hub_site_id", Field: "HubSiteId", APIName: "HubSiteId", Type: genframework.TypeString, Computed: true, Description: "The hub site GUID (server-assigned)."},
		},
		Create: genframework.Op{Method: "NewSPOHubSite", Params: "NewSPOHubSiteParams"},
		Read:   genframework.Op{Method: "GetSPOHubSite", Params: "GetSPOHubSiteParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "SetSPOHubSite", Params: "SetSPOHubSiteParams", IdentityField: "Identity"},
		Delete: genframework.Op{Method: "RemoveSPOHubSite", Params: "RemoveSPOHubSiteParams", IdentityField: "Identity"},
	}

	// --- tenant admin config resources ---
	tenantCdnRes := genframework.Resource{
		Noun:        "TenantCdn",
		TFName:      "tenant_cdn",
		Description: "Manages the Office 365 CDN for a type (Public or Private): enabled state and origins (Get/Set-SPOTenantCdnEnabled + Add/Remove-SPOTenantCdnOrigin, CSOM). Keyed by cdn type; adopts the existing config and is not disabled on destroy.",
		Config:      true,
		Attributes: []genframework.Attribute{
			{TFName: "enabled", Field: "Enabled", APIName: "Enabled", Type: genframework.TypeBool, Required: true, InCreate: true, InUpdate: true, Description: "Whether the CDN of this type is enabled."},
			{TFName: "origins", Field: "Origins", APIName: "Origins", Type: genframework.TypeStringSet, Computed: true, InCreate: true, InUpdate: true, Description: "CDN origins (relative or absolute paths)."},
		},
		Read:   genframework.Op{Method: "GetSPOTenantCdn", Params: "GetSPOTenantCdnParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "SetSPOTenantCdn", Params: "SetSPOTenantCdnParams", IdentityField: "Identity"},
	}

	knowledgeHubRes := genframework.Resource{
		Noun:        "KnowledgeHubSite",
		TFName:      "knowledge_hub_site",
		Description: "Manages the tenant knowledge hub site (Set/Get/Remove-SPOKnowledgeHubSite, CSOM). Singleton; not cleared on destroy.",
		Config:      true, Singleton: true,
		Attributes: []genframework.Attribute{
			{TFName: "url", Field: "Url", APIName: "Url", Type: genframework.TypeString, Required: true, InCreate: true, InUpdate: true, Description: "The knowledge hub site URL."},
		},
		Read:   genframework.Op{Method: "GetSPOKnowledgeHubSite", Params: "GetSPOKnowledgeHubSiteParams"},
		Update: genframework.Op{Method: "SetSPOKnowledgeHubSite", Params: "SetSPOKnowledgeHubSiteParams"},
	}

	restrictedSearchRes := genframework.Resource{
		Noun:        "RestrictedSearch",
		TFName:      "restricted_search",
		Description: "Manages tenant restricted search (mode + allowed site list; Set/Get-SPORestrictedSearchMode + Add/Remove-SPORestrictedSearchAllowedList, CSOM). Singleton. Requires a SharePoint Premium license.",
		Config:      true, Singleton: true,
		Attributes: []genframework.Attribute{
			{TFName: "mode", Field: "Mode", APIName: "Mode", Type: genframework.TypeString, Required: true, InCreate: true, InUpdate: true, Description: "Restricted search mode: Disabled or Enabled."},
			{TFName: "allowed_list", Field: "AllowedList", APIName: "AllowedList", Type: genframework.TypeStringSet, Computed: true, InCreate: true, InUpdate: true, Description: "Site URLs allowed in search when restricted."},
		},
		Read:   genframework.Op{Method: "GetSPORestrictedSearch", Params: "GetSPORestrictedSearchParams"},
		Update: genframework.Op{Method: "SetSPORestrictedSearch", Params: "SetSPORestrictedSearchParams"},
	}

	orgNewsRes := genframework.Resource{
		Noun:           "OrgNewsSite",
		TFName:         "org_news_site",
		Description:    "Designates an existing communication site as an organization news site (Set/Get/Remove-SPOOrgNewsSite, CSOM), keyed by the site URL.",
		IdentityIsName: true,
		Attributes: []genframework.Attribute{
			{TFName: "url", Field: "Url", APIName: "Url", Type: genframework.TypeString, Computed: true, Description: "The org news site URL (echoes identity)."},
		},
		Create: genframework.Op{Method: "NewSPOOrgNewsSite", Params: "NewSPOOrgNewsSiteParams"},
		Read:   genframework.Op{Method: "GetSPOOrgNewsSite", Params: "GetSPOOrgNewsSiteParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "NewSPOOrgNewsSite", Params: "NewSPOOrgNewsSiteParams", IdentityField: "Identity"},
		Delete: genframework.Op{Method: "RemoveSPOOrgNewsSite", Params: "RemoveSPOOrgNewsSiteParams", IdentityField: "Identity"},
	}

	siteDesignRightsRes := genframework.Resource{
		Noun:           "SiteDesignRights",
		TFName:         "site_design_rights",
		Description:    "Grants View rights on a site design to principals (Grant/Revoke/Get-SPOSiteDesignRights, CSOM), keyed by the site design GUID.",
		IdentityIsName: true,
		Attributes: []genframework.Attribute{
			{TFName: "principals", Field: "Principals", APIName: "Principals", Type: genframework.TypeStringSet, Required: true, InCreate: true, InUpdate: true, Description: "User/group principal names granted View rights on the design."},
		},
		Create: genframework.Op{Method: "NewSPOSiteDesignRights", Params: "NewSPOSiteDesignRightsParams"},
		Read:   genframework.Op{Method: "GetSPOSiteDesignRights", Params: "GetSPOSiteDesignRightsParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "SetSPOSiteDesignRights", Params: "SetSPOSiteDesignRightsParams", IdentityField: "Identity"},
		Delete: genframework.Op{Method: "RemoveSPOSiteDesignRights", Params: "RemoveSPOSiteDesignRightsParams", IdentityField: "Identity"},
	}

	homeSiteRes := genframework.Resource{
		Noun:        "HomeSite",
		TFName:      "home_site",
		Description: "Manages the tenant home site (Set/Get/Remove-SPOHomeSite, CSOM). Singleton; not cleared on destroy.",
		Config:      true, Singleton: true,
		Attributes: []genframework.Attribute{
			{TFName: "url", Field: "Url", APIName: "Url", Type: genframework.TypeString, Required: true, InCreate: true, InUpdate: true, Description: "The home site URL."},
		},
		Read:   genframework.Op{Method: "GetSPOHomeSite", Params: "GetSPOHomeSiteParams"},
		Update: genframework.Op{Method: "SetSPOHomeSite", Params: "SetSPOHomeSiteParams"},
	}

	blockedPageRes := genframework.Resource{
		Noun:           "BlockedPageContentType",
		TFName:         "blocked_page_content_type",
		Description:    "Blocks creation of pages of a content type (Set/Get/Remove-SPOBlockedPageCreationContentType, CSOM). Keyed by content type: StandardPage, WikiPage, FormPage, ClientSidePage.",
		IdentityIsName: true,
		Attributes: []genframework.Attribute{
			{TFName: "content_type", Field: "ContentType", APIName: "Id", Type: genframework.TypeString, Computed: true, Description: "The blocked content type (echoes identity)."},
		},
		Create: genframework.Op{Method: "NewSPOBlockedPageContentType", Params: "NewSPOBlockedPageContentTypeParams"},
		Read:   genframework.Op{Method: "GetSPOBlockedPageContentType", Params: "GetSPOBlockedPageContentTypeParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "NewSPOBlockedPageContentType", Params: "NewSPOBlockedPageContentTypeParams", IdentityField: "Identity"},
		Delete: genframework.Op{Method: "RemoveSPOBlockedPageContentType", Params: "RemoveSPOBlockedPageContentTypeParams", IdentityField: "Identity"},
	}

	// spo_storage_entity — tenant storage entity (site.RootWeb.Set/Get/RemoveStorageEntity,
	// CSOM), keyed by "<site-url>|<key>" (typically the app catalog site).
	storageEntityRes := genframework.Resource{
		Noun:           "StorageEntity",
		TFName:         "storage_entity",
		Description:    "Manages a SharePoint tenant storage entity / farm property bag value (Set/Get/Remove-SPOStorageEntity, CSOM). Identity is \"<site-url>|<key>\" — the site (usually the tenant app catalog) and the entity key.",
		IdentityIsName: true,
		Attributes: []genframework.Attribute{
			{TFName: "value", Field: "Value", APIName: "Value", Type: genframework.TypeString, Required: true, InCreate: true, InUpdate: true, Description: "The storage entity value."},
			{TFName: "description", Field: "Description", APIName: "Description", Type: genframework.TypeString, Computed: true, InCreate: true, InUpdate: true, Description: "The storage entity description."},
			{TFName: "comment", Field: "Comment", APIName: "Comment", Type: genframework.TypeString, Computed: true, InCreate: true, InUpdate: true, Description: "The storage entity comment."},
		},
		Create: genframework.Op{Method: "NewSPOStorageEntity", Params: "NewSPOStorageEntityParams"},
		Read:   genframework.Op{Method: "GetSPOStorageEntity", Params: "GetSPOStorageEntityParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "SetSPOStorageEntity", Params: "SetSPOStorageEntityParams", IdentityField: "Identity"},
		Delete: genframework.Op{Method: "RemoveSPOStorageEntity", Params: "RemoveSPOStorageEntityParams", IdentityField: "Identity"},
	}

	// spo_org_assets_library — register a document library as an organization
	// assets source (Add/Get/Set/Remove-SPOOrgAssetsLibrary, CSOM), keyed by URL.
	orgAssetsRes := genframework.Resource{
		Noun:           "OrgAssetsLibrary",
		TFName:         "org_assets_library",
		Description:    "Registers a document library as an organization assets library (Add/Get/Set/Remove-SPOOrgAssetsLibrary, CSOM), keyed by the library URL. Surfaces its images/templates/fonts to the org.",
		IdentityIsName: true,
		Attributes: []genframework.Attribute{
			{TFName: "org_asset_type", Field: "OrgAssetType", APIName: "OrgAssetType", Type: genframework.TypeString, Required: true, InCreate: true, InUpdate: true, Description: "Asset type: ImageDocumentLibrary, OfficeTemplateLibrary or OfficeFontLibrary."},
			{TFName: "thumbnail_url", Field: "ThumbnailUrl", APIName: "ThumbnailUrl", Type: genframework.TypeString, Computed: true, InCreate: true, InUpdate: true, Description: "Optional thumbnail image URL for the library."},
			{TFName: "cdn_type", Field: "CdnType", APIName: "CdnType", Type: genframework.TypeString, Computed: true, Replace: true, InCreate: true, Description: "CDN the library is served from: Public or Private (default Private). Changing it forces replacement."},
		},
		Create: genframework.Op{Method: "NewSPOOrgAssetsLibrary", Params: "NewSPOOrgAssetsLibraryParams"},
		Read:   genframework.Op{Method: "GetSPOOrgAssetsLibrary", Params: "GetSPOOrgAssetsLibraryParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "SetSPOOrgAssetsLibrary", Params: "SetSPOOrgAssetsLibraryParams", IdentityField: "Identity"},
		Delete: genframework.Op{Method: "RemoveSPOOrgAssetsLibrary", Params: "RemoveSPOOrgAssetsLibraryParams", IdentityField: "Identity"},
	}

	// spo_list_design — immutable (create/read/delete; no update — every attribute
	// forces replacement). References site scripts by GUID.
	listDesignRes := genframework.Resource{
		Noun:        "ListDesign",
		TFName:      "list_design",
		Description: "Manages a SharePoint list design (Add/Get/Remove-SPOListDesign, CSOM) — applies site scripts to a list. List designs are immutable: any change recreates the design.",
		Attributes: []genframework.Attribute{
			{TFName: "title", Field: "Title", APIName: "Title", Type: genframework.TypeString, Required: true, Replace: true, InCreate: true, Description: "The list design display name."},
			{TFName: "description", Field: "Description", APIName: "Description", Type: genframework.TypeString, Computed: true, Replace: true, InCreate: true, Description: "The list design description."},
			{TFName: "site_scripts", Field: "SiteScripts", APIName: "SiteScriptIds", Type: genframework.TypeStringSet, Required: true, Replace: true, InCreate: true, Description: "Ordered site-script GUIDs the design runs (at least one is required)."},
		},
		Create: genframework.Op{Method: "NewSPOListDesign", Params: "NewSPOListDesignParams"},
		Read:   genframework.Op{Method: "GetSPOListDesign", Params: "GetSPOListDesignParams", IdentityField: "Identity"},
		Update: genframework.Op{Method: "SetSPOListDesign", Params: "SetSPOListDesignParams", IdentityField: "Identity"},
		Delete: genframework.Op{Method: "RemoveSPOListDesign", Params: "RemoveSPOListDesignParams", IdentityField: "Identity"},
	}

	files, err := genframework.Generate(cfg, []genframework.Resource{tenantRes, siteRes, siteScriptRes, siteDesignRes, themeRes, hubSiteRes, tenantCdnRes, knowledgeHubRes, restrictedSearchRes, orgNewsRes, siteDesignRightsRes, homeSiteRes, blockedPageRes, storageEntityRes, orgAssetsRes, listDesignRes})
	check(err)
	check(os.MkdirAll(*out, 0o755))
	for _, f := range files {
		check(os.WriteFile(filepath.Join(*out, f.Name), f.Content, 0o644))
	}
	fmt.Printf("gen-tf: spo_tenant (%d attrs, %d skipped) + spo_site (%d attrs, %d skipped), %d files -> %s\n",
		len(tenantAttrs), tSkip, len(siteAttrs), sSkip, len(files), *out)
}

// buildAttrs maps a catalog's knobs to genframework attributes.
func buildAttrs(cat *spec.ObjectCatalog) ([]genframework.Attribute, int) {
	var attrs []genframework.Attribute
	skipped := 0
	for _, p := range cat.Knobs() {
		at, ptr, ok := attrType(p)
		if !ok {
			skipped++
			continue
		}
		field := p.Name // PascalCase, matches the generated Set*Params field
		attrs = append(attrs, genframework.Attribute{
			TFName:       tfName(field),
			Field:        field,
			APIName:      field,
			Type:         at,
			Computed:     true,
			Sensitive:    sensitive(field),
			Description:  describe(p),
			InCreate:     true,
			InUpdate:     true,
			PointerParam: ptr,
		})
	}
	sort.Slice(attrs, func(i, j int) bool { return attrs[i].Field < attrs[j].Field })
	return attrs, skipped
}

// attrType maps a CSOM knob to a Terraform attribute type + whether the go-spo
// binding field is a pointer. Must stay in lockstep with go-spo/cmd/gen-go
// classify (so every emitted attribute references an existing params field).
func attrType(p spec.Property) (t genframework.AttrType, pointer bool, ok bool) {
	if p.IsCollection || p.CsomType == "Array" {
		switch p.ElementType {
		case "Guid", "String":
			return genframework.TypeStringSet, false, true // []string (toStringSlice)
		default:
			return 0, false, false
		}
	}
	switch p.CsomType {
	case "Boolean":
		return genframework.TypeBool, true, true
	case "String", "Guid", "Enum":
		return genframework.TypeString, true, true
	case "Int16", "Int32", "Int64":
		return genframework.TypeInt, true, true
	default: // Double / complex CSOM object types — no field emitted by gen-go
		return 0, false, false
	}
}

func describe(p spec.Property) string {
	d := fmt.Sprintf("Maps to the CSOM %s property.", p.Name)
	if len(p.ValidateSet) > 0 {
		d += " Allowed values: " + strings.Join(p.ValidateSet, ", ") + "."
	}
	return d
}

func sensitive(name string) bool {
	l := strings.ToLower(name)
	return strings.Contains(l, "password") || strings.Contains(l, "secret") || strings.Contains(l, "credential")
}

// ---- name helpers (mirror the Teams frontend) ----

var reservedNames = map[string]bool{
	"alias": true, "count": true, "depends_on": true, "for_each": true,
	"lifecycle": true, "provider": true, "provisioner": true, "connection": true,
	"id": true, "identity": true,
}

func tfName(field string) string {
	n := pascalToSnake(field)
	if reservedNames[n] {
		n += "_"
	}
	return n
}

func pascalToSnake(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		isUpper := r >= 'A' && r <= 'Z'
		if isUpper && i > 0 {
			prev := runes[i-1]
			prevLower := prev >= 'a' && prev <= 'z' || prev >= '0' && prev <= '9'
			nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
			if prevLower || nextLower {
				b.WriteByte('_')
			}
		}
		if isUpper {
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen-tf:", err)
		os.Exit(1)
	}
}
