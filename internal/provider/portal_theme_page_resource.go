package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &portalThemePageResource{}
	_ resource.ResourceWithConfigure   = &portalThemePageResource{}
	_ resource.ResourceWithImportState = &portalThemePageResource{}
)

var portalPageTypes = []string{
	"login", "register", "totp", "forgot_password", "reset_password", "magic_link_verify",
	"magic_link_request", "verify_email", "email_verified", "totp_setup", "device_verify",
	"device_verified", "consent",
}

func NewPortalThemePageResource() resource.Resource {
	return &portalThemePageResource{}
}

type portalThemePageResource struct {
	client *client.Client
}

type portalThemePageResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Realm    types.String `tfsdk:"realm"`
	ThemeID  types.String `tfsdk:"theme_id"`
	PageType types.String `tfsdk:"page_type"`
	Tree     types.String `tfsdk:"tree"`
}

func (r *portalThemePageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_portal_theme_page"
}

func (r *portalThemePageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the component tree of one page (for example `login`) of a portal theme. The API has " +
			"no way to delete a page: destroying this resource resets the page to an empty tree (`[]`). If the theme is " +
			"active the API rejects an empty tree for pages with required blocks; in that case destroy emits a warning " +
			"and only removes the resource from state. The active theme must always keep the required blocks of " +
			"each page (for example `email_input`, `password_input`, `submit_button` and `identity_providers` for `login`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{theme_id}/{page_type}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the theme belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"theme_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the `ferriskey_portal_theme` owning the page. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"page_type": schema.StringAttribute{
				MarkdownDescription: "Page to manage: one of `" + strings.Join(portalPageTypes, "`, `") + "`. Changing this forces a new resource.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(portalPageTypes...)},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"tree": schema.StringAttribute{
				MarkdownDescription: "Component tree of the page as a JSON string, typically built with `jsonencode`.",
				Required:            true,
				Validators:          []validator.String{portalJSONValidator{}},
			},
		},
	}
}

func (r *portalThemePageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *portalThemePageResource) apply(ctx context.Context, plan *portalThemePageResourceModel) error {
	realm := plan.Realm.ValueString()
	pageType := plan.PageType.ValueString()
	theme, err := r.client.UpdatePortalThemePage(ctx, realm, plan.ThemeID.ValueString(), pageType, json.RawMessage(plan.Tree.ValueString()))
	if err != nil {
		return err
	}
	return r.flatten(realm, pageType, theme, plan)
}

func (r *portalThemePageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portalThemePageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error setting portal theme page", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *portalThemePageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portalThemePageResourceModel
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
		resp.Diagnostics.AddError("Error reading portal theme page", err.Error())
		return
	}
	if err := r.flatten(realm, state.PageType.ValueString(), theme, &state); err != nil {
		resp.Diagnostics.AddError("Error reading portal theme page", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *portalThemePageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan portalThemePageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating portal theme page", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *portalThemePageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state portalThemePageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdatePortalThemePage(ctx, state.Realm.ValueString(), state.ThemeID.ValueString(),
		state.PageType.ValueString(), json.RawMessage(`[]`))
	var apiErr *client.APIErrorResponse
	switch {
	case err == nil, errors.Is(err, client.ErrNotFound):
	case errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500:
		resp.Diagnostics.AddWarning("Portal theme page not reset",
			"The FerrisKey API rejected resetting the page of an active theme. It was removed from Terraform state only: "+err.Error())
	default:
		resp.Diagnostics.AddError("Error resetting portal theme page", err.Error())
	}
}

func (r *portalThemePageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := portalParseID(req.ID, "realm", "theme_id", "page_type")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("theme_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("page_type"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *portalThemePageResource) flatten(realm, pageType string, theme *client.PortalTheme, m *portalThemePageResourceModel) error {
	var pages map[string]json.RawMessage
	if err := json.Unmarshal(theme.Pages, &pages); err != nil {
		return err
	}
	tree, ok := pages[portalPageKey(pageType)]
	if !ok {
		tree = json.RawMessage(`[]`)
	}
	value, err := portalReconcileJSON(m.Tree, tree, false)
	if err != nil {
		return err
	}
	m.Realm = types.StringValue(realm)
	m.ThemeID = types.StringValue(theme.ID)
	m.PageType = types.StringValue(pageType)
	m.ID = types.StringValue(realm + "/" + theme.ID + "/" + pageType)
	m.Tree = value
	return nil
}

func portalPageKey(pageType string) string {
	parts := strings.Split(pageType, "_")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}
