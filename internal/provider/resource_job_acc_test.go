package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccJobResource creates a task -> workflow -> manual job chain, then renames
// the job and asserts an in-place update (rename guard). target_type values for
// instance targets are confirmed separately on the first live run.
func TestAccJobResource(t *testing.T) {
	name := accName("job")
	renamed := accName("job")
	const addr = "mtncloud_job.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // create
				Config: testAccJobConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "schedule_mode", "manual"),
					resource.TestCheckResourceAttrSet(addr, "workflow_id"),
					resource.TestCheckResourceAttrSet(addr, "id"),
				),
			},
			{ // rename: must update in place, never replace
				Config: testAccJobConfig(renamed),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "name", renamed),
			},
			{ // switching a job from workflow-backed to task-backed changes kind and must replace
				Config:             testAccJobTaskConfig(renamed),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionReplace),
					},
				},
			},
			{ // import: id/name/enabled/workflow_id/target_type round-trip; schedule_mode and instance_label are config-authoritative
				ResourceName:            addr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"schedule_mode", "instance_label"},
			},
		},
	})
}

func testAccJobConfig(name string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "mtncloud_task" "job_member" {
  name        = "%s-task"
  type        = "shell"
  source_type = "local"
  content     = "tf-acc job member content"
}

resource "mtncloud_workflow" "job_wf" {
  name = "%s-wf"
  type = "operation"

  task {
    task_id = mtncloud_task.job_member.id
    phase   = "operation"
  }
}

resource "mtncloud_job" "test" {
  name           = %q
  workflow_id    = mtncloud_workflow.job_wf.id
  schedule_mode  = "manual"
  target_type    = "instance-label"
  instance_label = "tf-acc"
}
`, name, name, name)
}

func testAccJobTaskConfig(name string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "mtncloud_task" "job_member" {
  name        = "%s-task"
  type        = "shell"
  source_type = "local"
  content     = "tf-acc job member content"
}

resource "mtncloud_job" "test" {
  name           = %q
  task_id        = mtncloud_task.job_member.id
  schedule_mode  = "manual"
  target_type    = "instance-label"
  instance_label = "tf-acc"
}
`, name, name)
}
