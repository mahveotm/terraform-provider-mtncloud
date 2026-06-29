package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccTaskResource exercises create, in-place content update, and — the key
// safety check — a name-only change that must update in place (rename guard, never
// destroy/recreate). Import only restores stable metadata; script content and
// type-specific options are config-authoritative, so they're ignored on verify.
func TestAccTaskResource(t *testing.T) {
	name := accName("task")
	renamed := accName("task")
	const addr = "mtncloud_task.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // create
				Config: testAccTaskConfig(name, "tf-acc script content one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "type", "shell"),
					resource.TestCheckResourceAttrSet(addr, "id"),
				),
			},
			{ // update content in place
				Config: testAccTaskConfig(name, "tf-acc script content two"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
			},
			{ // rename: must be an in-place update, never a replacement
				Config: testAccTaskConfig(renamed, "tf-acc script content two"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "name", renamed),
			},
			{ // changing task type is intentionally immutable and must replace
				Config:             testAccTaskConfigTyped(renamed, "python", "tf-acc script content two"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionReplace),
					},
				},
			},
			{ // import: metadata round-trips; content/source are config-authoritative
				ResourceName:            addr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"source_type", "content", "sudo", "password"},
			},
		},
	})
}

func testAccTaskConfig(name, content string) string {
	return testAccTaskConfigTyped(name, "shell", content)
}

func testAccTaskConfigTyped(name, taskType, content string) string {
	sudo := ""
	if taskType == "shell" {
		sudo = "  sudo           = true\n"
	}
	return testAccProviderConfig + fmt.Sprintf(`
resource "mtncloud_task" "test" {
  name           = %q
  type           = %q
  source_type    = "local"
  content        = %q
  execute_target = "resource"
%s}
`, name, taskType, content, sudo)
}
