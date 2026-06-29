package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &monitoringAlertDataSource{}
var _ datasource.DataSourceWithConfigure = &monitoringAlertDataSource{}

type monitoringAlertDataSource struct {
	dataSourceBase
}

type monitoringAlertDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	MinSeverity types.String `tfsdk:"min_severity"`
	AllChecks   types.Bool   `tfsdk:"all_checks"`
	CheckIDs    types.Set    `tfsdk:"check_ids"`
	GroupIDs    types.Set    `tfsdk:"group_ids"`
	AppIDs      types.Set    `tfsdk:"app_ids"`
	ContactIDs  types.Set    `tfsdk:"contact_ids"`
}

func NewMonitoringAlertDataSource() datasource.DataSource { return &monitoringAlertDataSource{} }

func (d *monitoringAlertDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitoring_alert"
}

func (d *monitoringAlertDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Looks up an MTN Cloud monitoring alert by name.",
		Attributes: map[string]dschema.Attribute{
			"id":           dschema.StringAttribute{Computed: true, Description: "Numeric identifier of the alert."},
			"name":         dschema.StringAttribute{Required: true, Description: "Name of the alert to look up."},
			"min_severity": dschema.StringAttribute{Computed: true, Description: "Minimum severity that triggers the alert."},
			"all_checks":   dschema.BoolAttribute{Computed: true, Description: "Whether the alert applies to all checks."},
			"check_ids":    dschema.SetAttribute{Computed: true, ElementType: types.Int64Type, Description: "IDs of checks this alert watches."},
			"group_ids":    dschema.SetAttribute{Computed: true, ElementType: types.Int64Type, Description: "IDs of check groups this alert watches."},
			"app_ids":      dschema.SetAttribute{Computed: true, ElementType: types.Int64Type, Description: "IDs of monitor apps this alert watches."},
			"contact_ids":  dschema.SetAttribute{Computed: true, ElementType: types.Int64Type, Description: "IDs of the contacts notified by this alert."},
		},
	}
}

func (d *monitoringAlertDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data monitoringAlertDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	alert, err := d.client.GetAlertByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Monitoring Alert Failed", err.Error())
		return
	}
	data.ID = types.StringValue(strconv.FormatInt(alert.ID, 10))
	data.Name = types.StringValue(alert.Name)
	data.MinSeverity = optionalString(alert.MinSeverity)
	data.AllChecks = maybeBool(alert.AllChecks)
	checkIDs, diags := int64SetValue(ctx, alert.CheckIDs())
	resp.Diagnostics.Append(diags...)
	data.CheckIDs = checkIDs
	groupIDs, diags := int64SetValue(ctx, alert.GroupIDs())
	resp.Diagnostics.Append(diags...)
	data.GroupIDs = groupIDs
	appIDs, diags := int64SetValue(ctx, alert.AppIDs())
	resp.Diagnostics.Append(diags...)
	data.AppIDs = appIDs
	contactIDs, diags := int64SetValue(ctx, alert.ContactIDs())
	resp.Diagnostics.Append(diags...)
	data.ContactIDs = contactIDs
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
