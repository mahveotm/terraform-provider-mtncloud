package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &contactDataSource{}
var _ datasource.DataSourceWithConfigure = &contactDataSource{}

type contactDataSource struct {
	dataSourceBase
}

type contactDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	EmailAddress types.String `tfsdk:"email_address"`
	SMSAddress   types.String `tfsdk:"sms_address"`
	SlackHook    types.String `tfsdk:"slack_hook"`
}

func NewContactDataSource() datasource.DataSource { return &contactDataSource{} }

func (d *contactDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact"
}

func (d *contactDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Looks up an MTN Cloud monitoring contact by name.",
		Attributes: map[string]dschema.Attribute{
			"id":            dschema.StringAttribute{Computed: true, Description: "Numeric identifier of the contact."},
			"name":          dschema.StringAttribute{Required: true, Description: "Name of the contact to look up."},
			"email_address": dschema.StringAttribute{Computed: true, Description: "Email notification address."},
			"sms_address":   dschema.StringAttribute{Computed: true, Description: "SMS notification address."},
			"slack_hook":    dschema.StringAttribute{Computed: true, Description: "Slack incoming-webhook URL."},
		},
	}
}

func (d *contactDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data contactDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	contact, err := d.client.GetContactByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Contact Failed", err.Error())
		return
	}
	data.ID = types.StringValue(strconv.FormatInt(contact.ID, 10))
	data.Name = types.StringValue(contact.Name)
	data.EmailAddress = optionalString(contact.EmailAddress)
	data.SMSAddress = optionalString(contact.SMSAddress)
	data.SlackHook = optionalString(contact.SlackHook)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
