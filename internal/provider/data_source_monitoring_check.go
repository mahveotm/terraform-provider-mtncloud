package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &monitoringCheckDataSource{}
var _ datasource.DataSourceWithConfigure = &monitoringCheckDataSource{}

type monitoringCheckDataSource struct {
	dataSourceBase
}

type monitoringCheckDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	CheckType     types.String `tfsdk:"check_type"`
	Description   types.String `tfsdk:"description"`
	CheckInterval types.Int64  `tfsdk:"check_interval"`
	Severity      types.String `tfsdk:"severity"`
}

func NewMonitoringCheckDataSource() datasource.DataSource { return &monitoringCheckDataSource{} }

func (d *monitoringCheckDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitoring_check"
}

func (d *monitoringCheckDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Looks up an MTN Cloud monitoring check by name.",
		Attributes: map[string]dschema.Attribute{
			"id":             dschema.StringAttribute{Computed: true, Description: "Numeric identifier of the check."},
			"name":           dschema.StringAttribute{Required: true, Description: "Name of the check to look up."},
			"check_type":     dschema.StringAttribute{Computed: true, Description: "Check-type code."},
			"description":    dschema.StringAttribute{Computed: true, Description: "Description of the check."},
			"check_interval": dschema.Int64Attribute{Computed: true, Description: "Milliseconds between check executions."},
			"severity":       dschema.StringAttribute{Computed: true, Description: "Severity threshold for notifications."},
		},
	}
}

func (d *monitoringCheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data monitoringCheckDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	check, err := d.client.GetCheckByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Monitoring Check Failed", err.Error())
		return
	}
	data.ID = types.StringValue(strconv.FormatInt(check.ID, 10))
	data.Name = types.StringValue(check.Name)
	data.CheckType = optionalString(check.CheckType.Code)
	data.Description = optionalString(check.Description)
	data.CheckInterval = maybeInt64(check.CheckInterval)
	data.Severity = optionalString(check.Severity)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
