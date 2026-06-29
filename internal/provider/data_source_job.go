package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &jobDataSource{}
var _ datasource.DataSourceWithConfigure = &jobDataSource{}

type jobDataSource struct {
	dataSourceBase
}

type jobDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	WorkflowID    types.Int64  `tfsdk:"workflow_id"`
	TaskID        types.Int64  `tfsdk:"task_id"`
	TargetType    types.String `tfsdk:"target_type"`
	InstanceLabel types.String `tfsdk:"instance_label"`
}

func NewJobDataSource() datasource.DataSource {
	return &jobDataSource{}
}

func (d *jobDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (d *jobDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Looks up an MTN Cloud job by name.",
		Attributes: map[string]dschema.Attribute{
			"id":          dschema.StringAttribute{Computed: true, Description: "Numeric identifier of the job."},
			"name":        dschema.StringAttribute{Required: true, Description: "Name of the job to look up."},
			"enabled":     dschema.BoolAttribute{Computed: true, Description: "Whether the job is enabled."},
			"workflow_id": dschema.Int64Attribute{Computed: true, Description: "ID of the workflow the job runs, if any."},
			"task_id":     dschema.Int64Attribute{Computed: true, Description: "ID of the task the job runs, if any."},
			"target_type": dschema.StringAttribute{Computed: true, Description: "Target type used by the job."},
			"instance_label": dschema.StringAttribute{
				Computed:    true,
				Description: "Instance label targeted by the job, if any.",
			},
		},
	}
}

func (d *jobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data jobDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	job, err := d.client.GetJobByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Job Failed", err.Error())
		return
	}
	data.ID = types.StringValue(strconv.FormatInt(job.ID, 10))
	data.Name = types.StringValue(job.Name)
	data.Enabled = maybeBool(job.Enabled)
	data.TargetType = optionalString(job.TargetType)
	data.InstanceLabel = optionalString(job.InstanceLabel)
	if job.Workflow != nil {
		data.WorkflowID = types.Int64Value(job.Workflow.ID)
	}
	if job.Task != nil {
		data.TaskID = types.Int64Value(job.Task.ID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
