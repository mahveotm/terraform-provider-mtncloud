package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &ipPoolResource{}
var _ resource.ResourceWithConfigure = &ipPoolResource{}
var _ resource.ResourceWithImportState = &ipPoolResource{}

type ipPoolResource struct {
	resourceBase
}

type ipRangeModel struct {
	StartingAddress types.String `tfsdk:"starting_address"`
	EndingAddress   types.String `tfsdk:"ending_address"`
}

type ipPoolResourceModel struct {
	ID        types.String   `tfsdk:"id"`
	Name      types.String   `tfsdk:"name"`
	Gateway   types.String   `tfsdk:"gateway"`
	Netmask   types.String   `tfsdk:"netmask"`
	DNSDomain types.String   `tfsdk:"dns_domain"`
	IPRanges  []ipRangeModel `tfsdk:"ip_range"`
}

func NewIPPoolResource() resource.Resource {
	return &ipPoolResource{}
}

func (r *ipPoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ipv4_ip_pool"
}

func (r *ipPoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages a Morpheus-managed IPv4 address pool in MTN Cloud.",
		Attributes: map[string]rschema.Attribute{
			"id":         computedIDAttribute("Numeric identifier of the IP pool."),
			"name":       rschema.StringAttribute{Required: true, Description: "Name of the IP pool."},
			"gateway":    rschema.StringAttribute{Optional: true, Computed: true, Description: "Gateway IP address for the pool."},
			"netmask":    rschema.StringAttribute{Optional: true, Computed: true, Description: "Netmask for the pool."},
			"dns_domain": rschema.StringAttribute{Optional: true, Computed: true, Description: "DNS domain for the pool."},
			"ip_range": rschema.ListNestedAttribute{
				Required:    true,
				Description: "One or more contiguous IP address ranges in the pool.",
				NestedObject: rschema.NestedAttributeObject{
					Attributes: map[string]rschema.Attribute{
						"starting_address": rschema.StringAttribute{Required: true, Description: "First address of the range."},
						"ending_address":   rschema.StringAttribute{Required: true, Description: "Last address of the range."},
					},
				},
			},
		},
	}
}

func (r *ipPoolResource) input(plan ipPoolResourceModel) client.IPPoolInput {
	ranges := make([]client.IPRange, 0, len(plan.IPRanges))
	for _, rng := range plan.IPRanges {
		ranges = append(ranges, client.IPRange{
			StartAddress: rng.StartingAddress.ValueString(),
			EndAddress:   rng.EndingAddress.ValueString(),
		})
	}
	return client.IPPoolInput{
		Name:      plan.Name.ValueString(),
		Gateway:   plan.Gateway.ValueString(),
		Netmask:   plan.Netmask.ValueString(),
		DNSDomain: plan.DNSDomain.ValueString(),
		IPRanges:  ranges,
	}
}

func (r *ipPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipPoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pool, err := r.client.CreateIPPool(ctx, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "IP Pool", err)
		return
	}
	setIPPoolState(&plan, pool)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipPoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "IP Pool", &resp.Diagnostics)
	if !ok {
		return
	}
	pool, err := r.client.GetIPPool(ctx, id)
	if handleReadError(ctx, err, "IP Pool", &resp.State, &resp.Diagnostics) {
		return
	}
	setIPPoolState(&state, pool)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ipPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ipPoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "IP Pool", &resp.Diagnostics)
	if !ok {
		return
	}
	pool, err := r.client.UpdateIPPool(ctx, id, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "IP Pool", err)
		return
	}
	setIPPoolState(&plan, pool)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipPoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "IP Pool", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteIPPool(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "IP Pool", err)
	}
}

func (r *ipPoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func setIPPoolState(data *ipPoolResourceModel, pool *client.IPPool) {
	data.ID = types.StringValue(strconv.FormatInt(pool.ID, 10))
	data.Name = types.StringValue(pool.Name)
	data.Gateway = mergeAPIString(data.Gateway, pool.Gateway)
	data.Netmask = mergeAPIString(data.Netmask, pool.Netmask)
	data.DNSDomain = mergeAPIString(data.DNSDomain, pool.DNSDomain)
	if len(pool.IPRanges) > 0 {
		ranges := make([]ipRangeModel, 0, len(pool.IPRanges))
		for _, rng := range pool.IPRanges {
			ranges = append(ranges, ipRangeModel{
				StartingAddress: types.StringValue(rng.StartAddress),
				EndingAddress:   types.StringValue(rng.EndAddress),
			})
		}
		data.IPRanges = ranges
	}
}
