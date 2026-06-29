package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &taskResource{}
var _ resource.ResourceWithConfigure = &taskResource{}
var _ resource.ResourceWithImportState = &taskResource{}
var _ resource.ResourceWithValidateConfig = &taskResource{}

type taskResource struct {
	resourceBase
}

type taskResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Code              types.String `tfsdk:"code"`
	Type              types.String `tfsdk:"type"`
	Labels            types.List   `tfsdk:"labels"`
	LabelsAll         types.List   `tfsdk:"labels_all"`
	ExecuteTarget     types.String `tfsdk:"execute_target"`
	ResultType        types.String `tfsdk:"result_type"`
	Retryable         types.Bool   `tfsdk:"retryable"`
	RetryCount        types.Int64  `tfsdk:"retry_count"`
	RetryDelaySeconds types.Int64  `tfsdk:"retry_delay_seconds"`
	ContinueOnError   types.Bool   `tfsdk:"continue_on_error"`
	AllowCustomConfig types.Bool   `tfsdk:"allow_custom_config"`
	Visibility        types.String `tfsdk:"visibility"`
	CredentialID      types.Int64  `tfsdk:"credential_id"`

	SourceType   types.String `tfsdk:"source_type"`
	Content      types.String `tfsdk:"content"`
	ContentPath  types.String `tfsdk:"content_path"`
	ContentRef   types.String `tfsdk:"content_ref"`
	RepositoryID types.Int64  `tfsdk:"repository_id"`

	Sudo     types.Bool   `tfsdk:"sudo"`
	Host     types.String `tfsdk:"host"`
	Port     types.String `tfsdk:"port"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Elevated types.Bool   `tfsdk:"elevated"`

	PythonBinary             types.String `tfsdk:"python_binary"`
	PythonArgs               types.String `tfsdk:"python_args"`
	PythonAdditionalPackages types.String `tfsdk:"python_additional_packages"`

	GitID    types.Int64  `tfsdk:"git_id"`
	GitRef   types.String `tfsdk:"git_ref"`
	Playbook types.String `tfsdk:"playbook"`
	Tags     types.String `tfsdk:"tags"`
	SkipTags types.String `tfsdk:"skip_tags"`
	Options  types.String `tfsdk:"options"`

	EmailAddress        types.String `tfsdk:"email_address"`
	Subject             types.String `tfsdk:"subject"`
	SkipWrappedTemplate types.Bool   `tfsdk:"skip_wrapped_template"`
}

func NewTaskResource() resource.Resource { return &taskResource{} }

func (r *taskResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_task"
}

func (r *taskResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	optStr := func(desc string) rschema.StringAttribute {
		return rschema.StringAttribute{Optional: true, Description: desc}
	}
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud automation task (script, playbook, email, or restart). " +
			"The `type` selects which fields apply; fields for other types are rejected at plan time. " +
			"Remote-exec `password` is write-only and never returned by the API. " +
			"Note: the MTN Cloud edge runs a WAF that inspects request bodies and may reject inline " +
			"script `content` containing shell commands (HTTP 403 \"Blocked By WAF\"); if so, source the " +
			"script from a repository or URL (`source_type = repository`/`url`), or have the WAF allow these payloads.",
		Attributes: map[string]rschema.Attribute{
			"id":   computedIDAttribute("Numeric identifier of the task."),
			"name": rschema.StringAttribute{Required: true, Description: "Name of the task."},
			"code": rschema.StringAttribute{Optional: true, Computed: true, Description: "User-defined code/identifier for the task."},
			"type": rschema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf(client.TaskTypes...)},
				Description:   "Task type. One of: " + joinQuoted(client.TaskTypes) + ". Changing it forces a new task.",
			},
			"labels": rschema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Labels applied to the task. Merged with the provider's default_labels into `labels_all`.",
			},
			"labels_all": rschema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Effective labels: the provider's default_labels merged (union) with `labels`.",
			},
			"execute_target": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.OneOf("local", "remote", "resource")},
				Description: "Where the task runs: `local`, `remote`, or `resource`.",
			},
			"result_type": rschema.StringAttribute{
				Optional:    true,
				Validators:  []validator.String{stringvalidator.OneOf("value", "keyValue", "json")},
				Description: "How script output is parsed (script types): `value`, `keyValue`, or `json`.",
			},
			"retryable":           rschema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the task retries on failure."},
			"retry_count":         rschema.Int64Attribute{Optional: true, Computed: true, Description: "Number of retries when `retryable`."},
			"retry_delay_seconds": rschema.Int64Attribute{Optional: true, Computed: true, Description: "Delay between retries, in seconds."},
			"continue_on_error":   rschema.BoolAttribute{Optional: true, Computed: true, Description: "Whether a workflow continues when this task fails."},
			"allow_custom_config": rschema.BoolAttribute{Optional: true, Computed: true, Description: "Whether custom config may be passed at execution."},
			"visibility": rschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Validators:  []validator.String{stringvalidator.OneOf("private", "public")},
				Description: "Task visibility: `private` or `public`.",
			},
			"credential_id": rschema.Int64Attribute{Optional: true, Description: "ID of a stored credential to use for remote execution (instead of inline username/password)."},

			"source_type":   rschema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.OneOf("local", "url", "repository")}, Description: "Script/template source: `local`, `url`, or `repository` (script and email types)."},
			"content":       optStr("Inline script/template content (when `source_type = local`)."),
			"content_path":  optStr("Source path or URL (when `source_type = url` or `repository`)."),
			"content_ref":   optStr("Git reference/branch (when `source_type = repository`)."),
			"repository_id": rschema.Int64Attribute{Optional: true, Description: "Integration/repository ID (when `source_type = repository`)."},

			"sudo":     rschema.BoolAttribute{Optional: true, Description: "Run the shell script with sudo (shell type)."},
			"host":     optStr("Remote host (shell/powershell with `execute_target = remote`)."),
			"port":     optStr("Remote port (shell/powershell with `execute_target = remote`)."),
			"username": optStr("Remote username (shell/powershell with `execute_target = remote`)."),
			"password": rschema.StringAttribute{Optional: true, Sensitive: true, Description: "Remote password (write-only; never returned by the API)."},
			"elevated": rschema.BoolAttribute{Optional: true, Description: "Run elevated (powershell type)."},

			"python_binary":              optStr("Python binary to use (python type)."),
			"python_args":                optStr("Command arguments passed to the script (python type)."),
			"python_additional_packages": optStr("Additional pip packages to install (python type)."),

			"git_id":    rschema.Int64Attribute{Optional: true, Description: "Ansible git integration ID (ansible type)."},
			"git_ref":   optStr("Ansible git reference/branch (ansible type)."),
			"playbook":  optStr("Ansible playbook to run (ansible type)."),
			"tags":      optStr("Ansible tags (ansible type)."),
			"skip_tags": optStr("Ansible skip-tags (ansible type)."),
			"options":   optStr("Additional ansible command options (ansible type)."),

			"email_address":         optStr("Recipient email address (email type)."),
			"subject":               optStr("Email subject (email type)."),
			"skip_wrapped_template": rschema.BoolAttribute{Optional: true, Description: "Skip the wrapped email template (email type)."},
		},
	}
}

func (r *taskResource) input(ctx context.Context, plan taskResourceModel) client.TaskInput {
	return client.TaskInput{
		Type:                     plan.Type.ValueString(),
		Name:                     plan.Name.ValueString(),
		Code:                     plan.Code.ValueString(),
		Labels:                   mergeLabels(r.defaults.DefaultLabels, stringList(ctx, plan.Labels)),
		ExecuteTarget:            plan.ExecuteTarget.ValueString(),
		ResultType:               plan.ResultType.ValueString(),
		Retryable:                boolPtr(plan.Retryable),
		RetryCount:               int64Ptr(plan.RetryCount),
		RetryDelaySeconds:        int64Ptr(plan.RetryDelaySeconds),
		ContinueOnError:          boolPtr(plan.ContinueOnError),
		AllowCustomConfig:        boolPtr(plan.AllowCustomConfig),
		Visibility:               plan.Visibility.ValueString(),
		CredentialID:             int64Ptr(plan.CredentialID),
		SourceType:               plan.SourceType.ValueString(),
		Content:                  plan.Content.ValueString(),
		ContentPath:              plan.ContentPath.ValueString(),
		ContentRef:               plan.ContentRef.ValueString(),
		RepositoryID:             int64Ptr(plan.RepositoryID),
		Sudo:                     plan.Sudo.ValueBool(),
		Host:                     plan.Host.ValueString(),
		Port:                     plan.Port.ValueString(),
		Username:                 plan.Username.ValueString(),
		Password:                 plan.Password.ValueString(),
		Elevated:                 plan.Elevated.ValueBool(),
		PythonBinary:             plan.PythonBinary.ValueString(),
		PythonArgs:               plan.PythonArgs.ValueString(),
		PythonAdditionalPackages: plan.PythonAdditionalPackages.ValueString(),
		GitID:                    int64Ptr(plan.GitID),
		GitRef:                   plan.GitRef.ValueString(),
		Playbook:                 plan.Playbook.ValueString(),
		Tags:                     plan.Tags.ValueString(),
		SkipTags:                 plan.SkipTags.ValueString(),
		Options:                  plan.Options.ValueString(),
		EmailAddress:             plan.EmailAddress.ValueString(),
		Subject:                  plan.Subject.ValueString(),
		SkipWrappedTemplate:      plan.SkipWrappedTemplate.ValueBool(),
	}
}

func (r *taskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan taskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	task, err := r.client.CreateTask(ctx, r.input(ctx, plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Task", err)
		return
	}
	setTaskState(&plan, task)
	resp.Diagnostics.Append(r.setTaskLabelsAll(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *taskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state taskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Task", &resp.Diagnostics)
	if !ok {
		return
	}
	task, err := r.client.GetTask(ctx, id)
	if handleReadError(ctx, err, "Task", &resp.State, &resp.Diagnostics) {
		return
	}
	setTaskState(&state, task)
	resp.Diagnostics.Append(r.setTaskLabelsAll(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *taskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan taskResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Task", &resp.Diagnostics)
	if !ok {
		return
	}
	task, err := r.client.UpdateTask(ctx, id, r.input(ctx, plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Task", err)
		return
	}
	setTaskState(&plan, task)
	resp.Diagnostics.Append(r.setTaskLabelsAll(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *taskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state taskResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Task", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteTask(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Task", err)
	}
}

func (r *taskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

// setTaskLabelsAll fills labels_all with the effective labels (provider
// default_labels merged with the resource's own labels).
func (r *taskResource) setTaskLabelsAll(ctx context.Context, data *taskResourceModel) diag.Diagnostics {
	labels := mergeLabels(r.defaults.DefaultLabels, stringList(ctx, data.Labels))
	labelsAll, diags := types.ListValueFrom(ctx, types.StringType, labels)
	data.LabelsAll = labelsAll
	return diags
}

// setTaskState reconciles only stable metadata; script content, type-specific
// options, and the write-only remote password are kept from prior state.
func setTaskState(data *taskResourceModel, task *client.Task) {
	data.ID = types.StringValue(strconv.FormatInt(task.ID, 10))
	data.Name = types.StringValue(task.Name)
	data.Code = mergeAPIString(data.Code, task.Code)
	if friendly := client.TaskTypeFromCode(task.TaskType.Code); friendly != "" {
		data.Type = types.StringValue(friendly)
	}
	data.ExecuteTarget = mergeAPIString(data.ExecuteTarget, task.ExecuteTarget)
	data.ResultType = mergeAPIString(data.ResultType, task.ResultType)
	data.Retryable = mergeAPIBool(data.Retryable, task.Retryable)
	data.RetryCount = mergeAPIInt64(data.RetryCount, task.RetryCount)
	data.RetryDelaySeconds = mergeAPIInt64(data.RetryDelaySeconds, task.RetryDelaySeconds)
	data.ContinueOnError = mergeAPIBool(data.ContinueOnError, task.ContinueOnError)
	data.AllowCustomConfig = mergeAPIBool(data.AllowCustomConfig, task.AllowCustomConfig)
	data.Visibility = mergeAPIString(data.Visibility, task.Visibility)
}
