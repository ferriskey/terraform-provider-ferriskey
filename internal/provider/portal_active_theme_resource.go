package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &portalActiveThemeResource{}
	_ resource.ResourceWithConfigure   = &portalActiveThemeResource{}
	_ resource.ResourceWithImportState = &portalActiveThemeResource{}
)

func NewPortalActiveThemeResource() resource.Resource {
	return &portalActiveThemeResource{}
}

type portalActiveThemeResource struct {
	client *client.Client
}

type portalActiveThemeResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Realm   types.String `tfsdk:"realm"`
	ThemeID types.String `tfsdk:"theme_id"`
}

func (r *portalActiveThemeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_portal_active_theme"
}

func (r *portalActiveThemeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Selects which portal theme is live for a realm (singleton per realm). Activation fails " +
			"unless every page of the theme contains its required blocks. The API has no way to deactivate a theme: " +
			"destroying this resource only removes it from Terraform state and leaves the theme active.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (equal to `realm`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm whose active theme is managed. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"theme_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the `ferriskey_portal_theme` to activate.",
				Required:            true,
			},
		},
	}
}

func (r *portalActiveThemeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *portalActiveThemeResource) activate(ctx context.Context, plan *portalActiveThemeResourceModel) error {
	realm := plan.Realm.ValueString()
	if err := r.client.ActivatePortalTheme(ctx, realm, plan.ThemeID.ValueString()); err != nil {
		return err
	}
	plan.ID = types.StringValue(realm)
	return nil
}

func (r *portalActiveThemeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portalActiveThemeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.activate(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error activating portal theme", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *portalActiveThemeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portalActiveThemeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	themeID, err := r.client.GetActivePortalThemeID(ctx, realm)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading active portal theme", err.Error())
		return
	}
	if themeID == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(realm)
	state.ThemeID = types.StringValue(themeID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *portalActiveThemeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan portalActiveThemeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.activate(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error activating portal theme", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *portalActiveThemeResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *portalActiveThemeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
