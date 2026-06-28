package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &cypherSecretResource{}
var _ resource.ResourceWithConfigure = &cypherSecretResource{}
var _ resource.ResourceWithImportState = &cypherSecretResource{}

type cypherSecretResource struct {
	resourceBase
}

type cypherSecretResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
	TTL   types.Int64  `tfsdk:"ttl"`
}

func NewCypherSecretResource() resource.Resource {
	return &cypherSecretResource{}
}

func (r *cypherSecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cypher_secret"
}

func (r *cypherSecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceString := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = rschema.Schema{
		Description: "Manages a secret in the MTN Cloud Cypher store (under the `secret/` mount). Secrets are immutable; any change forces a new resource.",
		Attributes: map[string]rschema.Attribute{
			"id":  computedIDAttribute("Numeric identifier of the cypher entry."),
			"key": rschema.StringAttribute{Required: true, PlanModifiers: replaceString, Description: "The secret path under the `secret/` mount, e.g. `myapp/db-password`. Changing it forces a new secret."},
			"value": rschema.StringAttribute{
				Required:      true,
				Sensitive:     true,
				PlanModifiers: replaceString,
				Description:   "The secret value. Changing it forces a new secret.",
			},
			"ttl": rschema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace(), int64planmodifier.UseStateForUnknown()},
				Description:   "Time-to-live of the secret lease in seconds. 0 means no expiry. Changing it forces a new secret.",
			},
		},
	}
}

func (r *cypherSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cypherSecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.CreateCypher(ctx, plan.Key.ValueString(), plan.Value.ValueString(), int64Ptr(plan.TTL))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Cypher Secret", err)
		return
	}
	plan.ID = types.StringValue(strconv.FormatInt(result.Cypher.ID, 10))
	plan.TTL = mergeAPIInt64(plan.TTL, result.LeaseDuration)
	// Keep value as configured; the write response does not echo it back.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cypherSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cypherSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.GetCypher(ctx, state.Key.ValueString())
	if handleReadError(ctx, err, "Cypher Secret", &resp.State, &resp.Diagnostics) {
		return
	}
	state.ID = types.StringValue(strconv.FormatInt(result.Cypher.ID, 10))
	state.TTL = mergeAPIInt64(state.TTL, result.LeaseDuration)
	// value is kept from prior state (write-only, RequiresReplace).
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but every attribute forces a replace.
func (r *cypherSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cypherSecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cypherSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cypherSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCypher(ctx, state.Key.ValueString()); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Cypher Secret", err)
	}
}

// ImportState imports by the secret key path (not the numeric id), since the key
// is what addresses the secret in the Cypher store.
func (r *cypherSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("key"), req, resp)
}
