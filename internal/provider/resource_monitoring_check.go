package provider

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &monitoringCheckResource{}
var _ resource.ResourceWithConfigure = &monitoringCheckResource{}
var _ resource.ResourceWithImportState = &monitoringCheckResource{}

type monitoringCheckResource struct {
	resourceBase
}

type monitoringCheckResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	CheckType     types.String `tfsdk:"check_type"`
	Description   types.String `tfsdk:"description"`
	CheckInterval types.Int64  `tfsdk:"check_interval"`
	InUptime      types.Bool   `tfsdk:"in_uptime"`
	Active        types.Bool   `tfsdk:"active"`
	Severity      types.String `tfsdk:"severity"`
	Config        types.String `tfsdk:"config"`
}

func NewMonitoringCheckResource() resource.Resource { return &monitoringCheckResource{} }

func (r *monitoringCheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitoring_check"
}

func (r *monitoringCheckResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud monitoring check: an individual monitor (HTTP, ping, SNMP, …). " +
			"`check_type` selects the monitor implementation and is immutable. Per-type settings go in " +
			"`config` as a JSON document (config-authoritative, not read back). Use the " +
			"`mtncloud_monitoring_check` data source or the Morpheus check-types reference to find a `check_type` code.",
		Attributes: map[string]rschema.Attribute{
			"id":   computedIDAttribute("Numeric identifier of the check."),
			"name": rschema.StringAttribute{Required: true, Description: "Name of the check."},
			"check_type": rschema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Check-type code (e.g. `webGetCheck`, `pingCheck`). Changing it forces a new check.",
			},
			"description":    rschema.StringAttribute{Optional: true, Computed: true, Description: "Description of the check."},
			"check_interval": rschema.Int64Attribute{Optional: true, Computed: true, Description: "Milliseconds between check executions (minimum is one minute, subject to your plan)."},
			"in_uptime":      rschema.BoolAttribute{Optional: true, Computed: true, Description: "Whether this check affects account-wide availability calculations."},
			"active":         rschema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the check is scheduled to execute."},
			"severity":       rschema.StringAttribute{Optional: true, Computed: true, Description: "Severity threshold for notifications (e.g. `info`, `warning`, `critical`)."},
			"config": rschema.StringAttribute{
				Optional: true,
				Description: "Per-type configuration as a JSON object, e.g. " +
					"`jsonencode({ webUrl = \"https://example.com\" })`. Config-authoritative (not read back).",
			},
		},
	}
}

func (r *monitoringCheckResource) input(plan monitoringCheckResourceModel, diags *diag.Diagnostics) client.CheckInput {
	var config map[string]any
	if raw := plan.Config.ValueString(); raw != "" {
		if err := json.Unmarshal([]byte(raw), &config); err != nil {
			diags.AddError("Invalid Check Config JSON", "The `config` attribute must be a JSON object: "+err.Error())
		}
	}
	return client.CheckInput{
		Name:          plan.Name.ValueString(),
		CheckType:     plan.CheckType.ValueString(),
		Description:   plan.Description.ValueString(),
		CheckInterval: int64Ptr(plan.CheckInterval),
		InUptime:      boolPtr(plan.InUptime),
		Active:        boolPtr(plan.Active),
		Severity:      plan.Severity.ValueString(),
		Config:        config,
	}
}

func (r *monitoringCheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitoringCheckResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input := r.input(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	check, err := r.client.CreateCheck(ctx, input)
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Monitoring Check", err)
		return
	}
	setMonitoringCheckState(&plan, check)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitoringCheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitoringCheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Monitoring Check", &resp.Diagnostics)
	if !ok {
		return
	}
	check, err := r.client.GetCheck(ctx, id)
	if handleReadError(ctx, err, "Monitoring Check", &resp.State, &resp.Diagnostics) {
		return
	}
	setMonitoringCheckState(&state, check)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *monitoringCheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitoringCheckResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Monitoring Check", &resp.Diagnostics)
	if !ok {
		return
	}
	input := r.input(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	check, err := r.client.UpdateCheck(ctx, id, input)
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Monitoring Check", err)
		return
	}
	setMonitoringCheckState(&plan, check)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitoringCheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitoringCheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Monitoring Check", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteCheck(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Monitoring Check", err)
	}
}

func (r *monitoringCheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

// setMonitoringCheckState reconciles read-back fields; check_type is immutable and
// config is config-authoritative, so both are kept from prior state.
func setMonitoringCheckState(data *monitoringCheckResourceModel, check *client.Check) {
	data.ID = types.StringValue(strconv.FormatInt(check.ID, 10))
	data.Name = types.StringValue(check.Name)
	if check.CheckType.Code != "" {
		data.CheckType = types.StringValue(check.CheckType.Code)
	}
	data.Description = mergeAPIString(data.Description, check.Description)
	data.CheckInterval = mergeAPIInt64(data.CheckInterval, check.CheckInterval)
	data.InUptime = mergeAPIBool(data.InUptime, check.InUptime)
	data.Active = mergeAPIBool(data.Active, check.Active)
	data.Severity = mergeAPIString(data.Severity, check.Severity)
}
