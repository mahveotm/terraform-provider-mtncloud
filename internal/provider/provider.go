package provider

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	pschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ provider.Provider = &mtnCloudProvider{}

type mtnCloudProvider struct {
	version string
}

type providerModel struct {
	URL              types.String `tfsdk:"url"`
	Token            types.String `tfsdk:"token"`
	Username         types.String `tfsdk:"username"`
	Password         types.String `tfsdk:"password"`
	Timeout          types.Int64  `tfsdk:"timeout"`
	Insecure         types.Bool   `tfsdk:"insecure"`
	Group            types.String `tfsdk:"group"`
	ResourcePool     types.String `tfsdk:"resource_pool"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	DefaultLabels    types.List   `tfsdk:"default_labels"`
	DefaultTags      types.Map    `tfsdk:"default_tags"`
}

// mtnCloudProviderData is shared with every resource and data source. It carries
// the API client plus provider-level defaults that resources inherit when their
// own attribute is unset (resource values always win).
type mtnCloudProviderData struct {
	Client           *client.Client
	Group            string
	ResourcePool     string
	AvailabilityZone string
	DefaultLabels    []string
	DefaultTags      map[string]string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &mtnCloudProvider{version: version}
	}
}

func (p *mtnCloudProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "mtncloud"
	resp.Version = p.version
}

func (p *mtnCloudProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = pschema.Schema{
		Description: "The MTN Cloud provider manages MTN Cloud (Morpheus-based) infrastructure: " +
			"compute instances, networks, security groups, and storage/archive buckets. " +
			"Configure it with an API token or username/password, and optionally set a " +
			"default group, resource pool, and tags/labels that resources inherit.",
		Attributes: map[string]pschema.Attribute{
			"url": pschema.StringAttribute{
				Optional:    true,
				Description: "MTN Cloud console URL. May also be set with MTN_CLOUD_URL.",
			},
			"token": pschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "MTN Cloud API token. May also be set with MTN_CLOUD_TOKEN.",
			},
			"username": pschema.StringAttribute{
				Optional:    true,
				Description: "MTN Cloud username for password authentication. May also be set with MTN_CLOUD_USERNAME.",
			},
			"password": pschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "MTN Cloud password for password authentication. May also be set with MTN_CLOUD_PASSWORD.",
			},
			"timeout": pschema.Int64Attribute{
				Optional:    true,
				Description: "HTTP request timeout in seconds. May also be set with MTN_CLOUD_TIMEOUT.",
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"insecure": pschema.BoolAttribute{
				Optional:    true,
				Description: "Disable TLS certificate verification. MTN_CLOUD_VERIFY_SSL=false also enables this.",
			},
			"group": pschema.StringAttribute{
				Optional:    true,
				Description: "Default group/site name used by resources that omit `group`. May also be set with MTN_CLOUD_GROUP.",
			},
			"resource_pool": pschema.StringAttribute{
				Optional:    true,
				Description: "Default resource pool name/code used by instances that omit `resource_pool`. May also be set with MTN_CLOUD_RESOURCE_POOL.",
			},
			"availability_zone": pschema.StringAttribute{
				Optional:    true,
				Description: "Default availability zone used by instances that omit `availability_zone`. May also be set with MTN_CLOUD_AVAILABILITY_ZONE.",
			},
			"default_labels": pschema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Labels added to every resource that supports labels. Merged (union) with resource-level `labels`.",
			},
			"default_tags": pschema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Tags applied to every resource that supports tags. Resource-level `tags` override these per key.",
			},
		},
	}
}

func (p *mtnCloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := valueOrEnv(config.URL, "MTN_CLOUD_URL", client.DefaultURL)
	token := valueOrEnv(config.Token, "MTN_CLOUD_TOKEN", "")
	username := valueOrEnv(config.Username, "MTN_CLOUD_USERNAME", "")
	password := valueOrEnv(config.Password, "MTN_CLOUD_PASSWORD", "")
	timeoutSeconds := int64OrEnv(config.Timeout, "MTN_CLOUD_TIMEOUT", 30)
	insecure := boolValue(config.Insecure, false)
	if strings.EqualFold(os.Getenv("MTN_CLOUD_VERIFY_SSL"), "false") {
		insecure = true
	}

	if token != "" && (username != "" || password != "") {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Conflicting Authentication Configuration",
			"Use either token authentication or username/password authentication, not both.",
		)
	}
	if token == "" && (username == "" || password == "") {
		resp.Diagnostics.AddError(
			"Missing Authentication Configuration",
			"Provide token or username/password, either in the provider block or via MTN_CLOUD_* environment variables.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient, err := client.New(client.Config{
		URL:      url,
		Token:    token,
		Username: username,
		Password: password,
		Timeout:  time.Duration(timeoutSeconds) * time.Second,
		Insecure: insecure,
	})
	if err != nil {
		resp.Diagnostics.AddError("MTN Cloud Client Configuration Failed", err.Error())
		return
	}

	data := &mtnCloudProviderData{
		Client:           apiClient,
		Group:            valueOrEnv(config.Group, "MTN_CLOUD_GROUP", ""),
		ResourcePool:     valueOrEnv(config.ResourcePool, "MTN_CLOUD_RESOURCE_POOL", ""),
		AvailabilityZone: valueOrEnv(config.AvailabilityZone, "MTN_CLOUD_AVAILABILITY_ZONE", ""),
		DefaultLabels:    stringList(ctx, config.DefaultLabels),
		DefaultTags:      stringMap(ctx, config.DefaultTags),
	}
	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *mtnCloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInstanceResource,
		NewNetworkResource,
		NewSecurityGroupResource,
		NewSecurityGroupRuleResource,
		NewStorageBucketResource,
		NewArchiveBucketResource,
	}
}

func (p *mtnCloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGroupDataSource,
		NewInstanceTypeDataSource,
		NewNetworkDataSource,
		NewResourcePoolDataSource,
		NewSecurityGroupDataSource,
		NewServicePlanDataSource,
		NewVirtualImageDataSource,
	}
}

// configuredProvider returns the full shared provider data (client + defaults).
func configuredProvider(providerData any) (*mtnCloudProviderData, bool) {
	data, ok := providerData.(*mtnCloudProviderData)
	return data, ok
}

// configuredClient returns just the API client, for resources/data sources that
// do not need provider-level defaults.
func configuredClient(providerData any) (*client.Client, bool) {
	data, ok := providerData.(*mtnCloudProviderData)
	if !ok {
		return nil, false
	}
	return data.Client, true
}

func valueOrEnv(value types.String, envName, fallback string) string {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueString()
	}
	if envValue := os.Getenv(envName); envValue != "" {
		return envValue
	}
	return fallback
}

func int64OrEnv(value types.Int64, envName string, fallback int64) int64 {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueInt64()
	}
	if envValue := os.Getenv(envName); envValue != "" {
		parsed, err := strconv.ParseInt(envValue, 10, 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func boolValue(value types.Bool, fallback bool) bool {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueBool()
	}
	return fallback
}

func boolPtr(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueBool()
	return &v
}

func int64Ptr(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueInt64()
	return &v
}

func stringPtr(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueString()
	return &v
}

func optionalString(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func maybeInt64(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}

func maybeBool(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

// mergeAPIString reconciles an Optional+Computed string with the API response.
// A non-empty API value always wins (covers backend defaults/normalization). An
// empty API value keeps the existing configured/prior value rather than nulling
// it, so the post-apply state never diverges from the plan when the API simply
// omits the field. Unknown (not yet set) collapses to null.
func mergeAPIString(existing types.String, apiValue string) types.String {
	if apiValue != "" {
		return types.StringValue(apiValue)
	}
	if existing.IsUnknown() {
		return types.StringNull()
	}
	return existing
}

func mergeAPIInt64(existing types.Int64, apiValue *int64) types.Int64 {
	if apiValue != nil {
		return types.Int64Value(*apiValue)
	}
	if existing.IsUnknown() {
		return types.Int64Null()
	}
	return existing
}

func mergeAPIBool(existing types.Bool, apiValue *bool) types.Bool {
	if apiValue != nil {
		return types.BoolValue(*apiValue)
	}
	if existing.IsUnknown() {
		return types.BoolNull()
	}
	return existing
}

// mergeLabels unions provider default labels with resource labels, preserving
// order (defaults first) and dropping duplicates and empties.
func mergeLabels(defaults, resource []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(defaults)+len(resource))
	for _, value := range append(append([]string{}, defaults...), resource...) {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

// mergeTags overlays resource tags on top of provider default tags (resource wins).
func mergeTags(defaults, resource map[string]string) map[string]string {
	out := make(map[string]string, len(defaults)+len(resource))
	for key, value := range defaults {
		out[key] = value
	}
	for key, value := range resource {
		out[key] = value
	}
	return out
}
