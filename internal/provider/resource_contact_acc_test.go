package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccContactResource covers create, in-place email update, a name-only rename
// (must update in place — rename guard), and import round-trip.
func TestAccContactResource(t *testing.T) {
	name := accName("contact")
	renamed := accName("contact")
	const addr = "mtncloud_contact.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // create
				Config: testAccContactConfig(name, "oncall@mtn.ng"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "email_address", "oncall@mtn.ng"),
					resource.TestCheckResourceAttrSet(addr, "id"),
				),
			},
			{ // change email in place
				Config: testAccContactConfig(name, "ops@mtn.ng"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "email_address", "ops@mtn.ng"),
			},
			{ // rename: must update in place, never replace
				Config: testAccContactConfig(renamed, "ops@mtn.ng"),
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

func testAccContactConfig(name, email string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "mtncloud_contact" "test" {
  name          = %q
  email_address = %q
}
`, name, email)
}
