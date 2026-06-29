package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

// TestMain enables `go test -sweep=...` for this package while leaving normal
// `go test` runs unaffected.
func TestMain(m *testing.M) {
	resource.TestMain(m)
}

// registerSweeper wires a generic name-prefix sweeper: it lists a resource type,
// keeps the ones isSweepable() recognizes (tf-acc-*/tf-smoke-*), and deletes them.
// This is the single place sweeper logic lives; each resource supplies only its
// list/id/name/delete accessors.
func registerSweeper[T any](
	name string,
	list func(*client.Client, context.Context) ([]T, error),
	id func(T) int64,
	objName func(T) string,
	del func(*client.Client, context.Context, int64) error,
) {
	resource.AddTestSweepers(name, &resource.Sweeper{
		Name: name,
		F: func(string) error {
			c, err := sweepClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			items, err := list(c, ctx)
			if err != nil {
				return err
			}
			for _, item := range items {
				if isSweepable(objName(item)) {
					_ = del(c, ctx, id(item))
				}
			}
			return nil
		},
	})
}

func init() {
	registerSweeper("mtncloud_credential", (*client.Client).ListCredentials,
		func(x client.Credential) int64 { return x.ID }, func(x client.Credential) string { return x.Name },
		(*client.Client).DeleteCredential)
	registerSweeper("mtncloud_network_domain", (*client.Client).ListNetworkDomains,
		func(x client.NetworkDomain) int64 { return x.ID }, func(x client.NetworkDomain) string { return x.Name },
		(*client.Client).DeleteNetworkDomain)
	registerSweeper("mtncloud_ipv4_ip_pool", (*client.Client).ListIPPools,
		func(x client.IPPool) int64 { return x.ID }, func(x client.IPPool) string { return x.Name },
		(*client.Client).DeleteIPPool)
	registerSweeper("mtncloud_scale_threshold", (*client.Client).ListScaleThresholds,
		func(x client.ScaleThreshold) int64 { return x.ID }, func(x client.ScaleThreshold) string { return x.Name },
		(*client.Client).DeleteScaleThreshold)
	registerSweeper("mtncloud_budget", (*client.Client).ListBudgets,
		func(x client.Budget) int64 { return x.ID }, func(x client.Budget) string { return x.Name },
		(*client.Client).DeleteBudget)
	registerSweeper("mtncloud_environment", (*client.Client).ListEnvironments,
		func(x client.Environment) int64 { return x.ID }, func(x client.Environment) string { return x.Name },
		(*client.Client).DeleteEnvironment)
	registerSweeper("mtncloud_wiki_page", (*client.Client).ListWikiPages,
		func(x client.WikiPage) int64 { return x.ID }, func(x client.WikiPage) string { return x.Name },
		(*client.Client).DeleteWikiPage)
	registerSweeper("mtncloud_key_pair", (*client.Client).ListKeyPairs,
		func(x client.KeyPair) int64 { return x.ID }, func(x client.KeyPair) string { return x.Name },
		(*client.Client).DeleteKeyPair)
	registerSweeper("mtncloud_task", (*client.Client).ListTasks,
		func(x client.Task) int64 { return x.ID }, func(x client.Task) string { return x.Name },
		(*client.Client).DeleteTask)
	registerSweeper("mtncloud_workflow", (*client.Client).ListWorkflows,
		func(x client.Workflow) int64 { return x.ID }, func(x client.Workflow) string { return x.Name },
		(*client.Client).DeleteWorkflow)
	registerSweeper("mtncloud_execute_schedule", (*client.Client).ListExecuteSchedules,
		func(x client.ExecuteSchedule) int64 { return x.ID }, func(x client.ExecuteSchedule) string { return x.Name },
		(*client.Client).DeleteExecuteSchedule)
	registerSweeper("mtncloud_job", (*client.Client).ListJobs,
		func(x client.Job) int64 { return x.ID }, func(x client.Job) string { return x.Name },
		(*client.Client).DeleteJob)
	registerSweeper("mtncloud_role", (*client.Client).ListRoles,
		func(x client.Role) int64 { return x.ID }, func(x client.Role) string { return x.Authority },
		(*client.Client).DeleteRole)
	registerSweeper("mtncloud_user", (*client.Client).ListUsers,
		func(x client.User) int64 { return x.ID }, func(x client.User) string { return x.Username },
		(*client.Client).DeleteUser)
	registerSweeper("mtncloud_user_group", (*client.Client).ListUserGroups,
		func(x client.UserGroup) int64 { return x.ID }, func(x client.UserGroup) string { return x.Name },
		(*client.Client).DeleteUserGroup)
	registerSweeper("mtncloud_contact", (*client.Client).ListContacts,
		func(x client.Contact) int64 { return x.ID }, func(x client.Contact) string { return x.Name },
		(*client.Client).DeleteContact)
	registerSweeper("mtncloud_monitoring_check", (*client.Client).ListChecks,
		func(x client.Check) int64 { return x.ID }, func(x client.Check) string { return x.Name },
		(*client.Client).DeleteCheck)
	registerSweeper("mtncloud_monitoring_group", (*client.Client).ListMonitoringGroups,
		func(x client.MonitoringGroup) int64 { return x.ID }, func(x client.MonitoringGroup) string { return x.Name },
		(*client.Client).DeleteMonitoringGroup)
	registerSweeper("mtncloud_monitoring_alert", (*client.Client).ListAlerts,
		func(x client.Alert) int64 { return x.ID }, func(x client.Alert) string { return x.Name },
		(*client.Client).DeleteAlert)
}
