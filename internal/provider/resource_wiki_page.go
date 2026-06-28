package provider

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &wikiPageResource{}
var _ resource.ResourceWithConfigure = &wikiPageResource{}
var _ resource.ResourceWithImportState = &wikiPageResource{}

// trimTrailingNewlineModifier normalizes a string's planned value by stripping a
// single trailing newline, matching what the API stores. This keeps the plan in
// sync with the post-apply state so wiki content does not show a perpetual diff.
type trimTrailingNewlineModifier struct{}

func (m trimTrailingNewlineModifier) Description(_ context.Context) string {
	return "Strips a single trailing newline to match the stored value."
}

func (m trimTrailingNewlineModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m trimTrailingNewlineModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	resp.PlanValue = types.StringValue(strings.TrimSuffix(req.ConfigValue.ValueString(), "\n"))
}

type wikiPageResource struct {
	resourceBase
}

type wikiPageResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Category types.String `tfsdk:"category"`
	Content  types.String `tfsdk:"content"`
}

func NewWikiPageResource() resource.Resource {
	return &wikiPageResource{}
}

func (r *wikiPageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wiki_page"
}

func (r *wikiPageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud wiki page.",
		Attributes: map[string]rschema.Attribute{
			"id":       computedIDAttribute("Numeric identifier of the wiki page."),
			"name":     rschema.StringAttribute{Required: true, Description: "Name (title) of the wiki page."},
			"category": rschema.StringAttribute{Optional: true, Computed: true, Description: "Category the wiki page belongs to."},
			"content": rschema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{trimTrailingNewlineModifier{}},
				Description:   "Markdown content of the wiki page. A trailing newline is trimmed to match the stored value.",
			},
		},
	}
}

func (r *wikiPageResource) input(plan wikiPageResourceModel) client.WikiPageInput {
	return client.WikiPageInput{
		Name:     plan.Name.ValueString(),
		Category: plan.Category.ValueString(),
		Content:  plan.Content.ValueString(),
	}
}

func (r *wikiPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan wikiPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	page, err := r.client.CreateWikiPage(ctx, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Wiki Page", err)
		return
	}
	setWikiPageState(&plan, page)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wikiPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state wikiPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Wiki Page", &resp.Diagnostics)
	if !ok {
		return
	}
	page, err := r.client.GetWikiPage(ctx, id)
	if handleReadError(ctx, err, "Wiki Page", &resp.State, &resp.Diagnostics) {
		return
	}
	setWikiPageState(&state, page)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wikiPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan wikiPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Wiki Page", &resp.Diagnostics)
	if !ok {
		return
	}
	page, err := r.client.UpdateWikiPage(ctx, id, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Wiki Page", err)
		return
	}
	setWikiPageState(&plan, page)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wikiPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state wikiPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Wiki Page", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteWikiPage(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Wiki Page", err)
	}
}

func (r *wikiPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func setWikiPageState(data *wikiPageResourceModel, page *client.WikiPage) {
	data.ID = types.StringValue(strconv.FormatInt(page.ID, 10))
	data.Name = types.StringValue(page.Name)
	data.Category = mergeAPIString(data.Category, page.Category)
	// content stays Optional (not Computed); only overwrite when the API
	// returns a value so an unset content does not flip to a populated string.
	if page.Content != "" {
		data.Content = types.StringValue(page.Content)
	}
}
