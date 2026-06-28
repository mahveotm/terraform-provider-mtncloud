package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &budgetResource{}
var _ resource.ResourceWithConfigure = &budgetResource{}
var _ resource.ResourceWithImportState = &budgetResource{}

type budgetResource struct {
	resourceBase
}

type budgetResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Scope       types.String `tfsdk:"scope"`
	Interval    types.String `tfsdk:"interval"`
	Year        types.String `tfsdk:"year"`
	Timezone    types.String `tfsdk:"timezone"`
	Currency    types.String `tfsdk:"currency"`
	Rollover    types.Bool   `tfsdk:"rollover"`
	Costs       []float64    `tfsdk:"costs"`
}

func NewBudgetResource() resource.Resource {
	return &budgetResource{}
}

func (r *budgetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_budget"
}

func (r *budgetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud cost budget over a yearly period at a chosen interval.",
		Attributes: map[string]rschema.Attribute{
			"id":   computedIDAttribute("Numeric identifier of the budget."),
			"name": rschema.StringAttribute{Required: true, Description: "Name of the budget."},
			"description": rschema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Description of the budget.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the budget is enabled. Defaults to `true`."},
			"scope": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("account"),
				Description: "Scope the budget applies to (e.g. `account`, `user`, `group`, `cloud`). Defaults to `account`.",
			},
			"interval": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("year"),
				Validators:  []validator.String{stringvalidator.OneOf("year", "quarter", "month")},
				Description: "Budget interval. One of `year` (1 cost), `quarter` (4 costs), `month` (12 costs). The length of `costs` must match. Defaults to `year`.",
			},
			"year": rschema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Calendar year the budget applies to, e.g. `2026`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"timezone": rschema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("UTC"), Description: "Timezone for the budget period. Defaults to `UTC`."},
			"currency": rschema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Currency code for the budget amounts. Set by MTN Cloud to the account currency (the budget API ignores any requested currency), so this is read-only.",
			},
			"rollover": rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Whether unused budget rolls over between intervals. Defaults to `false`."},
			"costs": rschema.ListAttribute{
				Required:    true,
				ElementType: types.Float64Type,
				Description: "Budget amounts per interval. Length must match `interval` (1/4/12).",
			},
		},
	}
}

func (r *budgetResource) input(plan budgetResourceModel) client.BudgetInput {
	return client.BudgetInput{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Enabled:     boolPtr(plan.Enabled),
		Scope:       plan.Scope.ValueString(),
		Interval:    plan.Interval.ValueString(),
		Year:        plan.Year.ValueString(),
		Timezone:    plan.Timezone.ValueString(),
		Currency:    plan.Currency.ValueString(),
		Rollover:    boolPtr(plan.Rollover),
		Costs:       plan.Costs,
	}
}

func (r *budgetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan budgetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	budget, err := r.client.CreateBudget(ctx, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Budget", err)
		return
	}
	budget = r.authoritative(ctx, budget)
	setBudgetState(&plan, budget)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// authoritative re-reads a budget via the single-object GET after a create or
// update. The MTN Cloud budget mutation (and list) responses echo a placeholder
// currency of "USD", while GET /budgets/{id} returns the real account currency.
// Re-reading keeps state consistent with what Read will see on the next refresh.
// On any read error the original mutation result is returned unchanged.
func (r *budgetResource) authoritative(ctx context.Context, budget *client.Budget) *client.Budget {
	if budget == nil {
		return budget
	}
	if fresh, err := r.client.GetBudget(ctx, budget.ID); err == nil {
		return fresh
	}
	return budget
}

func (r *budgetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state budgetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Budget", &resp.Diagnostics)
	if !ok {
		return
	}
	budget, err := r.client.GetBudget(ctx, id)
	if handleReadError(ctx, err, "Budget", &resp.State, &resp.Diagnostics) {
		return
	}
	setBudgetState(&state, budget)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *budgetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan budgetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Budget", &resp.Diagnostics)
	if !ok {
		return
	}
	budget, err := r.client.UpdateBudget(ctx, id, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Budget", err)
		return
	}
	budget = r.authoritative(ctx, budget)
	setBudgetState(&plan, budget)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *budgetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state budgetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Budget", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteBudget(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Budget", err)
	}
}

func (r *budgetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func setBudgetState(data *budgetResourceModel, budget *client.Budget) {
	data.ID = types.StringValue(strconv.FormatInt(budget.ID, 10))
	data.Name = types.StringValue(budget.Name)
	data.Description = mergeAPIString(data.Description, budget.Description)
	data.Enabled = mergeAPIBool(data.Enabled, budget.Enabled)
	data.Scope = mergeAPIString(data.Scope, budget.Scope)
	data.Interval = mergeAPIString(data.Interval, budget.Interval)
	data.Year = mergeAPIString(data.Year, budget.Year)
	data.Timezone = mergeAPIString(data.Timezone, budget.Timezone)
	data.Currency = mergeAPIString(data.Currency, budget.Currency)
	data.Rollover = mergeAPIBool(data.Rollover, budget.Rollover)
	if len(budget.Costs) > 0 {
		data.Costs = budget.Costs
	}
}
