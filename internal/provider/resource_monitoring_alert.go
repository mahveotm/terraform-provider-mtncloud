package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &monitoringAlertResource{}
var _ resource.ResourceWithConfigure = &monitoringAlertResource{}
var _ resource.ResourceWithImportState = &monitoringAlertResource{}

type monitoringAlertResource struct {
	resourceBase
}

type monitoringAlertResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	MinSeverity types.String `tfsdk:"min_severity"`
	AllChecks   types.Bool   `tfsdk:"all_checks"`
	CheckIDs    types.Set    `tfsdk:"check_ids"`
	GroupIDs    types.Set    `tfsdk:"group_ids"`
	AppIDs      types.Set    `tfsdk:"app_ids"`
	ContactIDs  types.Set    `tfsdk:"contact_ids"`
}

func NewMonitoringAlertResource() resource.Resource { return &monitoringAlertResource{} }

func (r *monitoringAlertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitoring_alert"
}

func (r *monitoringAlertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	idsDesc := "Config-authoritative: managed here, not reconciled from the API."
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud monitoring alert: a notification rule that pages the listed " +
			"contacts when a referenced check, group, or app (or all checks) crosses `min_severity`.",
		Attributes: map[string]rschema.Attribute{
			"id":           computedIDAttribute("Numeric identifier of the alert."),
			"name":         rschema.StringAttribute{Required: true, Description: "Name of the alert."},
			"min_severity": rschema.StringAttribute{Optional: true, Computed: true, Description: "Minimum severity that triggers the alert (e.g. `info`, `warning`, `critical`)."},
			"all_checks":   rschema.BoolAttribute{Optional: true, Computed: true, Description: "Apply the alert to all checks rather than an explicit list."},
			"check_ids":    rschema.SetAttribute{Optional: true, ElementType: types.Int64Type, Description: "IDs of checks this alert watches. " + idsDesc},
			"group_ids":    rschema.SetAttribute{Optional: true, ElementType: types.Int64Type, Description: "IDs of check groups this alert watches. " + idsDesc},
			"app_ids":      rschema.SetAttribute{Optional: true, ElementType: types.Int64Type, Description: "IDs of monitor apps this alert watches. " + idsDesc},
			"contact_ids":  rschema.SetAttribute{Optional: true, ElementType: types.Int64Type, Description: "IDs of the contacts notified by this alert. " + idsDesc},
		},
	}
}

func (r *monitoringAlertResource) input(ctx context.Context, plan monitoringAlertResourceModel) client.AlertInput {
	return client.AlertInput{
		Name:        plan.Name.ValueString(),
		MinSeverity: plan.MinSeverity.ValueString(),
		AllChecks:   boolPtr(plan.AllChecks),
		CheckIDs:    int64Set(ctx, plan.CheckIDs),
		GroupIDs:    int64Set(ctx, plan.GroupIDs),
		AppIDs:      int64Set(ctx, plan.AppIDs),
		ContactIDs:  int64Set(ctx, plan.ContactIDs),
	}
}

func (r *monitoringAlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitoringAlertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	alert, err := r.client.CreateAlert(ctx, r.input(ctx, plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Monitoring Alert", err)
		return
	}
	setMonitoringAlertState(&plan, alert)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitoringAlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitoringAlertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Monitoring Alert", &resp.Diagnostics)
	if !ok {
		return
	}
	alert, err := r.client.GetAlert(ctx, id)
	if handleReadError(ctx, err, "Monitoring Alert", &resp.State, &resp.Diagnostics) {
		return
	}
	setMonitoringAlertState(&state, alert)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *monitoringAlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitoringAlertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Monitoring Alert", &resp.Diagnostics)
	if !ok {
		return
	}
	alert, err := r.client.UpdateAlert(ctx, id, r.input(ctx, plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Monitoring Alert", err)
		return
	}
	setMonitoringAlertState(&plan, alert)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitoringAlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitoringAlertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Monitoring Alert", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteAlert(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Monitoring Alert", err)
	}
}

func (r *monitoringAlertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

// setMonitoringAlertState reconciles scalar fields; the relational id sets are
// config-authoritative and kept from prior state.
func setMonitoringAlertState(data *monitoringAlertResourceModel, alert *client.Alert) {
	data.ID = types.StringValue(strconv.FormatInt(alert.ID, 10))
	data.Name = types.StringValue(alert.Name)
	data.MinSeverity = mergeAPIString(data.MinSeverity, alert.MinSeverity)
	data.AllChecks = mergeAPIBool(data.AllChecks, alert.AllChecks)
}
