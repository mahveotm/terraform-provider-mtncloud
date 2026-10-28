package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

type externalNetworkDataSource struct {
	dataSourceBase
	plural bool
}

func NewExternalNetworkDataSource() datasource.DataSource { return &externalNetworkDataSource{} }
func NewExternalNetworksDataSource() datasource.DataSource {
	return &externalNetworkDataSource{plural: true}
}

func (d *externalNetworkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_network"
	if d.plural {
		resp.TypeName += "s"
	}
}

func (d *externalNetworkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]dschema.Attribute{
		"id":            dschema.StringAttribute{Computed: true, Description: "Morpheus external network ID."},
		"group":         dschema.StringAttribute{Optional: true, Computed: true, Description: "Group name. Defaults to the provider group."},
		"cloud_id":      dschema.Int64Attribute{Optional: true, Computed: true, Validators: []validator.Int64{int64validator.AtLeast(1)}, Description: "Cloud/zone ID. Inferred only when the selected group has exactly one cloud."},
		"resource_pool": dschema.StringAttribute{Optional: true, Computed: true, Description: "Resource pool name or code. Defaults to the provider resource_pool. Options are scoped to this project."},
	}
	description := "Looks up an external network by exact name for an MTN router. Uses router gateway options, which can include networks unavailable through mtncloud_network."
	if d.plural {
		description = "Lists external networks available to an MTN router in the selected group, cloud, and resource pool."
		attributes["id"] = dschema.StringAttribute{Computed: true, Description: "Identifier of the group, cloud, and resource pool used for discovery."}
		attributes["networks"] = dschema.ListNestedAttribute{Computed: true, Description: "Available external networks. IDs are Morpheus network IDs suitable for mtncloud_router.external_network_id, not OpenStack UUIDs.", NestedObject: dschema.NestedAttributeObject{Attributes: map[string]dschema.Attribute{
			"id":   dschema.Int64Attribute{Computed: true, Description: "Morpheus external network ID."},
			"name": dschema.StringAttribute{Computed: true, Description: "External network name."},
		}}}
	} else {
		attributes["name"] = dschema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}, Description: "Exact external network name, for example public-network-01. Ambiguous names are rejected."}
	}
	resp.Schema = dschema.Schema{Description: description, Attributes: attributes}
}

func (d *externalNetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var group, pool types.String
	var cloud types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("group"), &group)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("resource_pool"), &pool)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("cloud_id"), &cloud)...)
	if resp.Diagnostics.HasError() {
		return
	}
	scope, err := resolveRouterScope(ctx, d.client, d.defaults, group.ValueString(), pool.ValueString(), cloud.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Resolve MTN Router Scope Failed", err.Error())
		return
	}
	routerType, err := d.client.GetRouterTypeByName(ctx, client.MTNRouterTypeCode)
	if err != nil {
		resp.Diagnostics.AddError("Resolve MTN Router Type Failed", err.Error())
		return
	}
	if routerType.Code != client.MTNRouterTypeCode || routerType.HasNetworkServer || !routerType.Creatable {
		resp.Diagnostics.AddError("Resolve MTN Router Type Failed", "MTN neutronRouter is not available for creation.")
		return
	}
	networks, err := d.client.ListExternalNetworks(ctx, scope.CloudID, scope.Group.ID, routerType.ID, scope.Pool.Code)
	if err != nil {
		resp.Diagnostics.AddError("Read MTN External Networks Failed", err.Error())
		return
	}
	if group.IsNull() {
		group = types.StringValue(scope.Group.Name)
	}
	if pool.IsNull() {
		pool = types.StringValue(scope.Pool.Name)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group"), group)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_pool"), pool)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cloud_id"), scope.CloudID)...)
	if d.plural {
		objectType := types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.Int64Type, "name": types.StringType}}
		values := make([]attr.Value, 0, len(networks))
		for _, network := range networks {
			value, diags := types.ObjectValue(objectType.AttrTypes, map[string]attr.Value{"id": types.Int64Value(network.ID), "name": types.StringValue(network.Name)})
			resp.Diagnostics.Append(diags...)
			values = append(values, value)
		}
		value, diags := types.ListValue(objectType, values)
		resp.Diagnostics.Append(diags...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("networks"), value)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%d/%d/%s", scope.Group.ID, scope.CloudID, scope.Pool.Code))...)
	} else {
		var name types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("name"), &name)...)
		if resp.Diagnostics.HasError() {
			return
		}
		var match *client.ExternalNetwork
		available := make([]string, 0, len(networks))
		for i := range networks {
			available = append(available, networks[i].Name)
			if networks[i].Name == name.ValueString() {
				if match != nil {
					resp.Diagnostics.AddError("Ambiguous MTN External Network", fmt.Sprintf("More than one external network is named %q in the selected project. Use mtncloud_external_networks to inspect the IDs and choose explicitly.", name.ValueString()))
					return
				}
				match = &networks[i]
			}
		}
		if match == nil {
			resp.Diagnostics.AddError("MTN External Network Not Found", fmt.Sprintf("External network %q is not available in cloud %d, resource pool %q. Available names: %s.", name.ValueString(), scope.CloudID, scope.Pool.Name, strings.Join(available, ", ")))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.FormatInt(match.ID, 10))...)
	}
}
