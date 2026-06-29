package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &executeScheduleResource{}
var _ resource.ResourceWithConfigure = &executeScheduleResource{}
var _ resource.ResourceWithImportState = &executeScheduleResource{}

type executeScheduleResource struct {
	resourceBase
}

type executeScheduleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Cron        types.String `tfsdk:"cron"`
	Timezone    types.String `tfsdk:"timezone"`
	Visibility  types.String `tfsdk:"visibility"`
}

func NewExecuteScheduleResource() resource.Resource { return &executeScheduleResource{} }

func (r *executeScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_execute_schedule"
}

func (r *executeScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud execute schedule: a cron schedule that jobs can run on. " +
			"All attributes update in place — renaming or changing the cron never recreates the schedule.",
		Attributes: map[string]rschema.Attribute{
			"id":          computedIDAttribute("Numeric identifier of the schedule."),
			"name":        rschema.StringAttribute{Required: true, Description: "Name of the schedule."},
			"description": rschema.StringAttribute{Optional: true, Computed: true, Description: "Description of the schedule."},
			"enabled":     rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the schedule is enabled. Defaults to `true`."},
			"cron":        rschema.StringAttribute{Required: true, Description: "Cron expression, e.g. `0 0 * * *`."},
			"timezone":    rschema.StringAttribute{Required: true, Description: "Schedule timezone, e.g. `Africa/Lagos`."},
			"visibility": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.OneOf("private", "public")},
				Description: "Schedule visibility: `private` or `public`.",
			},
		},
	}
}

func (r *executeScheduleResource) input(plan executeScheduleResourceModel) client.ExecuteScheduleInput {
	return client.ExecuteScheduleInput{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Enabled:     boolPtr(plan.Enabled),
		Cron:        plan.Cron.ValueString(),
		Timezone:    plan.Timezone.ValueString(),
		Visibility:  plan.Visibility.ValueString(),
	}
}

func (r *executeScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan executeScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sched, err := r.client.CreateExecuteSchedule(ctx, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Execute Schedule", err)
		return
	}
	setExecuteScheduleState(&plan, sched)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *executeScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state executeScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Execute Schedule", &resp.Diagnostics)
	if !ok {
		return
	}
	sched, err := r.client.GetExecuteSchedule(ctx, id)
	if handleReadError(ctx, err, "Execute Schedule", &resp.State, &resp.Diagnostics) {
		return
	}
	setExecuteScheduleState(&state, sched)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *executeScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan executeScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Execute Schedule", &resp.Diagnostics)
	if !ok {
		return
	}
	sched, err := r.client.UpdateExecuteSchedule(ctx, id, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Execute Schedule", err)
		return
	}
	setExecuteScheduleState(&plan, sched)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *executeScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state executeScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Execute Schedule", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteExecuteSchedule(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Execute Schedule", err)
	}
}

func (r *executeScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func setExecuteScheduleState(data *executeScheduleResourceModel, sched *client.ExecuteSchedule) {
	data.ID = types.StringValue(strconv.FormatInt(sched.ID, 10))
	data.Name = types.StringValue(sched.Name)
	data.Description = mergeAPIString(data.Description, sched.Description)
	data.Enabled = mergeAPIBool(data.Enabled, sched.Enabled)
	if sched.Cron != "" {
		data.Cron = types.StringValue(sched.Cron)
	}
	if sched.Timezone != "" {
		data.Timezone = types.StringValue(sched.Timezone)
	}
	data.Visibility = mergeAPIString(data.Visibility, sched.Visibility)
}
