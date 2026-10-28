package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &networkDataSource{}
var _ datasource.DataSourceWithConfigure = &networkDataSource{}

type networkDataSource struct {
	dataSourceBase
}

type networkDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Code      types.String `tfsdk:"code"`
	CIDR      types.String `tfsdk:"cidr"`
	Gateway   types.String `tfsdk:"gateway"`
	Status    types.String `tfsdk:"status"`
	TypeID    types.Int64  `tfsdk:"type_id"`
	CloudID   types.Int64  `tfsdk:"cloud_id"`
	SubnetIDs types.Set    `tfsdk:"subnet_ids"`
}

func NewNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

func (d *networkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (d *networkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Looks up an MTN Cloud network by name. For router public gateways, use mtncloud_external_network instead.",
		Attributes: map[string]dschema.Attribute{
			"subnet_ids": dschema.SetAttribute{Computed: true, ElementType: types.Int64Type, Description: "Morpheus subnet IDs belonging to this network. Empty when the network has no subnets. Use with mtncloud_router.internal_subnet_ids to attach all subnets of an internal network."},
			"id":         dschema.StringAttribute{Computed: true, Description: "Numeric identifier of the network."},
			"name":       dschema.StringAttribute{Required: true, Description: "Name of the network to look up."},
			"code":       dschema.StringAttribute{Computed: true, Description: "Code of the network."},
			"cidr":       dschema.StringAttribute{Computed: true, Description: "CIDR block of the network."},
			"gateway":    dschema.StringAttribute{Computed: true, Description: "Gateway address of the network."},
			"status":     dschema.StringAttribute{Computed: true, Description: "Current status of the network."},
			"type_id":    dschema.Int64Attribute{Computed: true, Description: "ID of the network type."},
			"cloud_id":   dschema.Int64Attribute{Optional: true, Computed: true, Description: "Cloud/zone ID to disambiguate networks with the same name. Get it from mtncloud_group.cloud_ids."},
		},
	}
}

func (d *networkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data networkDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var zoneID int64
	if !data.CloudID.IsNull() && !data.CloudID.IsUnknown() {
		zoneID = data.CloudID.ValueInt64()
	}

	network, err := d.client.GetNetworkByName(ctx, data.Name.ValueString(), zoneID)
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Network Failed", err.Error())
		return
	}

	// The list response may omit subnets; read the full network details.
	network, err = d.client.GetNetwork(ctx, network.ID)
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Network Failed", err.Error())
		return
	}
	subnetIDs := make([]int64, 0, len(network.Subnets))
	for _, subnet := range network.Subnets {
		subnetIDs = append(subnetIDs, subnet.ID)
	}
	subnets, diags := types.SetValueFrom(ctx, types.Int64Type, subnetIDs)
	resp.Diagnostics.Append(diags...)
	data.SubnetIDs = subnets

	data.ID = types.StringValue(strconv.FormatInt(network.ID, 10))
	data.Code = optionalString(network.Code)
	data.CIDR = optionalString(network.CIDR)
	data.Gateway = optionalString(network.Gateway)
	data.Status = optionalString(network.Status)
	data.TypeID = int64ValueOrNull(nestedID(network.Type))
	data.CloudID = int64ValueOrNull(nestedID(network.Zone))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
