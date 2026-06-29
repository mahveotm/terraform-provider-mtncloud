package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccWorkflowResource creates a task + operational workflow that references
// it, then renames the workflow and asserts an in-place update (rename guard).
// NOTE: this is also the first run that confirms POST /task-sets succeeds for the
// token (the access map shows workflows=full but a legacy TaskSet perm=none).
func TestAccWorkflowResource(t *testing.T) {
	name := accName("wf")
	renamed := accName("wf")
	const addr = "mtncloud_workflow.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // create
				Config: testAccWorkflowConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "type", "operation"),
					resource.TestCheckResourceAttr(addr, "task.#", "1"),
					resource.TestCheckResourceAttrSet(addr, "id"),
				),
			},
			{ // rename: must update in place, never replace
				Config: testAccWorkflowConfig(renamed),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "name", renamed),
			},
			{ // import round-trip (members + metadata are read back)
				ResourceName:      addr,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccWorkflowConfig(name string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "mtncloud_task" "wf_member" {
  name        = "%s-task"
  type        = "shell"
  source_type = "local"
  content     = "tf-acc wf member content"
}

resource "mtncloud_workflow" "test" {
  name = %q
  type = "operation"

  task {
    task_id = mtncloud_task.wf_member.id
    phase   = "operation"
  }
}
`, name, name)
}
