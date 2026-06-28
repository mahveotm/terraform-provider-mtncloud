package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &networkDomainResource{}
var _ resource.ResourceWithConfigure = &networkDomainResource{}
var _ resource.ResourceWithImportState = &networkDomainResource{}

type networkDomainResource struct {
	resourceBase
}

type networkDomainResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	FQDN             types.String `tfsdk:"fqdn"`
	Visibility       types.String `tfsdk:"visibility"`
	Active           types.Bool   `tfsdk:"active"`
	PublicZone       types.Bool   `tfsdk:"public_zone"`
	DomainController types.Bool   `tfsdk:"domain_controller"`
	DomainUsername   types.String `tfsdk:"domain_username"`
	DomainPassword   types.String `tfsdk:"domain_password"`
}

func NewNetworkDomainResource() resource.Resource {
	return &networkDomainResource{}
}

func (r *networkDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_domain"
}

func (r *networkDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud network (DNS / Active Directory) domain.",
		Attributes: map[string]rschema.Attribute{
			"id":          computedIDAttribute("Numeric identifier of the network domain."),
			"name":        rschema.StringAttribute{Required: true, Description: "Name of the network domain."},
			"description": rschema.StringAttribute{Optional: true, Computed: true, Description: "Description of the network domain."},
			"fqdn":        rschema.StringAttribute{Optional: true, Computed: true, Description: "Fully qualified domain name."},
			"visibility": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("private"),
				Validators:  []validator.String{stringvalidator.OneOf("private", "public")},
				Description: "Visibility in sub-tenants: `private` or `public`. Defaults to `private`.",
			},
			"active":            rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the domain is active. Defaults to `true`."},
			"public_zone":       rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Whether this is a public DNS zone. Defaults to `false`."},
			"domain_controller": rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Whether this domain is joined to an Active Directory controller. Defaults to `false`."},
			"domain_username":   rschema.StringAttribute{Optional: true, Description: "Username for the Active Directory domain controller."},
			"domain_password":   rschema.StringAttribute{Optional: true, Sensitive: true, Description: "Password for the Active Directory domain controller. Write-only."},
		},
	}
}

func (r *networkDomainResource) input(plan networkDomainResourceModel) client.NetworkDomainInput {
	return client.NetworkDomainInput{
		Name:             plan.Name.ValueString(),
		Description:      plan.Description.ValueString(),
		FQDN:             plan.FQDN.ValueString(),
		Visibility:       plan.Visibility.ValueString(),
		Active:           boolPtr(plan.Active),
		PublicZone:       boolPtr(plan.PublicZone),
		DomainController: boolPtr(plan.DomainController),
		DomainUsername:   plan.DomainUsername.ValueString(),
		DomainPassword:   plan.DomainPassword.ValueString(),
	}
}

func (r *networkDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	domain, err := r.client.CreateNetworkDomain(ctx, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Network Domain", err)
		return
	}
	setNetworkDomainState(&plan, domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Network Domain", &resp.Diagnostics)
	if !ok {
		return
	}
	domain, err := r.client.GetNetworkDomain(ctx, id)
	if handleReadError(ctx, err, "Network Domain", &resp.State, &resp.Diagnostics) {
		return
	}
	setNetworkDomainState(&state, domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *networkDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan networkDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Network Domain", &resp.Diagnostics)
	if !ok {
		return
	}
	domain, err := r.client.UpdateNetworkDomain(ctx, id, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Network Domain", err)
		return
	}
	setNetworkDomainState(&plan, domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Network Domain", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteNetworkDomain(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Network Domain", err)
	}
}

func (r *networkDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func setNetworkDomainState(data *networkDomainResourceModel, domain *client.NetworkDomain) {
	data.ID = types.StringValue(strconv.FormatInt(domain.ID, 10))
	data.Name = types.StringValue(domain.Name)
	data.Description = mergeAPIString(data.Description, domain.Description)
	data.FQDN = mergeAPIString(data.FQDN, domain.FQDN)
	data.Visibility = mergeAPIString(data.Visibility, domain.Visibility)
	data.Active = mergeAPIBool(data.Active, domain.Active)
	data.PublicZone = mergeAPIBool(data.PublicZone, domain.PublicZone)
	data.DomainController = mergeAPIBool(data.DomainController, domain.DomainController)
	data.DomainUsername = mergeAPIString(data.DomainUsername, domain.DomainUsername)
	// domain_password is write-only; keep prior state.
}
