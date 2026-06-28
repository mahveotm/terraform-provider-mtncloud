package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &storageBucketResource{}
var _ resource.ResourceWithConfigure = &storageBucketResource{}
var _ resource.ResourceWithImportState = &storageBucketResource{}

type storageBucketResource struct {
	resourceBase
}

type storageBucketResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	BucketName                types.String `tfsdk:"bucket_name"`
	AccessKey                 types.String `tfsdk:"access_key"`
	SecretKey                 types.String `tfsdk:"secret_key"`
	Endpoint                  types.String `tfsdk:"endpoint"`
	StorageServer             types.Int64  `tfsdk:"storage_server"`
	CreateBucket              types.Bool   `tfsdk:"create_bucket"`
	DefaultBackupTarget       types.Bool   `tfsdk:"default_backup_target"`
	CopyToStore               types.Bool   `tfsdk:"copy_to_store"`
	DefaultDeploymentTarget   types.Bool   `tfsdk:"default_deployment_target"`
	DefaultVirtualImageTarget types.Bool   `tfsdk:"default_virtual_image_target"`
	RetentionPolicyType       types.String `tfsdk:"retention_policy_type"`
	RetentionPolicyDays       types.Int64  `tfsdk:"retention_policy_days"`
	RetentionProvider         types.String `tfsdk:"retention_provider"`
}

func NewStorageBucketResource() resource.Resource {
	return &storageBucketResource{}
}

func (r *storageBucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_bucket"
}

func (r *storageBucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud S3-compatible storage bucket.",
		Attributes: map[string]rschema.Attribute{
			"id":          computedIDAttribute("Numeric identifier of the storage bucket."),
			"name":        rschema.StringAttribute{Required: true, Description: "Unique storage bucket name in MTN Cloud."},
			"bucket_name": rschema.StringAttribute{Required: true, Description: "Backing S3 bucket name."},
			"access_key":  rschema.StringAttribute{Required: true, Sensitive: true, Description: "S3 access key."},
			"secret_key":  rschema.StringAttribute{Required: true, Sensitive: true, Description: "S3 secret key. The API never returns it, so changes here are applied but drift cannot be detected."},
			"endpoint":    rschema.StringAttribute{Required: true, Description: "S3-compatible endpoint URL."},
			"storage_server": rschema.Int64Attribute{
				Optional:    true,
				Description: "ID of the storage server backing this bucket.",
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"create_bucket": rschema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Create the backing bucket if it does not exist. Defaults to `true`.",
			},
			"default_backup_target":        rschema.BoolAttribute{Optional: true, Computed: true, Description: "Use this bucket as the default backup target."},
			"copy_to_store":                rschema.BoolAttribute{Optional: true, Computed: true, Description: "Copy backups to this store."},
			"default_deployment_target":    rschema.BoolAttribute{Optional: true, Computed: true, Description: "Use this bucket as the default deployment target."},
			"default_virtual_image_target": rschema.BoolAttribute{Optional: true, Computed: true, Description: "Use this bucket as the default virtual image target."},
			"retention_policy_type": rschema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Cleanup mode: `none`, `backup`, or `delete`.",
				Validators:  []validator.String{stringvalidator.OneOf("none", "backup", "delete")},
			},
			"retention_policy_days": rschema.Int64Attribute{
				Optional: true, Computed: true,
				Description: "Number of days to retain objects before the retention policy applies. Must be >= 1.",
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"retention_provider": rschema.StringAttribute{Optional: true, Computed: true, Description: "Provider used to enforce the retention policy."},
		},
	}
}

func (r *storageBucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan storageBucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	bucket, err := r.client.CreateStorageBucket(ctx, storageBucketInput(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Storage Bucket", err)
		return
	}
	setStorageBucketState(&plan, bucket)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageBucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state storageBucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Storage Bucket", &resp.Diagnostics)
	if !ok {
		return
	}
	bucket, err := r.client.GetStorageBucket(ctx, id)
	if handleReadError(ctx, err, "Storage Bucket", &resp.State, &resp.Diagnostics) {
		return
	}
	setStorageBucketState(&state, bucket)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *storageBucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan storageBucketResourceModel
	var state storageBucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Storage Bucket", &resp.Diagnostics)
	if !ok {
		return
	}
	bucket, err := r.client.UpdateStorageBucket(ctx, id, storageBucketInput(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Storage Bucket", err)
		return
	}
	setStorageBucketState(&plan, bucket)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *storageBucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state storageBucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Storage Bucket", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteStorageBucket(ctx, id, false); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Storage Bucket", err)
	}
}

func (r *storageBucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func storageBucketInput(plan storageBucketResourceModel) client.StorageBucketInput {
	return client.StorageBucketInput{
		Name:                      plan.Name.ValueString(),
		BucketName:                plan.BucketName.ValueString(),
		AccessKey:                 plan.AccessKey.ValueString(),
		SecretKey:                 plan.SecretKey.ValueString(),
		Endpoint:                  plan.Endpoint.ValueString(),
		StorageServer:             int64Ptr(plan.StorageServer),
		CreateBucket:              boolPtr(plan.CreateBucket),
		DefaultBackupTarget:       boolPtr(plan.DefaultBackupTarget),
		CopyToStore:               boolPtr(plan.CopyToStore),
		DefaultDeploymentTarget:   boolPtr(plan.DefaultDeploymentTarget),
		DefaultVirtualImageTarget: boolPtr(plan.DefaultVirtualImageTarget),
		RetentionPolicyType:       plan.RetentionPolicyType.ValueString(),
		RetentionPolicyDays:       int64Ptr(plan.RetentionPolicyDays),
		RetentionProvider:         plan.RetentionProvider.ValueString(),
	}
}

// setStorageBucketState maps observed values back to state. The API never
// returns the secret key, and create_bucket/storage_server are write-only
// directives, so those are preserved from the prior model value.
func setStorageBucketState(data *storageBucketResourceModel, bucket *client.StorageBucket) {
	data.ID = types.StringValue(strconv.FormatInt(bucket.ID, 10))
	data.Name = types.StringValue(bucket.Name)
	if bucket.BucketName != "" {
		data.BucketName = types.StringValue(bucket.BucketName)
	}
	if accessKey := configString(bucket.Config, "accessKey"); accessKey != "" {
		data.AccessKey = types.StringValue(accessKey)
	}
	if endpoint := configString(bucket.Config, "endpoint"); endpoint != "" {
		data.Endpoint = types.StringValue(endpoint)
	}
	data.DefaultBackupTarget = maybeBool(bucket.DefaultBackupTarget)
	data.CopyToStore = maybeBool(bucket.CopyToStore)
	data.DefaultDeploymentTarget = maybeBool(bucket.DefaultDeploymentTarget)
	data.DefaultVirtualImageTarget = maybeBool(bucket.DefaultVirtualImageTarget)
	data.RetentionPolicyType = optionalString(bucket.RetentionPolicyType)
	data.RetentionPolicyDays = anyToInt64(bucket.RetentionPolicyDays)
	data.RetentionProvider = optionalString(bucket.RetentionProvider)
}

func configString(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	if v, ok := config[key].(string); ok {
		return v
	}
	return ""
}

// anyToInt64 coerces a JSON value (number or numeric string) into a types.Int64.
func anyToInt64(value any) types.Int64 {
	switch n := value.(type) {
	case float64:
		return types.Int64Value(int64(n))
	case int64:
		return types.Int64Value(n)
	case int:
		return types.Int64Value(int64(n))
	case string:
		if n == "" {
			return types.Int64Null()
		}
		if parsed, err := strconv.ParseInt(n, 10, 64); err == nil {
			return types.Int64Value(parsed)
		}
	}
	return types.Int64Null()
}
