package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &scaleThresholdResource{}
var _ resource.ResourceWithConfigure = &scaleThresholdResource{}
var _ resource.ResourceWithImportState = &scaleThresholdResource{}

type scaleThresholdResource struct {
	resourceBase
}

type scaleThresholdResourceModel struct {
	ID             types.String  `tfsdk:"id"`
	Name           types.String  `tfsdk:"name"`
	AutoUp         types.Bool    `tfsdk:"auto_upscale"`
	AutoDown       types.Bool    `tfsdk:"auto_downscale"`
	MinCount       types.Int64   `tfsdk:"min_count"`
	MaxCount       types.Int64   `tfsdk:"max_count"`
	ScaleIncrement types.Int64   `tfsdk:"scale_increment"`
	CPUEnabled     types.Bool    `tfsdk:"enable_cpu_threshold"`
	MinCPU         types.Float64 `tfsdk:"min_cpu_percentage"`
	MaxCPU         types.Float64 `tfsdk:"max_cpu_percentage"`
	MemoryEnabled  types.Bool    `tfsdk:"enable_memory_threshold"`
	MinMemory      types.Float64 `tfsdk:"min_memory_percentage"`
	MaxMemory      types.Float64 `tfsdk:"max_memory_percentage"`
	DiskEnabled    types.Bool    `tfsdk:"enable_disk_threshold"`
	MinDisk        types.Float64 `tfsdk:"min_disk_percentage"`
	MaxDisk        types.Float64 `tfsdk:"max_disk_percentage"`
}

func NewScaleThresholdResource() resource.Resource {
	return &scaleThresholdResource{}
}

func (r *scaleThresholdResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scale_threshold"
}

func (r *scaleThresholdResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud autoscale threshold (CPU/memory/disk based scaling rules).",
		Attributes: map[string]rschema.Attribute{
			"id":                      computedIDAttribute("Numeric identifier of the scale threshold."),
			"name":                    rschema.StringAttribute{Required: true, Description: "Name of the scale threshold."},
			"auto_upscale":            rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Whether to automatically scale up. Defaults to `false`."},
			"auto_downscale":          rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Whether to automatically scale down. Defaults to `false`."},
			"min_count":               rschema.Int64Attribute{Optional: true, Computed: true, Description: "Minimum number of instances."},
			"max_count":               rschema.Int64Attribute{Optional: true, Computed: true, Description: "Maximum number of instances."},
			"scale_increment":         rschema.Int64Attribute{Optional: true, Computed: true, Description: "Number of instances to add/remove per scaling action."},
			"enable_cpu_threshold":    rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Whether CPU-based scaling is enabled."},
			"min_cpu_percentage":      rschema.Float64Attribute{Optional: true, Computed: true, Description: "CPU percentage below which to scale down."},
			"max_cpu_percentage":      rschema.Float64Attribute{Optional: true, Computed: true, Description: "CPU percentage above which to scale up."},
			"enable_memory_threshold": rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Whether memory-based scaling is enabled."},
			"min_memory_percentage":   rschema.Float64Attribute{Optional: true, Computed: true, Description: "Memory percentage below which to scale down."},
			"max_memory_percentage":   rschema.Float64Attribute{Optional: true, Computed: true, Description: "Memory percentage above which to scale up."},
			"enable_disk_threshold":   rschema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Whether disk-based scaling is enabled."},
			"min_disk_percentage":     rschema.Float64Attribute{Optional: true, Computed: true, Description: "Disk percentage below which to scale down."},
			"max_disk_percentage":     rschema.Float64Attribute{Optional: true, Computed: true, Description: "Disk percentage above which to scale up."},
		},
	}
}

func (r *scaleThresholdResource) input(plan scaleThresholdResourceModel) client.ScaleThresholdInput {
	return client.ScaleThresholdInput{
		Name:           plan.Name.ValueString(),
		AutoUp:         boolPtr(plan.AutoUp),
		AutoDown:       boolPtr(plan.AutoDown),
		MinCount:       int64Ptr(plan.MinCount),
		MaxCount:       int64Ptr(plan.MaxCount),
		ScaleIncrement: int64Ptr(plan.ScaleIncrement),
		CPUEnabled:     boolPtr(plan.CPUEnabled),
		MinCPU:         float64Ptr(plan.MinCPU),
		MaxCPU:         float64Ptr(plan.MaxCPU),
		MemoryEnabled:  boolPtr(plan.MemoryEnabled),
		MinMemory:      float64Ptr(plan.MinMemory),
		MaxMemory:      float64Ptr(plan.MaxMemory),
		DiskEnabled:    boolPtr(plan.DiskEnabled),
		MinDisk:        float64Ptr(plan.MinDisk),
		MaxDisk:        float64Ptr(plan.MaxDisk),
	}
}

func (r *scaleThresholdResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scaleThresholdResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	st, err := r.client.CreateScaleThreshold(ctx, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Scale Threshold", err)
		return
	}
	setScaleThresholdState(&plan, st)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *scaleThresholdResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scaleThresholdResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Scale Threshold", &resp.Diagnostics)
	if !ok {
		return
	}
	st, err := r.client.GetScaleThreshold(ctx, id)
	if handleReadError(ctx, err, "Scale Threshold", &resp.State, &resp.Diagnostics) {
		return
	}
	setScaleThresholdState(&state, st)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *scaleThresholdResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scaleThresholdResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Scale Threshold", &resp.Diagnostics)
	if !ok {
		return
	}
	st, err := r.client.UpdateScaleThreshold(ctx, id, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Scale Threshold", err)
		return
	}
	setScaleThresholdState(&plan, st)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *scaleThresholdResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scaleThresholdResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Scale Threshold", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteScaleThreshold(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Scale Threshold", err)
	}
}

func (r *scaleThresholdResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func setScaleThresholdState(data *scaleThresholdResourceModel, st *client.ScaleThreshold) {
	data.ID = types.StringValue(strconv.FormatInt(st.ID, 10))
	data.Name = types.StringValue(st.Name)
	data.AutoUp = mergeAPIBool(data.AutoUp, st.AutoUp)
	data.AutoDown = mergeAPIBool(data.AutoDown, st.AutoDown)
	data.MinCount = mergeAPIInt64(data.MinCount, st.MinCount)
	data.MaxCount = mergeAPIInt64(data.MaxCount, st.MaxCount)
	data.ScaleIncrement = mergeAPIInt64(data.ScaleIncrement, st.ScaleIncrement)
	data.CPUEnabled = mergeAPIBool(data.CPUEnabled, st.CPUEnabled)
	data.MinCPU = mergeAPIFloat64(data.MinCPU, st.MinCPU)
	data.MaxCPU = mergeAPIFloat64(data.MaxCPU, st.MaxCPU)
	data.MemoryEnabled = mergeAPIBool(data.MemoryEnabled, st.MemoryEnabled)
	data.MinMemory = mergeAPIFloat64(data.MinMemory, st.MinMemory)
	data.MaxMemory = mergeAPIFloat64(data.MaxMemory, st.MaxMemory)
	data.DiskEnabled = mergeAPIBool(data.DiskEnabled, st.DiskEnabled)
	data.MinDisk = mergeAPIFloat64(data.MinDisk, st.MinDisk)
	data.MaxDisk = mergeAPIFloat64(data.MaxDisk, st.MaxDisk)
}
