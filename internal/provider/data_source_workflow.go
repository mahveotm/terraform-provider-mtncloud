package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &workflowDataSource{}
var _ datasource.DataSourceWithConfigure = &workflowDataSource{}

type workflowDataSource struct {
	dataSourceBase
}

type workflowDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Visibility types.String `tfsdk:"visibility"`
	Platform   types.String `tfsdk:"platform"`
}

func NewWorkflowDataSource() datasource.DataSource {
	return &workflowDataSource{}
}

func (d *workflowDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (d *workflowDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Looks up an MTN Cloud workflow (task-set) by name.",
		Attributes: map[string]dschema.Attribute{
			"id":         dschema.StringAttribute{Computed: true, Description: "Numeric identifier of the workflow."},
			"name":       dschema.StringAttribute{Required: true, Description: "Name of the workflow to look up."},
			"type":       dschema.StringAttribute{Computed: true, Description: "Workflow type (operation or provision)."},
			"visibility": dschema.StringAttribute{Computed: true, Description: "Workflow visibility."},
			"platform":   dschema.StringAttribute{Computed: true, Description: "Platform filter."},
		},
	}
}

func (d *workflowDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data workflowDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	wf, err := d.client.GetWorkflowByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Workflow Failed", err.Error())
		return
	}
	data.ID = types.StringValue(strconv.FormatInt(wf.ID, 10))
	data.Name = types.StringValue(wf.Name)
	data.Type = optionalString(wf.Type)
	data.Visibility = optionalString(wf.Visibility)
	data.Platform = optionalString(wf.Platform)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
