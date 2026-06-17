package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories wires the in-process provider for acceptance
// tests. The provider is configured from FERRISKEY_* environment variables.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"ferriskey": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck validates that the environment is configured for acceptance
// testing against a live FerrisKey instance. Acceptance tests only run when
// TF_ACC is set.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("FERRISKEY_URL") == "" {
		t.Fatal("FERRISKEY_URL must be set for acceptance tests")
	}
	if os.Getenv("FERRISKEY_REALM") == "" {
		t.Fatal("FERRISKEY_REALM must be set for acceptance tests")
	}
	if os.Getenv("FERRISKEY_CLIENT_ID") == "" {
		t.Fatal("FERRISKEY_CLIENT_ID must be set for acceptance tests")
	}
	// One of password or client_secret must be present.
	if os.Getenv("FERRISKEY_PASSWORD") == "" && os.Getenv("FERRISKEY_CLIENT_SECRET") == "" {
		t.Fatal("either FERRISKEY_PASSWORD (password grant) or FERRISKEY_CLIENT_SECRET (client credentials) must be set")
	}
}
