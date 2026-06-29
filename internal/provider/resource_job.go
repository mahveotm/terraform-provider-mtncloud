package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &jobResource{}
var _ resource.ResourceWithConfigure = &jobResource{}
var _ resource.ResourceWithImportState = &jobResource{}
var _ resource.ResourceWithValidateConfig = &jobResource{}

type jobResource struct {
	resourceBase
}

type jobResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Labels            types.List   `tfsdk:"labels"`
	LabelsAll         types.List   `tfsdk:"labels_all"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	WorkflowID        types.Int64  `tfsdk:"workflow_id"`
	TaskID            types.Int64  `tfsdk:"task_id"`
	ScheduleMode      types.String `tfsdk:"schedule_mode"`
	DateTime          types.String `tfsdk:"date_time"`
	ExecuteScheduleID types.Int64  `tfsdk:"execute_schedule_id"`
	TargetType        types.String `tfsdk:"target_type"`
	Targets           types.List   `tfsdk:"targets"`
	InstanceLabel     types.String `tfsdk:"instance_label"`
	CustomOptions     types.Map    `tfsdk:"custom_options"`
	CustomConfig      types.String `tfsdk:"custom_config"`
}

func NewJobResource() resource.Resource { return &jobResource{} }

// requiresReplaceOnPresenceChange forces replacement only when an attribute
// toggles between set and unset — used so switching a job's kind (workflow_id
// <-> task_id) recreates it, while changing the referenced id updates in place.
func requiresReplaceOnPresenceChange() planmodifier.Int64 {
	return int64planmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.Int64Request, resp *int64planmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = req.StateValue.IsNull() != req.PlanValue.IsNull()
		},
		"Replace when the job switches between a workflow and a task.",
		"Replace when the job switches between a workflow and a task.",
	)
}

func (r *jobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (r *jobResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	presenceReplace := []planmodifier.Int64{requiresReplaceOnPresenceChange()}
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud job: runs a workflow or a task on a schedule against targets. " +
			"Exactly one of `workflow_id` or `task_id` must be set. Server targets are unsupported on MTN " +
			"(the /servers API is restricted); use instance or instance-label targets.",
		Attributes: map[string]rschema.Attribute{
			"id":   computedIDAttribute("Numeric identifier of the job."),
			"name": rschema.StringAttribute{Required: true, Description: "Name of the job."},
			"labels": rschema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Labels applied to the job. Merged with the provider's default_labels into `labels_all`.",
			},
			"labels_all": rschema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Effective labels: the provider's default_labels merged (union) with `labels`.",
			},
			"enabled": rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the job is enabled. Defaults to `true`."},
			"workflow_id": rschema.Int64Attribute{
				Optional:      true,
				PlanModifiers: presenceReplace,
				Description:   "ID of the workflow to run. Mutually exclusive with `task_id`.",
			},
			"task_id": rschema.Int64Attribute{
				Optional:      true,
				PlanModifiers: presenceReplace,
				Description:   "ID of the task to run. Mutually exclusive with `workflow_id`.",
			},
			"schedule_mode": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("manual"),
				Validators:  []validator.String{stringvalidator.OneOf(client.JobScheduleModes...)},
				Description: "When the job runs: `manual`, `date_time` (set `date_time`), or `schedule` (set `execute_schedule_id`).",
			},
			"date_time":           rschema.StringAttribute{Optional: true, Description: "RFC3339 timestamp for a one-off run (when `schedule_mode = date_time`)."},
			"execute_schedule_id": rschema.Int64Attribute{Optional: true, Description: "Execute schedule ID (when `schedule_mode = schedule`)."},
			"target_type": rschema.StringAttribute{
				Required:    true,
				Validators:  []validator.String{stringvalidator.OneOf(client.JobTargetTypes...)},
				Description: "Execution target type (required by the API): " + joinQuoted(client.JobTargetTypes) + ".",
			},
			"targets": rschema.ListAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
				Description: "Instance IDs to target (when `target_type = instance`).",
			},
			"instance_label": rschema.StringAttribute{Optional: true, Description: "Instance label to target (when `target_type = instance-label`)."},
			"custom_options": rschema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Custom option key/values passed to a workflow job.",
			},
			"custom_config": rschema.StringAttribute{Optional: true, Description: "Custom config passed to a task job."},
		},
	}
}

func (r *jobResource) input(ctx context.Context, plan jobResourceModel) client.JobInput {
	var targets []int64
	if !plan.Targets.IsNull() && !plan.Targets.IsUnknown() {
		plan.Targets.ElementsAs(ctx, &targets, false)
	}
	return client.JobInput{
		Name:              plan.Name.ValueString(),
		Labels:            mergeLabels(r.defaults.DefaultLabels, stringList(ctx, plan.Labels)),
		Enabled:           boolPtr(plan.Enabled),
		WorkflowID:        int64Ptr(plan.WorkflowID),
		TaskID:            int64Ptr(plan.TaskID),
		ScheduleMode:      plan.ScheduleMode.ValueString(),
		DateTime:          plan.DateTime.ValueString(),
		ExecuteScheduleID: int64Ptr(plan.ExecuteScheduleID),
		TargetType:        plan.TargetType.ValueString(),
		Targets:           targets,
		InstanceLabel:     plan.InstanceLabel.ValueString(),
		CustomOptions:     stringMap(ctx, plan.CustomOptions),
		CustomConfig:      plan.CustomConfig.ValueString(),
	}
}

func (r *jobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan jobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	job, err := r.client.CreateJob(ctx, r.input(ctx, plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Job", err)
		return
	}
	setJobState(&plan, job)
	resp.Diagnostics.Append(r.setJobLabelsAll(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *jobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state jobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Job", &resp.Diagnostics)
	if !ok {
		return
	}
	job, err := r.client.GetJob(ctx, id)
	if handleReadError(ctx, err, "Job", &resp.State, &resp.Diagnostics) {
		return
	}
	setJobState(&state, job)
	resp.Diagnostics.Append(r.setJobLabelsAll(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *jobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan jobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Job", &resp.Diagnostics)
	if !ok {
		return
	}
	job, err := r.client.UpdateJob(ctx, id, r.input(ctx, plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Job", err)
		return
	}
	setJobState(&plan, job)
	resp.Diagnostics.Append(r.setJobLabelsAll(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *jobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state jobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Job", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteJob(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Job", err)
	}
}

func (r *jobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func (r *jobResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg jobResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	wf, task := attrPresent(cfg.WorkflowID), attrPresent(cfg.TaskID)
	switch {
	case wf && task:
		resp.Diagnostics.AddError("Conflicting Job Target",
			"Set only one of `workflow_id` or `task_id`, not both.")
	case !wf && !task:
		resp.Diagnostics.AddError("Missing Job Target",
			"Exactly one of `workflow_id` or `task_id` must be set.")
	}
	if task && attrPresent(cfg.CustomOptions) {
		resp.Diagnostics.AddAttributeError(path.Root("custom_options"), "Attribute Not Valid For Task Job",
			"`custom_options` applies to workflow jobs; use `custom_config` for a task job.")
	}
	if wf && attrPresent(cfg.CustomConfig) {
		resp.Diagnostics.AddAttributeError(path.Root("custom_config"), "Attribute Not Valid For Workflow Job",
			"`custom_config` applies to task jobs; use `custom_options` for a workflow job.")
	}
	if attrSet(cfg.ScheduleMode) {
		switch cfg.ScheduleMode.ValueString() {
		case "date_time":
			if !attrPresent(cfg.DateTime) {
				resp.Diagnostics.AddAttributeError(path.Root("date_time"), "Missing Required Attribute",
					"`date_time` is required when `schedule_mode = date_time`.")
			}
		case "schedule":
			if !attrPresent(cfg.ExecuteScheduleID) {
				resp.Diagnostics.AddAttributeError(path.Root("execute_schedule_id"), "Missing Required Attribute",
					"`execute_schedule_id` is required when `schedule_mode = schedule`.")
			}
		}
	}
	if attrSet(cfg.TargetType) && cfg.TargetType.ValueString() == "instance-label" && !attrPresent(cfg.InstanceLabel) {
		resp.Diagnostics.AddAttributeError(path.Root("instance_label"), "Missing Required Attribute",
			"`instance_label` is required when `target_type = instance-label`.")
	}
}

func (r *jobResource) setJobLabelsAll(ctx context.Context, data *jobResourceModel) diag.Diagnostics {
	labels := mergeLabels(r.defaults.DefaultLabels, stringList(ctx, data.Labels))
	labelsAll, diags := types.ListValueFrom(ctx, types.StringType, labels)
	data.LabelsAll = labelsAll
	return diags
}

// setJobState reconciles stable metadata and the workflow/task reference; the
// schedule, target, and custom_* fields are config-authoritative.
func setJobState(data *jobResourceModel, job *client.Job) {
	data.ID = types.StringValue(strconv.FormatInt(job.ID, 10))
	data.Name = types.StringValue(job.Name)
	data.Enabled = mergeAPIBool(data.Enabled, job.Enabled)
	data.TargetType = mergeAPIString(data.TargetType, job.TargetType)
	data.InstanceLabel = mergeAPIString(data.InstanceLabel, job.InstanceLabel)
	if job.Workflow != nil {
		data.WorkflowID = mergeAPIInt64(data.WorkflowID, &job.Workflow.ID)
	}
	if job.Task != nil {
		data.TaskID = mergeAPIInt64(data.TaskID, &job.Task.ID)
	}
}
