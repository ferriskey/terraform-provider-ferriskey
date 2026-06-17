// terraform-provider-ferriskey is a Terraform provider that manages the
// configuration of a FerrisKey CIAM/IAM instance through its REST API.
//
// It is not a deployment tool for FerrisKey itself (that remains the role of
// the Helm chart): it is an HTTP client of the FerrisKey REST API exposed as
// declarative Terraform resources, in the spirit of the Keycloak and Auth0
// providers.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/provider"
)

// version is set by the linker at release time (see .goreleaser.yml). It is
// surfaced to Terraform so the value shows up in user-agent strings and logs.
var version = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// The address must match the source declared in user configuration:
		//   terraform { required_providers { ferriskey = { source = "ferriskey/ferriskey" } } }
		Address: "registry.terraform.io/ferriskey/ferriskey",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
