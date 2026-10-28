package provider

import (
	"context"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

// resourceBase is embedded by every resource. It implements Configure once and
// exposes the API client plus provider-level defaults, so individual resources
// don't repeat the wiring. Resources reference r.client (and r.defaults when they
// inherit provider defaults like group/labels/tags).
type resourceBase struct {
	client   *client.Client
	defaults *mtnCloudProviderData
}

func (b *resourceBase) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := configuredProvider(req.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", "Expected *mtnCloudProviderData.")
		return
	}
	b.client = data.Client
	b.defaults = data
}

// dataSourceBase provides the client and optional provider defaults to data sources.
type dataSourceBase struct {
	client   *client.Client
	defaults *mtnCloudProviderData
}

func (b *dataSourceBase) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := configuredClient(req.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Data", "Expected *client.Client.")
		return
	}
	b.client = apiClient
	b.defaults, _ = configuredProvider(req.ProviderData)
}

// configuredProvider returns the full shared provider data (client + defaults).
func configuredProvider(providerData any) (*mtnCloudProviderData, bool) {
	data, ok := providerData.(*mtnCloudProviderData)
	return data, ok
}

// configuredClient returns just the API client, for code that does not need
// provider-level defaults.
func configuredClient(providerData any) (*client.Client, bool) {
	data, ok := providerData.(*mtnCloudProviderData)
	if !ok {
		return nil, false
	}
	return data.Client, true
}

// valueOrEnv resolves a string config value, falling back to an environment
// variable and then a static default.
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
		if parsed, err := strconv.ParseInt(envValue, 10, 64); err == nil {
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

// stringList converts a framework List of strings to a Go slice.
func stringList(ctx context.Context, value types.List) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var out []string
	value.ElementsAs(ctx, &out, false)
	return out
}

// int64Slice converts a framework List of Int64 to a Go slice.
func int64Slice(ctx context.Context, value types.List) []int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var out []int64
	value.ElementsAs(ctx, &out, false)
	return out
}

// stringMap converts a framework Map of strings to a Go map.
func stringMap(ctx context.Context, value types.Map) map[string]string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	out := make(map[string]string)
	value.ElementsAs(ctx, &out, false)
	return out
}
