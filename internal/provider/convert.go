package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// These helpers translate between the Terraform framework's null/unknown-aware
// types and the plain Go (pointer) values the API client uses.

// strPtr returns a *string for a known, non-null types.String, or nil.
func strPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// boolPtr returns a *bool for a known, non-null types.Bool, or nil.
func boolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// int64Ptr returns a *int64 for a known, non-null types.Int64, or nil.
func int64Ptr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

// stringFromPtr converts a *string to a types.String (nil -> null).
func stringFromPtr(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

// int64FromPtr converts a *int64 to a types.Int64 (nil -> null).
func int64FromPtr(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}

// stringSlice extracts a []string from a types.Set of strings. A null/unknown
// set yields nil.
func stringSlice(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := set.ElementsAs(ctx, &out, false)
	return out, diags
}

// stringMap extracts a map[string]string from a types.Map of strings. A
// null/unknown map yields nil.
func stringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	out := map[string]string{}
	diags := m.ElementsAs(ctx, &out, false)
	return out, diags
}

// filterStringMap rebuilds a types.Map keeping only the keys present in prior
// (the user-managed keys), taking values from server. When prior is null/unknown
// (e.g. import), all non-nil server entries are kept. This neutralizes
// server-added keys that would otherwise look like drift.
func filterStringMap(prior types.Map, server map[string]*string) (types.Map, diag.Diagnostics) {
	elems := map[string]attr.Value{}
	if prior.IsNull() || prior.IsUnknown() {
		for k, v := range server {
			if v != nil {
				elems[k] = types.StringValue(*v)
			}
		}
	} else {
		for k := range prior.Elements() {
			if v, ok := server[k]; ok && v != nil {
				elems[k] = types.StringValue(*v)
			}
		}
	}
	if len(elems) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValue(types.StringType, elems)
}

// stringSetValue builds a types.Set of strings from a slice.
func stringSetValue(values []string) (types.Set, diag.Diagnostics) {
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.SetValue(types.StringType, elems)
}
