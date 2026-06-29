package provider

import (
	"context"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &workflowResource{}
var _ resource.ResourceWithConfigure = &workflowResource{}
var _ resource.ResourceWithImportState = &workflowResource{}
var _ resource.ResourceWithValidateConfig = &workflowResource{}

type workflowResource struct {
	resourceBase
}

type workflowResourceModel struct {
	ID                types.String        `tfsdk:"id"`
	Name              types.String        `tfsdk:"name"`
	Description       types.String        `tfsdk:"description"`
	Type              types.String        `tfsdk:"type"`
	Labels            types.List          `tfsdk:"labels"`
	LabelsAll         types.List          `tfsdk:"labels_all"`
	Visibility        types.String        `tfsdk:"visibility"`
	Platform          types.String        `tfsdk:"platform"`
	AllowCustomConfig types.Bool          `tfsdk:"allow_custom_config"`
	Tasks             []workflowTaskModel `tfsdk:"task"`
}

type workflowTaskModel struct {
	TaskID types.Int64  `tfsdk:"task_id"`
	Phase  types.String `tfsdk:"phase"`
}

func NewWorkflowResource() resource.Resource { return &workflowResource{} }

func (r *workflowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *workflowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud workflow (task-set): an ordered set of tasks run together, " +
			"either operational (`type = operation`) or provisioning (`type = provision`).",
		Attributes: map[string]rschema.Attribute{
			"id":          computedIDAttribute("Numeric identifier of the workflow."),
			"name":        rschema.StringAttribute{Required: true, Description: "Name of the workflow."},
			"description": rschema.StringAttribute{Optional: true, Computed: true, Description: "Description of the workflow."},
			"type": rschema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf(client.WorkflowTypes...)},
				Description:   "Workflow type: " + joinQuoted(client.WorkflowTypes) + ". Changing it forces a new workflow.",
			},
			"labels": rschema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Labels applied to the workflow. Merged with the provider's default_labels into `labels_all`.",
			},
			"labels_all": rschema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Effective labels: the provider's default_labels merged (union) with `labels`.",
			},
			"visibility": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.OneOf("private", "public")},
				Description: "Workflow visibility: `private` or `public`.",
			},
			"platform": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.OneOf(client.WorkflowPlatforms...)},
				Description: "Platform filter: " + joinQuoted(client.WorkflowPlatforms) + ".",
			},
			"allow_custom_config": rschema.BoolAttribute{Optional: true, Computed: true, Description: "Whether custom config may be passed at execution."},
		},
		Blocks: map[string]rschema.Block{
			"task": rschema.ListNestedBlock{
				Description: "Ordered member tasks. Order is preserved as the execution order.",
				NestedObject: rschema.NestedBlockObject{
					Attributes: map[string]rschema.Attribute{
						"task_id": rschema.Int64Attribute{Required: true, Description: "ID of the task to include."},
						"phase": rschema.StringAttribute{
							Required: true,
							Description: "Phase the task runs in. For `operation` workflows this is `operation`; " +
								"for `provision` workflows one of: " + joinQuoted(client.ProvisionPhases) + ".",
						},
					},
				},
			},
		},
	}
}

func (r *workflowResource) input(ctx context.Context, plan workflowResourceModel) client.WorkflowInput {
	tasks := make([]client.WorkflowTask, 0, len(plan.Tasks))
	for _, t := range plan.Tasks {
		tasks = append(tasks, client.WorkflowTask{TaskID: t.TaskID.ValueInt64(), Phase: t.Phase.ValueString()})
	}
	return client.WorkflowInput{
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		Type:              plan.Type.ValueString(),
		Labels:            mergeLabels(r.defaults.DefaultLabels, stringList(ctx, plan.Labels)),
		Visibility:        plan.Visibility.ValueString(),
		Platform:          plan.Platform.ValueString(),
		AllowCustomConfig: boolPtr(plan.AllowCustomConfig),
		Tasks:             tasks,
	}
}

func (r *workflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	wf, err := r.client.CreateWorkflow(ctx, r.input(ctx, plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Workflow", err)
		return
	}
	setWorkflowState(&plan, wf)
	resp.Diagnostics.Append(r.setWorkflowLabelsAll(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *workflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Workflow", &resp.Diagnostics)
	if !ok {
		return
	}
	wf, err := r.client.GetWorkflow(ctx, id)
	if handleReadError(ctx, err, "Workflow", &resp.State, &resp.Diagnostics) {
		return
	}
	setWorkflowState(&state, wf)
	resp.Diagnostics.Append(r.setWorkflowLabelsAll(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *workflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Workflow", &resp.Diagnostics)
	if !ok {
		return
	}
	wf, err := r.client.UpdateWorkflow(ctx, id, r.input(ctx, plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Workflow", err)
		return
	}
	setWorkflowState(&plan, wf)
	resp.Diagnostics.Append(r.setWorkflowLabelsAll(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *workflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Workflow", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteWorkflow(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Workflow", err)
	}
}

func (r *workflowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func (r *workflowResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg workflowResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() || !attrSet(cfg.Type) {
		return
	}
	tt := cfg.Type.ValueString()
	for i, blk := range cfg.Tasks {
		if !attrSet(blk.Phase) {
			continue
		}
		phase := blk.Phase.ValueString()
		p := path.Root("task").AtListIndex(i).AtName("phase")
		switch tt {
		case "operation":
			if phase != client.OperationalPhase {
				resp.Diagnostics.AddAttributeError(p, "Invalid Workflow Phase",
					"`phase` must be `"+client.OperationalPhase+"` for an operational workflow.")
			}
		case "provision":
			if !containsStr(client.ProvisionPhases, phase) {
				resp.Diagnostics.AddAttributeError(p, "Invalid Workflow Phase",
					"`phase` for a provisioning workflow must be one of: "+joinQuoted(client.ProvisionPhases)+".")
			}
		}
	}
}

func (r *workflowResource) setWorkflowLabelsAll(ctx context.Context, data *workflowResourceModel) diag.Diagnostics {
	labels := mergeLabels(r.defaults.DefaultLabels, stringList(ctx, data.Labels))
	labelsAll, diags := types.ListValueFrom(ctx, types.StringType, labels)
	data.LabelsAll = labelsAll
	return diags
}

func setWorkflowState(data *workflowResourceModel, wf *client.Workflow) {
	data.ID = types.StringValue(strconv.FormatInt(wf.ID, 10))
	data.Name = types.StringValue(wf.Name)
	data.Description = mergeAPIString(data.Description, wf.Description)
	if wf.Type != "" {
		data.Type = types.StringValue(wf.Type)
	}
	data.Visibility = mergeAPIString(data.Visibility, wf.Visibility)
	data.Platform = mergeAPIString(data.Platform, wf.Platform)
	data.AllowCustomConfig = mergeAPIBool(data.AllowCustomConfig, wf.AllowCustomConfig)

	// The member list is returned reliably; reflect it (ordered) for real drift
	// detection. Keep the configured list if the response omitted it (create lag).
	if len(wf.TaskSetTasks) > 0 {
		tasks := append([]client.TaskSetTask(nil), wf.TaskSetTasks...)
		sort.Slice(tasks, func(i, j int) bool { return tasks[i].TaskOrder < tasks[j].TaskOrder })
		out := make([]workflowTaskModel, 0, len(tasks))
		for _, t := range tasks {
			out = append(out, workflowTaskModel{
				TaskID: types.Int64Value(t.Task.ID),
				Phase:  types.StringValue(t.TaskPhase),
			})
		}
		data.Tasks = out
	}
}
