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

var _ resource.Resource = &environmentResource{}
var _ resource.ResourceWithConfigure = &environmentResource{}
var _ resource.ResourceWithImportState = &environmentResource{}

type environmentResource struct {
	resourceBase
}

type environmentResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Code        types.String `tfsdk:"code"`
	Visibility  types.String `tfsdk:"visibility"`
	Active      types.Bool   `tfsdk:"active"`
}

func NewEnvironmentResource() resource.Resource {
	return &environmentResource{}
}

func (r *environmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *environmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud deployment environment (e.g. dev, staging, production).",
		Attributes: map[string]rschema.Attribute{
			"id":          computedIDAttribute("Numeric identifier of the environment."),
			"name":        rschema.StringAttribute{Required: true, Description: "Name of the environment."},
			"description": rschema.StringAttribute{Optional: true, Computed: true, Description: "Human-readable description of the environment."},
			"code":        rschema.StringAttribute{Optional: true, Computed: true, Description: "Short code identifying the environment."},
			"visibility": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("private"),
				Validators:  []validator.String{stringvalidator.OneOf("private", "public")},
				Description: "Whether the environment is visible to sub-tenants: `private` or `public`. Defaults to `private`.",
			},
			"active": rschema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the environment is enabled. Defaults to `true`.",
			},
		},
	}
}

func (r *environmentResource) input(plan environmentResourceModel) client.EnvironmentInput {
	return client.EnvironmentInput{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Code:        plan.Code.ValueString(),
		Visibility:  plan.Visibility.ValueString(),
		Active:      boolPtr(plan.Active),
	}
}

func (r *environmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	env, err := r.client.CreateEnvironment(ctx, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Environment", err)
		return
	}
	setEnvironmentState(&plan, env)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *environmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state environmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Environment", &resp.Diagnostics)
	if !ok {
		return
	}
	env, err := r.client.GetEnvironment(ctx, id)
	if handleReadError(ctx, err, "Environment", &resp.State, &resp.Diagnostics) {
		return
	}
	setEnvironmentState(&state, env)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Environment", &resp.Diagnostics)
	if !ok {
		return
	}
	env, err := r.client.UpdateEnvironment(ctx, id, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Environment", err)
		return
	}
	setEnvironmentState(&plan, env)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *environmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Environment", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteEnvironment(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Environment", err)
	}
}

func (r *environmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func setEnvironmentState(data *environmentResourceModel, env *client.Environment) {
	data.ID = types.StringValue(strconv.FormatInt(env.ID, 10))
	data.Name = types.StringValue(env.Name)
	data.Description = mergeAPIString(data.Description, env.Description)
	data.Code = mergeAPIString(data.Code, env.Code)
	data.Visibility = mergeAPIString(data.Visibility, env.Visibility)
	data.Active = mergeAPIBool(data.Active, env.Active)
}
