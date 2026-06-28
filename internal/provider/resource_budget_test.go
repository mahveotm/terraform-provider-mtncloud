package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccBudget_inPlaceUpdate locks in the fix for "Invalid Budget ID:
// Could not parse \"\" as a numeric ID". A Computed id without UseStateForUnknown
// plans as "known after apply" during an in-place update, so plan.ID was unknown
// and Update failed. It also covers the read-only currency: the budget API
// ignores any requested currency and reports the account currency, so currency
// is Computed-only and must round-trip without a perpetual diff.
func TestAccBudget_inPlaceUpdate(t *testing.T) {
	name := fmt.Sprintf("tf-acc-budget-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBudgetConfig(name, "1000000"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("mtncloud_budget.test", "id"),
					// currency is API-controlled; it must be known (not null) after apply.
					resource.TestCheckResourceAttrSet("mtncloud_budget.test", "currency"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				// In-place update (name + cost). Previously failed parsing plan.ID.
				Config: testAccBudgetConfig(name+"-renamed", "2500000"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("mtncloud_budget.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func testAccBudgetConfig(name, cost string) string {
	return fmt.Sprintf(`
resource "mtncloud_budget" "test" {
  name     = %[1]q
  interval = "year"
  year     = "2026"
  costs    = [%[2]s]
}
`, name, cost)
}
