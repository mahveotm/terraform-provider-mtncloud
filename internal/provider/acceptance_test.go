package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories wires the in-process provider into the
// terraform-plugin-testing harness under the address used in main.go.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"mtncloud": func() (tfprotov6.ProviderServer, error) {
		return providerserver.NewProtocol6WithError(New("test")())()
	},
}

// testAccPreCheck fails fast when the live API credentials are not configured.
// Acceptance tests only run when TF_ACC is set; this keeps `go test` (unit only)
// green in environments without API access.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("MTN_CLOUD_TOKEN") == "" {
		t.Fatal("MTN_CLOUD_TOKEN must be set for acceptance tests")
	}
}
