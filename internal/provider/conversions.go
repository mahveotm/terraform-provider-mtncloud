package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Conversion helpers between framework types and Go values. Two directions:
//   *Ptr   : framework value -> *Go (nil when null/unknown), for request payloads.
//   maybe* : *Go -> framework value (null when nil), for computed read-back.
// The mergeAPI* helpers reconcile Optional+Computed attributes with the API
// response: a present API value wins; an absent one keeps the prior value rather
// than nulling it, so post-apply state never diverges from plan.

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

func float64Ptr(value types.Float64) *float64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueFloat64()
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
// it. Unknown (not yet set) collapses to null.
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

func mergeAPIFloat64(existing types.Float64, apiValue *float64) types.Float64 {
	if apiValue != nil {
		return types.Float64Value(*apiValue)
	}
	if existing.IsUnknown() {
		return types.Float64Null()
	}
	return existing
}

// int64Set converts a framework Set of Int64 to a Go slice. A null/unknown set
// yields nil so the request payload omits the field entirely.
func int64Set(ctx context.Context, value types.Set) []int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var out []int64
	value.ElementsAs(ctx, &out, false)
	return out
}

// int64SetValue converts a Go slice of IDs to a framework Set of Int64 (used by
// data sources to expose relational id collections).
func int64SetValue(ctx context.Context, ids []int64) (types.Set, diag.Diagnostics) {
	return types.SetValueFrom(ctx, types.Int64Type, ids)
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
