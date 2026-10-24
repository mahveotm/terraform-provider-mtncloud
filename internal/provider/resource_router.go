package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Type            types.String `tfsdk:"type"`
	Group           types.String `tfsdk:"group"`
	CloudID         types.Int64  `tfsdk:"cloud_id"`
	NetworkServerID types.Int64  `tfsdk:"network_server_id"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	OptionsJSON     types.String `tfsdk:"options_json"`
	Status          types.String `tfsdk:"status"`
	ExternalIP      types.String `tfsdk:"external_ip"`
}

func NewRouterResource() resource.Resource { return &routerResource{} }
func (r *routerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_router"
}
func (r *routerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = rschema.Schema{
		Description: "Manages a Morpheus network router. Type-specific options depend on the router types available in your MTN Cloud account. Creation records the router ID without waiting for backend provisioning to finish.",
		Attributes: map[string]rschema.Attribute{
			"id":                computedIDAttribute("Numeric router ID."),
			"name":              rschema.StringAttribute{Required: true, Description: "Router name."},
			"type":              rschema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Router type name or code from mtncloud_router_types. Changes replace the router."},
			"group":             rschema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}, Description: "Group name; defaults to the provider group. Changes replace the router. Shared routers are not supported."},
			"cloud_id":          rschema.Int64Attribute{Optional: true, Computed: true, Validators: []validator.Int64{int64validator.AtLeast(1)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace(), int64planmodifier.UseStateForUnknown()}, Description: "Cloud ID for types without a network server. Inferred only when the group has exactly one cloud. Changes replace the router."},
			"network_server_id": rschema.Int64Attribute{Optional: true, Validators: []validator.Int64{int64validator.AtLeast(1)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}, Description: "Required for router types with hasNetworkServer=true; conflicts with cloud_id. Changes replace the router."},
			"enabled":           rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the router is enabled."},
			"options_json":      rschema.StringAttribute{Optional: true, Sensitive: true, PlanModifiers: replace, Description: "JSON object of additional fields inside networkRouter, based on the selected type's optionTypes (for example config or interfaces). Cannot override core attributes. Sent only on create; changes replace the router. Remote drift is not tracked and import cannot recover this value. Sensitive values remain in Terraform state."},
			"status":            rschema.StringAttribute{Computed: true, Description: "Observed router status; creation does not wait for readiness."},
			"external_ip":       rschema.StringAttribute{Computed: true, Description: "External IP, if assigned."},
		},
	}
}
func (r *routerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data routerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !data.OptionsJSON.IsUnknown() {
		if _, err := client.RouterOptions(data.OptionsJSON.ValueString()); err != nil {
			resp.Diagnostics.AddError("Invalid Router Options", err.Error())
		}
	}
	if !data.CloudID.IsNull() && !data.NetworkServerID.IsNull() {
		resp.Diagnostics.AddError("Conflicting Router Scope", "Set cloud_id or network_server_id, not both.")
	}
}
func (r *routerResource) createFields(ctx context.Context, data *routerModel) (map[string]any, error) {
	fields, err := client.RouterOptions(data.OptionsJSON.ValueString())
	if err != nil {
		return nil, err
	}
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
	routerType, err := r.client.GetRouterTypeByName(ctx, data.Type.ValueString())
	if err != nil {
		return nil, err
	}
	if !routerType.Creatable {
		return nil, fmt.Errorf("router type %q does not support creation", data.Type.ValueString())
	}
	fields["name"] = data.Name.ValueString()
	fields["type"] = map[string]any{"id": routerType.ID}
	fields["site"] = map[string]any{"id": group.ID}
	fields["enabled"] = data.Enabled.ValueBool()
	data.Group = types.StringValue(groupName)
	if routerType.HasNetworkServer {
		if data.NetworkServerID.IsNull() || data.NetworkServerID.ValueInt64() < 1 {
			return nil, fmt.Errorf("this router type requires network_server_id")
		}
		if !data.CloudID.IsNull() && !data.CloudID.IsUnknown() {
			return nil, fmt.Errorf("this router type requires network_server_id instead of cloud_id")
		}
		fields["networkServer"] = map[string]any{"id": data.NetworkServerID.ValueInt64()}
		data.CloudID = types.Int64Null()
	} else {
		if !data.NetworkServerID.IsNull() {
			return nil, fmt.Errorf("this router type requires a cloud, not network_server_id")
		}
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
	router, err := r.client.GetRouter(ctx, id)
	if err != nil {
		opError(&resp.Diagnostics, "Read Created", "Router", err)
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
	if router.NetworkServer.ID > 0 {
		data.NetworkServerID = types.Int64Value(router.NetworkServer.ID)
		data.CloudID = types.Int64Null()
	} else {
		data.NetworkServerID = types.Int64Null()
		if router.Zone.ID > 0 {
			data.CloudID = types.Int64Value(router.Zone.ID)
		}
	}
	data.Status = optionalString(router.Status)
	data.ExternalIP = optionalString(router.ExternalIP)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *routerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data routerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(data.ID, "Router", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.UpdateRouter(ctx, id, data.Name.ValueString(), data.Enabled.ValueBool()); err != nil {
		opError(&resp.Diagnostics, "Update", "Router", err)
		return
	}
	data.Status = types.StringNull()
	data.ExternalIP = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	router, err := r.client.GetRouter(ctx, id)
	if err != nil {
		opError(&resp.Diagnostics, "Read Updated", "Router", err)
		return
	}
	data.Status = optionalString(router.Status)
	data.ExternalIP = optionalString(router.ExternalIP)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
	if err := r.client.DeleteRouter(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Router", err)
	}
}
func (r *routerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}
