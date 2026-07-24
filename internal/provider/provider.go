package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/terraprovider/go-spo/spo"
	"github.com/terraprovider/go-spo/spoapi"
	"github.com/terraprovider/terraform-provider-spo/internal/clients"
	"github.com/terraprovider/tf-msadmin/authschema"
)

// spoProvider implements the SharePoint Online admin provider.
type spoProvider struct{ version string }

// New returns the provider constructor for the given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &spoProvider{version: version} }
}

func (p *spoProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "spo"
	resp.Version = p.version
}

// providerModel is the shared azuread/azurerm-aligned auth block plus the
// SharePoint admin endpoint (which is also the token audience).
type providerModel struct {
	authschema.Model
	TenantName types.String `tfsdk:"tenant_name"`
	AdminURL   types.String `tfsdk:"admin_url"`
}

func (p *spoProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	attrs := authschema.Attributes()
	attrs["tenant_name"] = schema.StringAttribute{
		Optional:    true,
		Description: "The tenant name — the '{tenant}' in https://{tenant}-admin.sharepoint.com. Env: SPO_TENANT_NAME. Ignored if admin_url is set.",
	}
	attrs["admin_url"] = schema.StringAttribute{
		Optional:    true,
		Description: "The SharePoint admin URL, e.g. https://contoso-admin.sharepoint.com (required for national clouds). Env: SPO_ADMIN_URL. Overrides tenant_name.",
	}
	resp.Schema = schema.Schema{
		Description: "Manage SharePoint Online tenant/site admin configuration via the CSOM admin API. Authentication mirrors the AzureAD/AzureRM providers (ARM_*/AZURE_* env vars). App-only needs SharePoint Sites.FullControl.All AND a certificate credential — client-secret app-only auth is NOT supported by SharePoint (its CSOM endpoint rejects secret-based app tokens with '401 Unsupported app only token', before any permission check). Use client_certificate / client_certificate_path.",
		Attributes:  attrs,
	}
}

func (p *spoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var m providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve the SharePoint admin URL (also the token audience).
	adminURL := firstNonEmpty(m.AdminURL.ValueString(), os.Getenv("SPO_ADMIN_URL"))
	tenantName := firstNonEmpty(m.TenantName.ValueString(), os.Getenv("SPO_TENANT_NAME"))
	if adminURL == "" && tenantName == "" {
		resp.Diagnostics.AddError(
			"SharePoint admin endpoint required",
			"Set `admin_url` (e.g. https://contoso-admin.sharepoint.com) or `tenant_name` on the provider, or SPO_ADMIN_URL / SPO_TENANT_NAME.",
		)
		return
	}
	if adminURL == "" {
		adminURL = "https://" + tenantName + "-admin." + spoapi.Commercial.SPOSuffix
	}
	adminURL = strings.TrimRight(adminURL, "/")

	// Explicit config overlaid onto the ARM_*/AZURE_* environment.
	tp, err := m.Config().Build()
	if err != nil {
		resp.Diagnostics.AddError("Authentication configuration error", err.Error())
		return
	}

	api, err := spoapi.New(spoapi.Options{
		AdminURL: adminURL,
		Tokens:   tp,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client initialisation error", err.Error())
		return
	}

	c := &clients.Client{API: api, SPO: spo.New(api), AdminURL: adminURL}
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *spoProvider) Resources(_ context.Context) []func() resource.Resource {
	return generatedResources()
}

func (p *spoProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return generatedDataSources()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
