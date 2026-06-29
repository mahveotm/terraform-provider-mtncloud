package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccMonitoringAlertResource covers create (wiring a contact created in the
// same config), in-place min_severity update, a name-only rename (rename guard),
// and import round-trip. The relational id sets are config-authoritative (not read
// back) so they are ignored on import.
func TestAccMonitoringAlertResource(t *testing.T) {
	name := accName("alert")
	renamed := accName("alert")
	contact := accName("alert-contact")
	const addr = "mtncloud_monitoring_alert.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // create
				Config: testAccMonitoringAlertConfig(name, contact, "warning"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "all_checks", "true"),
					resource.TestCheckResourceAttr(addr, "min_severity", "warning"),
					resource.TestCheckResourceAttrSet(addr, "id"),
				),
			},
			{ // change min_severity in place
				Config: testAccMonitoringAlertConfig(name, contact, "critical"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "min_severity", "critical"),
			},
			{ // rename: must update in place, never replace
				Config: testAccMonitoringAlertConfig(renamed, contact, "critical"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "name", renamed),
			},
			{ // import round-trip; relational id sets are config-authoritative
				ResourceName:            addr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"contact_ids", "check_ids", "group_ids", "app_ids"},
			},
		},
	})
}

func testAccMonitoringAlertConfig(name, contact, minSeverity string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "mtncloud_contact" "notify" {
  name          = %q
  email_address = "alert-oncall@mtn.ng"
}

resource "mtncloud_monitoring_alert" "test" {
  name         = %q
  all_checks   = true
  min_severity = %q
  contact_ids  = [tonumber(mtncloud_contact.notify.id)]
}
`, contact, name, minSeverity)
}
