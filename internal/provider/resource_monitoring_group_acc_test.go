package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccMonitoringGroupResource covers create, in-place min_happy update, a
// name-only rename (rename guard), and import round-trip. check_ids is
// config-authoritative (not read back) so it is ignored on import.
func TestAccMonitoringGroupResource(t *testing.T) {
	name := accName("group")
	renamed := accName("group")
	const addr = "mtncloud_monitoring_group.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // create
				Config: testAccMonitoringGroupConfig(name, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "min_happy", "1"),
					resource.TestCheckResourceAttrSet(addr, "id"),
				),
			},
			{ // change min_happy in place
				Config: testAccMonitoringGroupConfig(name, 2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "min_happy", "2"),
			},
			{ // rename: must update in place, never replace
				Config: testAccMonitoringGroupConfig(renamed, 2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "name", renamed),
			},
			{ // import round-trip
				ResourceName:      addr,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccMonitoringGroupConfig(name string, minHappy int) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "mtncloud_monitoring_group" "test" {
  name      = %q
  min_happy = %d
  severity  = "warning"
}
`, name, minHappy)
}
