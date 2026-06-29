package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccExecuteScheduleResource covers create, in-place cron update, a name-only
// rename (must update in place — rename guard), and import round-trip.
func TestAccExecuteScheduleResource(t *testing.T) {
	name := accName("sched")
	renamed := accName("sched")
	const addr = "mtncloud_execute_schedule.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // create
				Config: testAccExecuteScheduleConfig(name, "0 2 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "cron", "0 2 * * *"),
					resource.TestCheckResourceAttr(addr, "enabled", "true"),
					resource.TestCheckResourceAttrSet(addr, "id"),
				),
			},
			{ // change cron in place
				Config: testAccExecuteScheduleConfig(name, "0 3 * * *"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "cron", "0 3 * * *"),
			},
			{ // rename: must update in place, never replace
				Config: testAccExecuteScheduleConfig(renamed, "0 3 * * *"),
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

func testAccExecuteScheduleConfig(name, cron string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "mtncloud_execute_schedule" "test" {
  name     = %q
  cron     = %q
  timezone = "Africa/Lagos"
}
`, name, cron)
}
