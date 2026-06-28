package provider

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &credentialResource{}
var _ resource.ResourceWithConfigure = &credentialResource{}
var _ resource.ResourceWithImportState = &credentialResource{}

type credentialResource struct {
	resourceBase
}

type credentialResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Type         types.String `tfsdk:"type"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	AccessKey    types.String `tfsdk:"access_key"`
	SecretKey    types.String `tfsdk:"secret_key"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Email        types.String `tfsdk:"email"`
	APIKey       types.String `tfsdk:"api_key"`
	Tenant       types.String `tfsdk:"tenant"`
	KeyPairID    types.Int64  `tfsdk:"key_pair_id"`
}

func NewCredentialResource() resource.Resource {
	return &credentialResource{}
}

func joinQuoted(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = "`" + v + "`"
	}
	return strings.Join(quoted, ", ")
}

func (r *credentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_credential"
}

func (r *credentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an entry in the MTN Cloud credential store. Secret fields are write-only; the API never returns them.",
		Attributes: map[string]rschema.Attribute{
			"id":   computedIDAttribute("Numeric identifier of the credential."),
			"name": rschema.StringAttribute{Required: true, Description: "Name of the credential."},
			"type": rschema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf(client.CredentialTypes...)},
				Description:   "Credential type. One of: " + joinQuoted(client.CredentialTypes) + ". Changing it forces a new credential.",
			},
			"description":   rschema.StringAttribute{Optional: true, Computed: true, Description: "Description of the credential."},
			"enabled":       rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the credential is enabled. Defaults to `true`."},
			"access_key":    rschema.StringAttribute{Optional: true, Description: "Access key (type `access-key-secret`)."},
			"secret_key":    rschema.StringAttribute{Optional: true, Sensitive: true, Description: "Secret key (type `access-key-secret`)."},
			"username":      rschema.StringAttribute{Optional: true, Description: "Username (username-* and tenant-username-keypair types)."},
			"password":      rschema.StringAttribute{Optional: true, Sensitive: true, Description: "Password (username-password* types)."},
			"client_id":     rschema.StringAttribute{Optional: true, Description: "Client ID (type `client-id-secret`)."},
			"client_secret": rschema.StringAttribute{Optional: true, Sensitive: true, Description: "Client secret (type `client-id-secret`)."},
			"email":         rschema.StringAttribute{Optional: true, Description: "Email (type `email-private-key`)."},
			"api_key":       rschema.StringAttribute{Optional: true, Sensitive: true, Description: "API key (type `api-key` / `username-api-key`)."},
			"tenant":        rschema.StringAttribute{Optional: true, Description: "Tenant / auth path (type `tenant-username-keypair`)."},
			"key_pair_id":   rschema.Int64Attribute{Optional: true, Description: "Key pair ID (keypair-based types)."},
		},
	}
}

func (r *credentialResource) input(plan credentialResourceModel) client.CredentialInput {
	return client.CredentialInput{
		Type:         plan.Type.ValueString(),
		Name:         plan.Name.ValueString(),
		Description:  plan.Description.ValueString(),
		Enabled:      boolPtr(plan.Enabled),
		AccessKey:    plan.AccessKey.ValueString(),
		SecretKey:    plan.SecretKey.ValueString(),
		Username:     plan.Username.ValueString(),
		Password:     plan.Password.ValueString(),
		ClientID:     plan.ClientID.ValueString(),
		ClientSecret: plan.ClientSecret.ValueString(),
		Email:        plan.Email.ValueString(),
		APIKey:       plan.APIKey.ValueString(),
		Tenant:       plan.Tenant.ValueString(),
		KeyPairID:    int64Ptr(plan.KeyPairID),
	}
}

func (r *credentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan credentialResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cred, err := r.client.CreateCredential(ctx, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Credential", err)
		return
	}
	setCredentialState(&plan, cred)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *credentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state credentialResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Credential", &resp.Diagnostics)
	if !ok {
		return
	}
	cred, err := r.client.GetCredential(ctx, id)
	if handleReadError(ctx, err, "Credential", &resp.State, &resp.Diagnostics) {
		return
	}
	setCredentialState(&state, cred)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *credentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan credentialResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Credential", &resp.Diagnostics)
	if !ok {
		return
	}
	cred, err := r.client.UpdateCredential(ctx, id, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Credential", err)
		return
	}
	setCredentialState(&plan, cred)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *credentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state credentialResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Credential", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteCredential(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Credential", err)
	}
}

func (r *credentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

// setCredentialState reconciles only non-secret metadata; all credential material
// (access_key, secret_key, password, etc.) is write-only and kept from prior state.
func setCredentialState(data *credentialResourceModel, cred *client.Credential) {
	data.ID = types.StringValue(strconv.FormatInt(cred.ID, 10))
	data.Name = types.StringValue(cred.Name)
	data.Description = mergeAPIString(data.Description, cred.Description)
	data.Enabled = mergeAPIBool(data.Enabled, cred.Enabled)
	if cred.Type.Code != "" {
		data.Type = types.StringValue(cred.Type.Code)
	}
}
