package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &monitoringGroupDataSource{}
var _ datasource.DataSourceWithConfigure = &monitoringGroupDataSource{}

type monitoringGroupDataSource struct {
	dataSourceBase
}

type monitoringGroupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	MinHappy    types.Int64  `tfsdk:"min_happy"`
	Severity    types.String `tfsdk:"severity"`
	CheckIDs    types.Set    `tfsdk:"check_ids"`
}

func NewMonitoringGroupDataSource() datasource.DataSource { return &monitoringGroupDataSource{} }

func (d *monitoringGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitoring_group"
}

func (d *monitoringGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Looks up an MTN Cloud monitoring check group by name.",
		Attributes: map[string]dschema.Attribute{
			"id":          dschema.StringAttribute{Computed: true, Description: "Numeric identifier of the check group."},
			"name":        dschema.StringAttribute{Required: true, Description: "Name of the check group to look up."},
			"description": dschema.StringAttribute{Computed: true, Description: "Description of the check group."},
			"min_happy":   dschema.Int64Attribute{Computed: true, Description: "Minimum number of member checks that must be happy."},
			"severity":    dschema.StringAttribute{Computed: true, Description: "Maximum severity this group can incur."},
			"check_ids":   dschema.SetAttribute{Computed: true, ElementType: types.Int64Type, Description: "IDs of the member checks."},
		},
	}
}

func (d *monitoringGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data monitoringGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	group, err := d.client.GetMonitoringGroupByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Monitoring Group Failed", err.Error())
		return
	}
	data.ID = types.StringValue(strconv.FormatInt(group.ID, 10))
	data.Name = types.StringValue(group.Name)
	data.Description = optionalString(group.Description)
	data.MinHappy = maybeInt64(group.MinHappy)
	data.Severity = optionalString(group.Severity)
	checkIDs, diags := int64SetValue(ctx, group.CheckIDs())
	resp.Diagnostics.Append(diags...)
	data.CheckIDs = checkIDs
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
