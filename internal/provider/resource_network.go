package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &networkResource{}
var _ resource.ResourceWithConfigure = &networkResource{}
var _ resource.ResourceWithImportState = &networkResource{}

type networkResource struct {
	resourceBase
}

type networkResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	Name                types.String   `tfsdk:"name"`
	Group               types.String   `tfsdk:"group"`
	Type                types.String   `tfsdk:"type"`
	ResourcePool        types.String   `tfsdk:"resource_pool"`
	Description         types.String   `tfsdk:"description"`
	Labels              types.List     `tfsdk:"labels"`
	LabelsAll           types.List     `tfsdk:"labels_all"`
	CIDR                types.String   `tfsdk:"cidr"`
	Gateway             types.String   `tfsdk:"gateway"`
	DNSPrimary          types.String   `tfsdk:"dns_primary"`
	DNSSecondary        types.String   `tfsdk:"dns_secondary"`
	VlanID              types.Int64    `tfsdk:"vlan_id"`
	DHCPServer          types.Bool     `tfsdk:"dhcp_server"`
	AssignPublicIP      types.Bool     `tfsdk:"assign_public_ip"`
	AllowStaticOverride types.Bool     `tfsdk:"allow_static_override"`
	Active              types.Bool     `tfsdk:"active"`
	Visibility          types.String   `tfsdk:"visibility"`
	Code                types.String   `tfsdk:"code"`
	Status              types.String   `tfsdk:"status"`
	CloudID             types.Int64    `tfsdk:"cloud_id"`
	GroupID             types.Int64    `tfsdk:"group_id"`
	TypeID              types.Int64    `tfsdk:"type_id"`
	ResourcePoolID      types.Int64    `tfsdk:"resource_pool_id"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

func (r *networkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *networkResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceString := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	inheritString := []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}

	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud network using human-friendly group/zone/type/pool names.",
		Attributes: map[string]rschema.Attribute{
			"id":            computedIDAttribute("Numeric identifier of the network."),
			"name":          rschema.StringAttribute{Required: true, Description: "Name of the network."},
			"group":         rschema.StringAttribute{Optional: true, Computed: true, PlanModifiers: inheritString, Description: "Group/site name. Defaults to the provider's `group`. The group's first cloud is used as the network's zone (see cloud_id). Changing it forces a new network."},
			"type":          rschema.StringAttribute{Optional: true, PlanModifiers: replaceString, Description: "Network type name or code (e.g. an OpenStack network type). Changing it forces a new network."},
			"resource_pool": rschema.StringAttribute{Optional: true, Computed: true, PlanModifiers: inheritString, Description: "Resource pool name or code. Defaults to the provider's `resource_pool`. Required for OpenStack networks. Changing it forces a new network."},
			"description":   rschema.StringAttribute{Optional: true, Description: "Human-readable description of the network."},
			"cidr": rschema.StringAttribute{
				Optional:    true,
				Description: "CIDR block for the network, e.g. `10.0.0.0/24`.",
				Validators:  []validator.String{validCIDR()},
			},
			"gateway":       rschema.StringAttribute{Optional: true, Description: "Gateway IP address for the network."},
			"dns_primary":   rschema.StringAttribute{Optional: true, Description: "Primary DNS server for the network."},
			"dns_secondary": rschema.StringAttribute{Optional: true, Description: "Secondary DNS server for the network."},
			"vlan_id": rschema.Int64Attribute{
				Optional:    true,
				Description: "VLAN ID for the network (1-4094).",
				Validators:  []validator.Int64{int64validator.Between(1, 4094)},
			},
			"dhcp_server":           rschema.BoolAttribute{Optional: true, Description: "Whether DHCP is enabled on the network."},
			"assign_public_ip":      rschema.BoolAttribute{Optional: true, Description: "Whether instances on this network are assigned a public IP."},
			"allow_static_override": rschema.BoolAttribute{Optional: true, Description: "Whether static IP assignment may override DHCP."},
			"active":                rschema.BoolAttribute{Optional: true, Description: "Whether the network is active."},
			"visibility": rschema.StringAttribute{
				Optional:    true,
				Description: "Network visibility: `private` or `public`.",
				Validators:  []validator.String{stringvalidator.OneOf("private", "public")},
			},
			"labels": rschema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Labels applied to the network. Merged with the provider's default_labels into `labels_all`.",
			},
			"labels_all": rschema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Effective labels: the provider's default_labels merged (union) with `labels`.",
			},
			"code":             rschema.StringAttribute{Computed: true, Description: "Code of the network."},
			"status":           rschema.StringAttribute{Computed: true, Description: "Current status of the network."},
			"cloud_id":         rschema.Int64Attribute{Computed: true, Description: "ID of the cloud/zone the network was created in (the group's first cloud)."},
			"group_id":         rschema.Int64Attribute{Computed: true, Description: "Resolved numeric ID of the group."},
			"type_id":          rschema.Int64Attribute{Computed: true, Description: "Resolved numeric ID of the network type."},
			"resource_pool_id": rschema.Int64Attribute{Computed: true, Description: "Resolved numeric ID of the resource pool."},
		},
		Blocks: map[string]rschema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	resolved, err := r.resolveNetwork(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Resolve MTN Cloud Network Inputs Failed", err.Error())
		return
	}
	// Record resolved values for inherited Optional+Computed inputs.
	plan.Group = types.StringValue(resolved.GroupName)
	plan.ResourcePool = optionalString(resolved.ResourcePoolName)
	resp.Diagnostics.Append(r.setNetworkLabelsAll(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateNetwork(ctx, r.networkInput(ctx, plan, resolved))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Network", err)
		return
	}
	network, err := r.client.GetNetwork(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Network Failed", err.Error())
		return
	}
	setNetworkComputed(&plan, network)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Network", &resp.Diagnostics)
	if !ok {
		return
	}
	network, err := r.client.GetNetwork(ctx, id)
	if handleReadError(ctx, err, "Network", &resp.State, &resp.Diagnostics) {
		return
	}
	setNetworkComputed(&state, network)
	resp.Diagnostics.Append(r.setNetworkLabelsAll(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan networkResourceModel
	var state networkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateTimeout, diags := plan.Timeouts.Update(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	id, ok := parseID(state.ID, "Network", &resp.Diagnostics)
	if !ok {
		return
	}
	if _, err := r.client.UpdateNetwork(ctx, id, r.networkInput(ctx, plan, resolvedNetwork{})); err != nil {
		opError(&resp.Diagnostics, "Update", "Network", err)
		return
	}
	network, err := r.client.GetNetwork(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Read MTN Cloud Network Failed", err.Error())
		return
	}
	setNetworkComputed(&plan, network)
	resp.Diagnostics.Append(r.setNetworkLabelsAll(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteTimeout, diags := state.Timeouts.Delete(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	id, ok := parseID(state.ID, "Network", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteNetwork(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Network", err)
	}
}

func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

type resolvedNetwork struct {
	GroupName        string
	GroupID          int64
	CloudID          int64
	TypeID           *int64
	ResourcePoolID   *int64
	ResourcePoolName string
}

func (r *networkResource) resolveNetwork(ctx context.Context, plan networkResourceModel) (resolvedNetwork, error) {
	groupName := firstNonEmpty(plan.Group.ValueString(), r.defaults.Group)
	if groupName == "" {
		return resolvedNetwork{}, fmt.Errorf("`group` is required: set it on the resource or as a provider-level default")
	}
	group, err := r.client.GetGroupByName(ctx, groupName)
	if err != nil {
		return resolvedNetwork{}, err
	}
	if len(group.CloudIDs) == 0 {
		return resolvedNetwork{}, fmt.Errorf("group %q has no cloud IDs", group.Name)
	}
	resolved := resolvedNetwork{GroupName: groupName, GroupID: group.ID, CloudID: group.CloudIDs[0]}

	if !plan.Type.IsNull() && plan.Type.ValueString() != "" {
		networkType, err := r.client.GetNetworkTypeByName(ctx, plan.Type.ValueString(), false)
		if err != nil {
			return resolvedNetwork{}, err
		}
		resolved.TypeID = &networkType.ID
	}
	if poolName := firstNonEmpty(plan.ResourcePool.ValueString(), r.defaults.ResourcePool); poolName != "" {
		pool, err := r.client.GetResourcePool(ctx, poolName, group)
		if err != nil {
			return resolvedNetwork{}, err
		}
		resolved.ResourcePoolID = &pool.ID
		resolved.ResourcePoolName = firstNonEmpty(plan.ResourcePool.ValueString(), pool.Name, pool.Code)
	}
	return resolved, nil
}

func (r *networkResource) networkInput(ctx context.Context, plan networkResourceModel, resolved resolvedNetwork) client.NetworkInput {
	return client.NetworkInput{
		Name:                plan.Name.ValueString(),
		GroupID:             resolved.GroupID,
		CloudID:             resolved.CloudID,
		TypeID:              resolved.TypeID,
		ResourcePoolID:      resolved.ResourcePoolID,
		Description:         plan.Description.ValueString(),
		Labels:              mergeLabels(r.defaults.DefaultLabels, stringList(ctx, plan.Labels)),
		CIDR:                plan.CIDR.ValueString(),
		Gateway:             plan.Gateway.ValueString(),
		DNSPrimary:          plan.DNSPrimary.ValueString(),
		DNSSecondary:        plan.DNSSecondary.ValueString(),
		VlanID:              int64Ptr(plan.VlanID),
		DHCPServer:          boolPtr(plan.DHCPServer),
		AssignPublicIP:      boolPtr(plan.AssignPublicIP),
		AllowStaticOverride: boolPtr(plan.AllowStaticOverride),
		Active:              boolPtr(plan.Active),
		Visibility:          plan.Visibility.ValueString(),
	}
}

// setNetworkLabelsAll fills labels_all with the effective labels (provider
// default_labels merged with the resource's own labels).
func (r *networkResource) setNetworkLabelsAll(ctx context.Context, data *networkResourceModel) diag.Diagnostics {
	labels := mergeLabels(r.defaults.DefaultLabels, stringList(ctx, data.Labels))
	labelsAll, diags := types.ListValueFrom(ctx, types.StringType, labels)
	data.LabelsAll = labelsAll
	return diags
}

func setNetworkComputed(data *networkResourceModel, network *client.Network) {
	data.ID = types.StringValue(strconv.FormatInt(network.ID, 10))
	data.Code = optionalString(network.Code)
	data.Status = optionalString(network.Status)
	data.CloudID = int64ValueOrNull(nestedID(network.Zone))
	data.GroupID = int64ValueOrNull(nestedID(network.Site))
	data.TypeID = int64ValueOrNull(nestedID(network.Type))
	data.ResourcePoolID = int64ValueOrNull(nestedID(network.ZonePool))
}

func int64ValueOrNull(value int64) types.Int64 {
	if value == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(value)
}

// nestedID extracts the "id" field from a nested JSON object decoded into a
// map[string]any (numbers decode as float64).
func nestedID(m map[string]any) int64 {
	if m == nil {
		return 0
	}
	switch n := m["id"].(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	}
	return 0
}
