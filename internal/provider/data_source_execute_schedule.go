package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &executeScheduleDataSource{}
var _ datasource.DataSourceWithConfigure = &executeScheduleDataSource{}

type executeScheduleDataSource struct {
	dataSourceBase
}

type executeScheduleDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Cron     types.String `tfsdk:"cron"`
	Timezone types.String `tfsdk:"timezone"`
	Enabled  types.Bool   `tfsdk:"enabled"`
}

func NewExecuteScheduleDataSource() datasource.DataSource {
	return &executeScheduleDataSource{}
}

func (d *executeScheduleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_execute_schedule"
}

func (d *executeScheduleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Looks up an MTN Cloud execute schedule by name.",
		Attributes: map[string]dschema.Attribute{
			"id":       dschema.StringAttribute{Computed: true, Description: "Numeric identifier of the schedule."},
			"name":     dschema.StringAttribute{Required: true, Description: "Name of the schedule to look up."},
			"cron":     dschema.StringAttribute{Computed: true, Description: "Cron expression."},
			"timezone": dschema.StringAttribute{Computed: true, Description: "Schedule timezone."},
			"enabled":  dschema.BoolAttribute{Computed: true, Description: "Whether the schedule is enabled."},
		},
	}
}

func (d *executeScheduleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data executeScheduleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sched, err := d.client.GetExecuteScheduleByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Execute Schedule Failed", err.Error())
		return
	}
	data.ID = types.StringValue(strconv.FormatInt(sched.ID, 10))
	data.Name = types.StringValue(sched.Name)
	data.Cron = optionalString(sched.Cron)
	data.Timezone = optionalString(sched.Timezone)
	data.Enabled = maybeBool(sched.Enabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
