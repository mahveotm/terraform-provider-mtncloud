package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &monitoringGroupResource{}
var _ resource.ResourceWithConfigure = &monitoringGroupResource{}
var _ resource.ResourceWithImportState = &monitoringGroupResource{}

type monitoringGroupResource struct {
	resourceBase
}

type monitoringGroupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	MinHappy    types.Int64  `tfsdk:"min_happy"`
	InUptime    types.Bool   `tfsdk:"in_uptime"`
	Severity    types.String `tfsdk:"severity"`
	Active      types.Bool   `tfsdk:"active"`
	CheckIDs    types.Set    `tfsdk:"check_ids"`
}

func NewMonitoringGroupResource() resource.Resource { return &monitoringGroupResource{} }

func (r *monitoringGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitoring_group"
}

func (r *monitoringGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud monitoring check group: several checks rolled up to one " +
			"health status (healthy while at least `min_happy` members are happy).",
		Attributes: map[string]rschema.Attribute{
			"id":          computedIDAttribute("Numeric identifier of the check group."),
			"name":        rschema.StringAttribute{Required: true, Description: "Name of the check group."},
			"description": rschema.StringAttribute{Optional: true, Computed: true, Description: "Description of the check group."},
			"min_happy":   rschema.Int64Attribute{Optional: true, Computed: true, Description: "Minimum number of member checks that must be happy to keep the group healthy."},
			"in_uptime":   rschema.BoolAttribute{Optional: true, Computed: true, Description: "Whether this group affects account-wide availability calculations."},
			"severity":    rschema.StringAttribute{Optional: true, Computed: true, Description: "Maximum severity this group can incur when failing (e.g. `info`, `warning`, `critical`)."},
			"active":      rschema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the check group is active."},
			"check_ids": rschema.SetAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
				Description: "IDs of the member checks. Config-authoritative: managed here, not reconciled from the API " +
					"(membership changed outside Terraform is not detected as drift).",
			},
		},
	}
}

func (r *monitoringGroupResource) input(ctx context.Context, plan monitoringGroupResourceModel) client.MonitoringGroupInput {
	return client.MonitoringGroupInput{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		MinHappy:    int64Ptr(plan.MinHappy),
		InUptime:    boolPtr(plan.InUptime),
		Severity:    plan.Severity.ValueString(),
		Active:      boolPtr(plan.Active),
		CheckIDs:    int64Set(ctx, plan.CheckIDs),
	}
}

func (r *monitoringGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitoringGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	group, err := r.client.CreateMonitoringGroup(ctx, r.input(ctx, plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Monitoring Group", err)
		return
	}
	setMonitoringGroupState(&plan, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitoringGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitoringGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Monitoring Group", &resp.Diagnostics)
	if !ok {
		return
	}
	group, err := r.client.GetMonitoringGroup(ctx, id)
	if handleReadError(ctx, err, "Monitoring Group", &resp.State, &resp.Diagnostics) {
		return
	}
	setMonitoringGroupState(&state, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *monitoringGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitoringGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Monitoring Group", &resp.Diagnostics)
	if !ok {
		return
	}
	group, err := r.client.UpdateMonitoringGroup(ctx, id, r.input(ctx, plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Monitoring Group", err)
		return
	}
	setMonitoringGroupState(&plan, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitoringGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitoringGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Monitoring Group", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteMonitoringGroup(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Monitoring Group", err)
	}
}

func (r *monitoringGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

// setMonitoringGroupState reconciles scalar fields; check_ids is
// config-authoritative and kept from prior state.
func setMonitoringGroupState(data *monitoringGroupResourceModel, group *client.MonitoringGroup) {
	data.ID = types.StringValue(strconv.FormatInt(group.ID, 10))
	data.Name = types.StringValue(group.Name)
	data.Description = mergeAPIString(data.Description, group.Description)
	data.MinHappy = mergeAPIInt64(data.MinHappy, group.MinHappy)
	data.InUptime = mergeAPIBool(data.InUptime, group.InUptime)
	data.Severity = mergeAPIString(data.Severity, group.Severity)
	data.Active = mergeAPIBool(data.Active, group.Active)
}
