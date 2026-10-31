package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &portalThemeResource{}
	_ resource.ResourceWithConfigure   = &portalThemeResource{}
	_ resource.ResourceWithImportState = &portalThemeResource{}
)

func NewPortalThemeResource() resource.Resource {
	return &portalThemeResource{}
}

type portalThemeResource struct {
	client *client.Client
}

type portalThemeResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Realm    types.String `tfsdk:"realm"`
	ThemeID  types.String `tfsdk:"theme_id"`
	Name     types.String `tfsdk:"name"`
	LayoutID types.String `tfsdk:"layout_id"`
	Config   types.String `tfsdk:"config"`
}

func (r *portalThemeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_portal_theme"
}

func (r *portalThemeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a login portal theme (design tokens plus optional layout) in a FerrisKey realm. " +
			"Page content is managed with `ferriskey_portal_theme_page` and activation with `ferriskey_portal_active_theme`. " +
			"The API refuses to delete the currently active theme: destroying such a theme emits a warning and only " +
			"removes it from Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{theme_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the theme belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"theme_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the theme.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Theme name.",
				Required:            true,
			},
			"layout_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the `ferriskey_portal_layout` framing the theme's pages; null for none.",
				Optional:            true,
			},
			"config": schema.StringAttribute{
				MarkdownDescription: "Design tokens (`colors`, `fonts`, `borders`, `spacing`) as a JSON string, typically " +
					"built with `jsonencode`. Omitted keys take server defaults; the server's defaults are not treated as " +
					"drift as long as every configured key matches. When unset, the server-side config is exposed " +
					"read-only. Note that the API replaces the whole config on every update.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{portalJSONValidator{}},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *portalThemeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *portalThemeResource) request(plan *portalThemeResourceModel) client.PortalThemeRequest {
	req := client.PortalThemeRequest{
		Name:     plan.Name.ValueString(),
		LayoutID: strPtr(plan.LayoutID),
	}
	if cfg := strPtr(plan.Config); cfg != nil {
		req.Config = json.RawMessage(*cfg)
	}
	return req
}

func (r *portalThemeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portalThemeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	created, err := r.client.CreatePortalTheme(ctx, realm, r.request(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating portal theme", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *portalThemeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portalThemeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	theme, err := r.client.GetPortalTheme(ctx, realm, state.ThemeID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading portal theme", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, theme, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *portalThemeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state portalThemeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	theme, err := r.client.UpdatePortalTheme(ctx, realm, state.ThemeID.ValueString(), r.request(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating portal theme", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, theme, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *portalThemeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state portalThemeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePortalTheme(ctx, state.Realm.ValueString(), state.ThemeID.ValueString())
	switch {
	case err == nil, errors.Is(err, client.ErrNotFound):
	case portalAPIMessageContains(err, "currently active"):
		resp.Diagnostics.AddWarning("Active portal theme not deleted",
			"The FerrisKey API cannot delete the active portal theme. It was removed from Terraform state only and "+
				"remains on the server until another theme is activated or the realm is deleted.")
	default:
		resp.Diagnostics.AddError("Error deleting portal theme", err.Error())
	}
}

func (r *portalThemeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := portalParseID(req.ID, "realm", "theme_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("theme_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *portalThemeResource) flatten(realm string, theme *client.PortalTheme, m *portalThemeResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Realm = types.StringValue(realm)
	m.ThemeID = types.StringValue(theme.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: theme.ID}.String())
	m.Name = types.StringValue(theme.Name)
	m.LayoutID = stringFromPtr(theme.LayoutID)

	cfg, err := portalReconcileJSON(m.Config, theme.Config, true)
	if err != nil {
		diags.AddError("Invalid portal theme config returned by the API", err.Error())
		return diags
	}
	m.Config = cfg
	return diags
}

type portalJSONValidator struct{}

func (portalJSONValidator) Description(_ context.Context) string {
	return "value must be valid JSON"
}

func (v portalJSONValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (portalJSONValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if !json.Valid([]byte(req.ConfigValue.ValueString())) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", "The value must be a valid JSON document.")
	}
}

func portalJSONContains(prior, server any, subset bool) bool {
	if subset {
		pm, pok := prior.(map[string]any)
		sm, sok := server.(map[string]any)
		if pok && sok {
			for k, pv := range pm {
				sv, found := sm[k]
				if !found || !portalJSONContains(pv, sv, true) {
					return false
				}
			}
			return true
		}
	}
	return reflect.DeepEqual(prior, server)
}

func portalReconcileJSON(prior types.String, server json.RawMessage, subset bool) (types.String, error) {
	var sv any
	if err := json.Unmarshal(server, &sv); err != nil {
		return types.StringNull(), err
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		var pv any
		if err := json.Unmarshal([]byte(prior.ValueString()), &pv); err == nil && portalJSONContains(pv, sv, subset) {
			return prior, nil
		}
	}
	normalized, err := json.Marshal(sv)
	if err != nil {
		return types.StringNull(), err
	}
	return types.StringValue(string(normalized)), nil
}

func portalParseID(s string, names ...string) ([]string, error) {
	parts := strings.SplitN(s, "/", len(names))
	valid := len(parts) == len(names)
	for _, p := range parts {
		valid = valid && p != ""
	}
	if !valid {
		return nil, fmt.Errorf("invalid import ID %q: expected the format \"{%s}\"", s, strings.Join(names, "}/{"))
	}
	return parts, nil
}

func portalAPIMessageContains(err error, substr string) bool {
	var apiErr *client.APIErrorResponse
	if !errors.As(err, &apiErr) {
		return false
	}
	if strings.Contains(apiErr.Err.Message, substr) {
		return true
	}
	for _, fe := range apiErr.Err.Errors {
		if strings.Contains(fe.Message, substr) {
			return true
		}
	}
	return false
}
