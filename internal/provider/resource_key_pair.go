package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &keyPairResource{}
var _ resource.ResourceWithConfigure = &keyPairResource{}
var _ resource.ResourceWithImportState = &keyPairResource{}

type keyPairResource struct {
	resourceBase
}

type keyPairResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	PublicKey  types.String `tfsdk:"public_key"`
	PrivateKey types.String `tfsdk:"private_key"`
	Passphrase types.String `tfsdk:"passphrase"`
}

func NewKeyPairResource() resource.Resource {
	return &keyPairResource{}
}

func (r *keyPairResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key_pair"
}

func (r *keyPairResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud SSH key pair. Key pairs are immutable; any change forces a new resource.",
		Attributes: map[string]rschema.Attribute{
			"id":         computedIDAttribute("Numeric identifier of the key pair."),
			"name":       rschema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Name of the key pair. Changing it forces a new key pair."},
			"public_key": rschema.StringAttribute{Required: true, PlanModifiers: replace, Description: "The public key material. Changing it forces a new key pair."},
			"private_key": rschema.StringAttribute{
				Optional:      true,
				Sensitive:     true,
				PlanModifiers: replace,
				Description:   "The private key material. Write-only; the API stores only a hash and never returns it. Changing it forces a new key pair.",
			},
			"passphrase": rschema.StringAttribute{
				Optional:      true,
				Sensitive:     true,
				PlanModifiers: replace,
				Description:   "Passphrase protecting the private key. Write-only. Changing it forces a new key pair.",
			},
		},
	}
}

func (r *keyPairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan keyPairResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	kp, err := r.client.CreateKeyPair(ctx, client.KeyPairInput{
		Name:       plan.Name.ValueString(),
		PublicKey:  plan.PublicKey.ValueString(),
		PrivateKey: plan.PrivateKey.ValueString(),
		Passphrase: plan.Passphrase.ValueString(),
	})
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Key Pair", err)
		return
	}
	// Keep private_key/passphrase as configured; the API never returns them.
	plan.ID = types.StringValue(strconv.FormatInt(kp.ID, 10))
	plan.Name = types.StringValue(kp.Name)
	plan.PublicKey = types.StringValue(kp.PublicKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *keyPairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state keyPairResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Key Pair", &resp.Diagnostics)
	if !ok {
		return
	}
	kp, err := r.client.GetKeyPair(ctx, id)
	if handleReadError(ctx, err, "Key Pair", &resp.State, &resp.Diagnostics) {
		return
	}
	state.Name = types.StringValue(kp.Name)
	state.PublicKey = types.StringValue(kp.PublicKey)
	// private_key/passphrase are write-only; leave prior state untouched.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is required by the interface but every settable attribute forces a
// replace, so this should never run with a real change.
func (r *keyPairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan keyPairResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *keyPairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state keyPairResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Key Pair", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteKeyPair(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Key Pair", err)
	}
}

func (r *keyPairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}
