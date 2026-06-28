package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccSecurityGroupRule_ethertypeAndDefaultedDestPort locks in the fix for
// "Provider produced inconsistent result after apply": the rule is created with
// ethertype set but destination_port_range omitted. The API echoes ethertype back
// only sometimes and defaults destination_port_range to "*", so the provider must
// keep the configured ethertype and accept the API-defaulted destination port
// without the post-apply plan diverging from the config.
func TestAccSecurityGroupRule_ethertypeAndDefaultedDestPort(t *testing.T) {
	name := fmt.Sprintf("tf-acc-sgr-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupRuleConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// ethertype must survive even though the API may omit it.
					resource.TestCheckResourceAttr("mtncloud_security_group_rule.test", "ethertype", "IPv4"),
					// destination_port_range was never configured; the API defaults it.
					// It must be known (Computed) after apply, not null.
					resource.TestCheckResourceAttrSet("mtncloud_security_group_rule.test", "destination_port_range"),
				),
				// The framework runs a plan after apply; an empty plan here is the
				// core regression guard against the previous inconsistent-result bug.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				// Re-applying the same config must be a no-op (no perpetual diff).
				Config: testAccSecurityGroupRuleConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func testAccSecurityGroupRuleConfig(name string) string {
	return fmt.Sprintf(`
resource "mtncloud_security_group" "test" {
  name        = %[1]q
  description = "terraform-provider-mtncloud acceptance test"
}

resource "mtncloud_security_group_rule" "test" {
  security_group_id = mtncloud_security_group.test.id

  direction   = "ingress"
  protocol    = "tcp"
  port_range  = "22"
  ethertype   = "IPv4"
  source_type = "cidr"
  source      = "0.0.0.0/0"
}
`, name)
}
