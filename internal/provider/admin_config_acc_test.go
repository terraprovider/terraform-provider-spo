package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSPOTenantCdn adopts the Public CDN config and sets it to disabled (a no-op
// on most tenants), then a clean re-plan. Config resource → destroy is a no-op.
func TestAccSPOTenantCdn(t *testing.T) {
	const rn = "spo_tenant_cdn.public"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "spo_tenant_cdn" "public" {
  identity = "Public"
  enabled  = false
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "identity", "Public"),
					resource.TestCheckResourceAttr(rn, "enabled", "false"),
				),
			},
		},
	})
}

// TestAccSPOKnowledgeHubSite needs a disposable site URL (SPO_ACC_SITE_URL); unset -> skip.
func TestAccSPOKnowledgeHubSite(t *testing.T) {
	url := os.Getenv("SPO_ACC_SITE_URL")
	if url == "" {
		t.Skip("set SPO_ACC_SITE_URL to run the knowledge-hub test")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "spo_knowledge_hub_site" "this" {
  url = %q
}`, url),
				Check: resource.TestCheckResourceAttr("spo_knowledge_hub_site.this", "url", url),
			},
		},
	})
}

// TestAccSPORestrictedSearch needs a SharePoint Premium license (SPO_ACC_RESTRICTED_SEARCH=1); unset -> skip.
func TestAccSPORestrictedSearch(t *testing.T) {
	if os.Getenv("SPO_ACC_RESTRICTED_SEARCH") == "" {
		t.Skip("set SPO_ACC_RESTRICTED_SEARCH=1 (requires a SharePoint Premium license) to run this test")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "spo_restricted_search" "this" { mode = "Disabled" }`,
				Check:  resource.TestCheckResourceAttr("spo_restricted_search.this", "mode", "Disabled"),
			},
		},
	})
}

// TestAccSPOOrgNewsSite needs an org-news-eligible communication site
// (SPO_ACC_ORGNEWS_URL); unset -> skip.
func TestAccSPOOrgNewsSite(t *testing.T) {
	url := os.Getenv("SPO_ACC_ORGNEWS_URL")
	if url == "" {
		t.Skip("set SPO_ACC_ORGNEWS_URL to an org-news-eligible communication site")
	}
	const rn = "spo_org_news_site.this"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "spo_org_news_site" "this" { identity = %q }`, url),
				Check:  resource.TestCheckResourceAttr(rn, "identity", url),
			},
		},
	})
}

// TestAccSPOHomeSite needs a disposable site URL (SPO_ACC_SITE_URL); unset -> skip.
func TestAccSPOHomeSite(t *testing.T) {
	url := os.Getenv("SPO_ACC_SITE_URL")
	if url == "" {
		t.Skip("set SPO_ACC_SITE_URL to run the home-site test")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "spo_home_site" "this" { url = %q }`, url),
				Check:  resource.TestCheckResourceAttr("spo_home_site.this", "url", url),
			},
		},
	})
}

// TestAccSPOBlockedPageContentType blocks a content type. Needs a blockable type in
// SPO_ACC_BLOCKED_TYPE (e.g. ClientSidePage); unset -> skip.
func TestAccSPOBlockedPageContentType(t *testing.T) {
	ct := os.Getenv("SPO_ACC_BLOCKED_TYPE")
	if ct == "" {
		t.Skip("set SPO_ACC_BLOCKED_TYPE (a blockable content type) to run this test")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "spo_blocked_page_content_type" "this" { identity = %q }`, ct),
				Check:  resource.TestCheckResourceAttr("spo_blocked_page_content_type.this", "identity", ct),
			},
		},
	})
}

// TestAccSPOStorageEntity sets a tenant storage entity on the app catalog site.
// Needs the tenant app catalog URL (SPO_ACC_APPCATALOG_URL) — storage entities can
// only be written on the app catalog web; unset -> skip.
func TestAccSPOStorageEntity(t *testing.T) {
	appcat := os.Getenv("SPO_ACC_APPCATALOG_URL")
	if appcat == "" {
		t.Skip("set SPO_ACC_APPCATALOG_URL (the tenant app catalog site URL) to run the storage-entity test")
	}
	id := appcat + "|" + acctest.RandomWithPrefix("tfAccKey")
	const rn = "spo_storage_entity.test"
	cfg := func(v string) string {
		return fmt.Sprintf(`resource "spo_storage_entity" "test" {
  identity    = %q
  value       = %q
  description = "tf-acc"
  comment     = "tf-acc"
}`, id, v)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("v1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "identity", id),
					resource.TestCheckResourceAttr(rn, "value", "v1"),
				),
			},
			{Config: cfg("v2"), Check: resource.TestCheckResourceAttr(rn, "value", "v2")}, // update in place
		},
	})
}

// TestAccSPOOrgAssetsLibrary registers a document library as an org-assets library.
// Needs a library URL (SPO_ACC_ORGASSETS_LIB_URL); the tenant Office 365 CDN must be
// enabled for the chosen type or the add is rejected. Unset -> skip.
func TestAccSPOOrgAssetsLibrary(t *testing.T) {
	lib := os.Getenv("SPO_ACC_ORGASSETS_LIB_URL")
	if lib == "" {
		t.Skip("set SPO_ACC_ORGASSETS_LIB_URL (a document library URL; the tenant CDN must be enabled) to run the org-assets test")
	}
	const rn = "spo_org_assets_library.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "spo_org_assets_library" "test" {
  identity       = %q
  org_asset_type = "ImageDocumentLibrary"
}`, lib),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "identity", lib),
					resource.TestCheckResourceAttr(rn, "org_asset_type", "ImageDocumentLibrary"),
				),
			},
		},
	})
}

// TestAccSPOSiteDesignRights creates a throwaway site design and grants View rights
// to a principal (SPO_ACC_PRINCIPAL, a user/group login name); unset -> skip.
func TestAccSPOSiteDesignRights(t *testing.T) {
	principal := os.Getenv("SPO_ACC_PRINCIPAL")
	if principal == "" {
		t.Skip("set SPO_ACC_PRINCIPAL (a user/group login name) to run the site-design-rights test")
	}
	title := acctest.RandomWithPrefix("tf-acc-rights")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "spo_site_design" "d" {
  title        = %q
  web_template = "64"
}
resource "spo_site_design_rights" "r" {
  identity   = spo_site_design.d.id
  principals = [%q]
}`, title, principal),
				Check: resource.TestCheckResourceAttr("spo_site_design_rights.r", "principals.#", "1"),
			},
		},
	})
}
