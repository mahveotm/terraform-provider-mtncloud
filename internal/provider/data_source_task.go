package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ datasource.DataSource = &taskDataSource{}
var _ datasource.DataSourceWithConfigure = &taskDataSource{}

type taskDataSource struct {
	dataSourceBase
}

type taskDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	Code          types.String `tfsdk:"code"`
	ExecuteTarget types.String `tfsdk:"execute_target"`
}

func NewTaskDataSource() datasource.DataSource {
	return &taskDataSource{}
}

func (d *taskDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_task"
}

func (d *taskDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Looks up an MTN Cloud automation task by name.",
		Attributes: map[string]dschema.Attribute{
			"id":             dschema.StringAttribute{Computed: true, Description: "Numeric identifier of the task."},
			"name":           dschema.StringAttribute{Required: true, Description: "Name of the task to look up."},
			"type":           dschema.StringAttribute{Computed: true, Description: "Friendly task type (shell, python, …)."},
			"code":           dschema.StringAttribute{Computed: true, Description: "User-defined code/identifier."},
			"execute_target": dschema.StringAttribute{Computed: true, Description: "Where the task runs."},
		},
	}
}

func (d *taskDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data taskDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	task, err := d.client.GetTaskByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Task Failed", err.Error())
		return
	}
	data.ID = types.StringValue(strconv.FormatInt(task.ID, 10))
	data.Name = types.StringValue(task.Name)
	data.Type = optionalString(client.TaskTypeFromCode(task.TaskType.Code))
	data.Code = optionalString(task.Code)
	data.ExecuteTarget = optionalString(task.ExecuteTarget)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
