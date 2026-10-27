package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &routerResource{}
var _ resource.ResourceWithConfigure = &routerResource{}
var _ resource.ResourceWithImportState = &routerResource{}
var _ resource.ResourceWithValidateConfig = &routerResource{}

type routerResource struct{ resourceBase }
type routerModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Type              types.String `tfsdk:"type"`
	Group             types.String `tfsdk:"group"`
	CloudID           types.Int64  `tfsdk:"cloud_id"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	ResourcePool      types.String `tfsdk:"resource_pool"`
	SubnetIDs         types.Set    `tfsdk:"internal_subnet_ids"`
	ExternalNetworkID types.Int64  `tfsdk:"external_network_id"`
	Status            types.String `tfsdk:"status"`
	ExternalIP        types.String `tfsdk:"external_ip"`
}

func NewRouterResource() resource.Resource { return &routerResource{} }
func (r *routerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_router"
}
func (r *routerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud router using the MTN neutronRouter offering. Other router backends are not supported.",
		Attributes: map[string]rschema.Attribute{
			"id":                  computedIDAttribute("Numeric router ID."),
			"name":                rschema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Router name. Changes replace the router."},
			"type":                rschema.StringAttribute{Computed: true, Description: "MTN router backend code (neutronRouter)."},
			"group":               rschema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}, Description: "Group name; defaults to the provider group. Changes replace the router. Shared routers are not supported."},
			"cloud_id":            rschema.Int64Attribute{Optional: true, Computed: true, Validators: []validator.Int64{int64validator.AtLeast(1)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace(), int64planmodifier.UseStateForUnknown()}, Description: "MTN Cloud ID. Inferred only when the group has exactly one cloud. Changes replace the router."},
			"enabled":             rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the router is enabled at creation. Changes replace the router.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"resource_pool":       rschema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}, Description: "MTN resource pool name/code, inherited from the provider when omitted. Changes replace the router."},
			"internal_subnet_ids": rschema.SetAttribute{Required: true, ElementType: types.Int64Type, PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()}, Description: "Morpheus internal subnet IDs to attach. Changes replace the router."},
			"external_network_id": rschema.Int64Attribute{Optional: true, Validators: []validator.Int64{int64validator.AtLeast(1)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}, Description: "Optional Morpheus external network ID. Changes replace the router."},
			"status":              rschema.StringAttribute{Computed: true, Description: "Observed MTN router status. Creation waits up to five minutes for readiness."},
			"external_ip":         rschema.StringAttribute{Computed: true, Description: "External IP, if assigned."},
		},
	}
}
func (r *routerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data routerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !data.SubnetIDs.IsUnknown() && !data.SubnetIDs.IsNull() {
		var ids []int64
		resp.Diagnostics.Append(data.SubnetIDs.ElementsAs(ctx, &ids, false)...)
		if len(ids) == 0 {
			resp.Diagnostics.AddError("Missing Internal Subnet", "Specify at least one internal_subnet_ids value.")
		}
		for _, id := range ids {
			if id < 1 {
				resp.Diagnostics.AddError("Invalid Internal Subnet", "Subnet IDs must be positive.")
			}
		}
	}

}
func (r *routerResource) createFields(ctx context.Context, data *routerModel) (map[string]any, error) {
	fields := map[string]any{}
	groupName := data.Group.ValueString()
	if groupName == "" && r.defaults != nil {
		groupName = r.defaults.Group
	}
	if groupName == "" {
		return nil, fmt.Errorf("set group on the router or provider")
	}
	group, err := r.client.GetGroupByName(ctx, groupName)
	if err != nil {
		return nil, err
	}
	routerType, err := r.client.GetRouterTypeByName(ctx, client.MTNRouterTypeCode)
	if err != nil {
		return nil, err
	}
	if routerType.Code != client.MTNRouterTypeCode || routerType.HasNetworkServer || !routerType.Creatable {
		return nil, fmt.Errorf("router type %q does not support creation", client.MTNRouterTypeCode)
	}
	fields["name"] = data.Name.ValueString()
	fields["type"] = map[string]any{"id": routerType.ID}
	fields["site"] = map[string]any{"id": group.ID}
	fields["enabled"] = data.Enabled.ValueBool()
	data.Group = types.StringValue(groupName)
	cloudID := data.CloudID.ValueInt64()
	if cloudID == 0 {
		if len(group.CloudIDs) != 1 {
			return nil, fmt.Errorf("set cloud_id explicitly: group has %d clouds", len(group.CloudIDs))
		}
		cloudID = group.CloudIDs[0]
	}
	found := false
	for _, id := range group.CloudIDs {
		if id == cloudID {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("cloud_id %d is not attached to group %q", cloudID, groupName)
	}
	fields["zone"] = map[string]any{"id": cloudID}
	data.CloudID = types.Int64Value(cloudID)
	data.Type = types.StringValue(client.MTNRouterTypeCode)
	poolName := data.ResourcePool.ValueString()
	if poolName == "" && r.defaults != nil {
		poolName = r.defaults.ResourcePool
	}
	if poolName == "" {
		return nil, fmt.Errorf("set resource_pool on the router or provider")
	}
	pools, err := r.client.ListResourcePools(ctx, cloudID, group.ID)
	if err != nil {
		return nil, err
	}
	var poolID int64
	for _, pool := range pools {
		if pool.Name == poolName || pool.Code == poolName {
			poolID = pool.ID
			break
		}
	}
	if poolID == 0 {
		return nil, fmt.Errorf("resource pool %q not found in selected cloud", poolName)
	}
	fields["poolId"] = fmt.Sprintf("pool-%d", poolID)
	// The REST provisioning backend reads the project from config.poolId.
	// A top-level poolId alone creates a failed local router record.
	fields["config"] = map[string]any{"poolId": fmt.Sprintf("pool-%d", poolID)}
	data.ResourcePool = types.StringValue(poolName)
	var ids []int64
	if diags := data.SubnetIDs.ElementsAs(ctx, &ids, false); diags.HasError() {
		return nil, fmt.Errorf("invalid internal_subnet_ids")
	}
	interfaces := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		interfaces = append(interfaces, map[string]any{"networkSubnet": map[string]any{"id": id}})
	}
	fields["interfaces"] = interfaces
	if !data.ExternalNetworkID.IsNull() {
		fields["externalNetwork"] = map[string]any{"id": data.ExternalNetworkID.ValueInt64()}
	}

	return fields, nil
}
func (r *routerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data routerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fields, err := r.createFields(ctx, &data)
	if err != nil {
		opError(&resp.Diagnostics, "Resolve", "Router", err)
		return
	}
	id, err := r.client.CreateRouter(ctx, fields)
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Router", err)
		return
	}
	data.ID = types.StringValue(strconv.FormatInt(id, 10))
	data.Status = types.StringNull()
	data.ExternalIP = types.StringNull()
	// Persist the ID even if the follow-up read fails, avoiding orphaned routers.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	router, err := r.client.WaitForRouter(waitCtx, id, false)
	if err != nil {
		opError(&resp.Diagnostics, "Wait for Created", "Router", err)
		return
	}
	data.Status = optionalString(router.Status)
	data.ExternalIP = optionalString(router.ExternalIP)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *routerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data routerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(data.ID, "Router", &resp.Diagnostics)
	if !ok {
		return
	}
	router, err := r.client.GetRouter(ctx, id)
	if handleReadError(ctx, err, "Router", &resp.State, &resp.Diagnostics) {
		return
	}
	data.Name = types.StringValue(router.Name)
	data.Enabled = types.BoolValue(router.Enabled)
	if data.Type.IsNull() || (data.Type.ValueString() != router.Type.Name && data.Type.ValueString() != router.Type.Code) {
		data.Type = types.StringValue(router.Type.Code)
	}
	if router.Site.Name != "" {
		data.Group = types.StringValue(router.Site.Name)
	} else if router.Site.ID > 0 {
		group, err := r.client.GetGroup(ctx, router.Site.ID)
		if err != nil {
			opError(&resp.Diagnostics, "Read", "Router Group", err)
			return
		}
		data.Group = types.StringValue(group.Name)
	}
	if router.Type.Code != client.MTNRouterTypeCode {
		resp.Diagnostics.AddError("Unsupported Router", "Only MTN neutronRouter resources are supported.")
		return
	}
	if router.Zone.ID > 0 {
		data.CloudID = types.Int64Value(router.Zone.ID)
	}

	var subnetIDs []int64
	var pool *client.RouterRef
	for _, iface := range router.Interfaces {
		if iface.Network.ID == router.ExternalNetwork.ID {
			continue
		}
		network, err := r.client.GetRouterNetwork(ctx, iface.Network.ID)
		if err != nil {
			opError(&resp.Diagnostics, "Read", "Router Network", err)
			return
		}
		pool = &network.ZonePool
		for _, subnet := range network.Subnets {
			if subnet.CIDR == iface.CIDR {
				subnetIDs = append(subnetIDs, subnet.ID)
			}
		}
	}
	value, diags := types.SetValueFrom(ctx, types.Int64Type, subnetIDs)
	resp.Diagnostics.Append(diags...)
	data.SubnetIDs = value
	if pool != nil && pool.ID > 0 {
		code := fmt.Sprintf("pool-%d", pool.ID)
		if data.ResourcePool.IsNull() || (data.ResourcePool.ValueString() != pool.Name && data.ResourcePool.ValueString() != code) {
			data.ResourcePool = types.StringValue(code)
		}
	}
	if router.ExternalNetwork.ID > 0 {
		data.ExternalNetworkID = types.Int64Value(router.ExternalNetwork.ID)
	} else {
		data.ExternalNetworkID = types.Int64Null()
	}

	data.Status = optionalString(router.Status)
	data.ExternalIP = optionalString(router.ExternalIP)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *routerResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Router Replacement Required", "MTN router configuration changes require replacement.")
}
func (r *routerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data routerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(data.ID, "Router", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteRouter(ctx, id); err != nil {
		if !client.IsNotFound(err) {
			opError(&resp.Diagnostics, "Delete", "Router", err)
		}
		return
	}
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if _, err := r.client.WaitForRouter(waitCtx, id, true); err != nil {
		opError(&resp.Diagnostics, "Wait for Deleted", "Router", err)
	}
}
func (r *routerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}
