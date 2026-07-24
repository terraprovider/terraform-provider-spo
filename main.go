// terraform-provider-spo is a Terraform/OpenTofu provider for SharePoint Online
// tenant/site admin configuration, generated from the SharePoint Online Management
// Shell CSOM metadata via go-spo.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/terraprovider/terraform-provider-spo/internal/provider"
)

//go:generate go run ./cmd/gen-tf

// version is set by goreleaser at build time.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/terraprovider/spo",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
