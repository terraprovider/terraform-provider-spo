package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The acceptance tests below exercise the full create → update → destroy lifecycle
// of each method-CRUD resource against a live tenant. The framework destroys the
// object at the end of each case. Run with TF_ACC=1 + SPO_TENANT_NAME + a
// certificate/OIDC credential pointing at a disposable dev tenant.

func TestAccSPOSiteScript_basic(t *testing.T) {
	title := acctest.RandomWithPrefix("tf-acc-script")
	const rn = "spo_site_script.test"
	content := `jsonencode({ "$schema" = "schema.json", actions = [{ verb = "applyTheme", themeName = "Blue" }], bindata = {}, version = 1 })`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "spo_site_script" "test" {
  title   = %q
  content = %s
}`, title, content),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "title", title),
					resource.TestCheckResourceAttrSet(rn, "id"),
				),
			},
			{
				Config: fmt.Sprintf(`resource "spo_site_script" "test" {
  title   = "%s-v2"
  content = %s
}`, title, content),
				Check: resource.TestCheckResourceAttr(rn, "title", title+"-v2"),
			},
		},
	})
}

func TestAccSPOSiteDesign_basic(t *testing.T) {
	title := acctest.RandomWithPrefix("tf-acc-design")
	const rn = "spo_site_design.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "spo_site_design" "test" {
  title        = %q
  web_template = "64"
}`, title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "title", title),
					resource.TestCheckResourceAttr(rn, "web_template", "64"),
					resource.TestCheckResourceAttrSet(rn, "id"),
				),
			},
			{
				Config: fmt.Sprintf(`resource "spo_site_design" "test" {
  title        = "%s-v2"
  web_template = "64"
}`, title),
				Check: resource.TestCheckResourceAttr(rn, "title", title+"-v2"),
			},
		},
	})
}

func TestAccSPOTheme_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-theme")
	const rn = "spo_theme.test"
	cfg := func(primary string) string {
		return fmt.Sprintf(`resource "spo_theme" "test" {
  identity   = %q
  theme_json = jsonencode({ palette = { themePrimary = %q } })
}`, name, primary)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("#0078d4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "identity", name),
					resource.TestCheckResourceAttr(rn, "id", name),
				),
			},
			{Config: cfg("#e81123")}, // in-place update (delete + re-add)
		},
	})
}

// TestAccSPOHubSite_basic promotes an existing site to a hub. It needs a disposable
// site URL in SPO_ACC_SITE_URL; unset -> skipped.
func TestAccSPOHubSite_basic(t *testing.T) {
	url := os.Getenv("SPO_ACC_SITE_URL")
	if url == "" {
		t.Skip("set SPO_ACC_SITE_URL to an existing site URL to run the hub-site test")
	}
	const rn = "spo_hub_site.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "spo_hub_site" "test" {
  identity = %q
  title    = "tf-acc-hub"
}`, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "identity", url),
					resource.TestCheckResourceAttrSet(rn, "hub_site_id"),
				),
			},
			{
				Config: fmt.Sprintf(`resource "spo_hub_site" "test" {
  identity = %q
  title    = "tf-acc-hub-v2"
}`, url),
				Check: resource.TestCheckResourceAttr(rn, "title", "tf-acc-hub-v2"),
			},
		},
	})
}

// TestAccSPOListDesign_basic creates a list design over a throwaway site script.
// List designs are immutable, so there is a single create step (any change would
// force replacement). The site script must carry list-scoped actions (createSPList).
func TestAccSPOListDesign_basic(t *testing.T) {
	title := acctest.RandomWithPrefix("tf-acc-listdesign")
	const rn = "spo_list_design.test"
	content := `jsonencode({ "$schema" = "schema.json", actions = [{ verb = "createSPList", listName = "tf-acc-list", templateType = 100, subactions = [{ verb = "setDescription", description = "acc" }] }], bindata = {}, version = 1 })`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "spo_site_script" "ls" {
  title   = "%s-script"
  content = %s
}
resource "spo_list_design" "test" {
  title        = %q
  description  = "tf-acc list design"
  site_scripts = [spo_site_script.ls.id]
}`, title, content, title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "title", title),
					resource.TestCheckResourceAttr(rn, "site_scripts.#", "1"),
					resource.TestCheckResourceAttrSet(rn, "id"),
				),
			},
		},
	})
}

// TestAccSPOTenantDataSource reads the tenant singleton through its data source
// (read-only; no mutation).
func TestAccSPOTenantDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "spo_tenant" "current" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.spo_tenant.current", "sharing_capability"),
			},
		},
	})
}
