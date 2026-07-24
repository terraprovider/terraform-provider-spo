package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories serves the spo provider to acceptance tests. The
// provider is configured from the environment (SPO_TENANT_NAME / SPO_ADMIN_URL +
// ARM_*/AZURE_* credentials), so acceptance configs need no provider block.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"spo": providerserver.NewProtocol6WithError(New("acc")()),
}

// testAccPreCheck skips acceptance tests unless a SharePoint tenant + a credential
// are configured. These tests create and destroy real objects and must run against
// a disposable dev tenant (never one with real data). SharePoint rejects
// client-secret app-only tokens — use a certificate or OIDC federation.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("SPO_TENANT_NAME") == "" && os.Getenv("SPO_ADMIN_URL") == "" {
		t.Skip("set SPO_TENANT_NAME (or SPO_ADMIN_URL) to run acceptance tests")
	}
	if os.Getenv("ARM_TENANT_ID") == "" || os.Getenv("ARM_CLIENT_ID") == "" {
		t.Skip("set ARM_TENANT_ID, ARM_CLIENT_ID and a certificate/OIDC credential")
	}
	hasCred := os.Getenv("ARM_CLIENT_CERTIFICATE") != "" ||
		os.Getenv("ARM_CLIENT_CERTIFICATE_PATH") != "" ||
		os.Getenv("ARM_USE_OIDC") != "" ||
		os.Getenv("ARM_OIDC_TOKEN") != ""
	if !hasCred {
		t.Skip("no cert/OIDC credential set (SharePoint rejects client secrets for app-only)")
	}
}

func TestProviderSchemaValid(t *testing.T) {
	ctx := context.Background()
	srv := providerserver.NewProtocol6(New("test")())()
	resp, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Diagnostics) > 0 {
		for _, d := range resp.Diagnostics {
			t.Errorf("schema diag: %s — %s", d.Summary, d.Detail)
		}
	}
	for _, name := range []string{"spo_tenant", "spo_site"} {
		r, ok := resp.ResourceSchemas[name]
		if !ok {
			t.Fatalf("%s resource missing; have %v", name, keys(resp.ResourceSchemas))
		}
		t.Logf("%s schema OK: %d attributes", name, len(r.Block.Attributes))
		if _, ok := resp.DataSourceSchemas[name]; !ok {
			t.Errorf("%s data source missing", name)
		}
	}
}
func keys[V any](m map[string]V) []string {
	var o []string
	for k := range m {
		o = append(o, k)
	}
	return o
}
