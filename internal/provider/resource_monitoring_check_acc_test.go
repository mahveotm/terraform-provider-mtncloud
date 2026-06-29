package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccMonitoringCheckResource covers create, in-place description update, a
// name-only rename (rename guard), and import round-trip. config is
// config-authoritative (not read back) so it is ignored on import.
//
// NOTE: webGetCheck is documented in the API; if the live token cannot create it,
// swap check_type/config for a creatable type discovered during the Step 0 probe.
func TestAccMonitoringCheckResource(t *testing.T) {
	name := accName("check")
	renamed := accName("check")
	const addr = "mtncloud_monitoring_check.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // create
				Config: testAccMonitoringCheckConfig(name, "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "check_type", "webGetCheck"),
					resource.TestCheckResourceAttr(addr, "description", "first"),
					resource.TestCheckResourceAttrSet(addr, "id"),
				),
			},
			{ // change description in place
				Config: testAccMonitoringCheckConfig(name, "second"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "description", "second"),
			},
			{ // rename: must update in place, never replace
				Config: testAccMonitoringCheckConfig(renamed, "second"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "name", renamed),
			},
			{ // import round-trip; config is write-through (config-authoritative)
				ResourceName:            addr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config"},
			},
		},
	})
}

func testAccMonitoringCheckConfig(name, description string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "mtncloud_monitoring_check" "test" {
  name        = %q
  check_type  = "webGetCheck"
  description = %q
  severity    = "warning"
  config      = jsonencode({ webUrl = "https://www.mtn.ng" })
}
`, name, description)
}
