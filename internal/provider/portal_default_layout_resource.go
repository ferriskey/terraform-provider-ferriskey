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
	_ resource.Resource                = &portalDefaultLayoutResource{}
	_ resource.ResourceWithConfigure   = &portalDefaultLayoutResource{}
	_ resource.ResourceWithImportState = &portalDefaultLayoutResource{}
)

func NewPortalDefaultLayoutResource() resource.Resource {
	return &portalDefaultLayoutResource{}
}

type portalDefaultLayoutResource struct {
	client *client.Client
}

type portalDefaultLayoutResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Realm    types.String `tfsdk:"realm"`
	LayoutID types.String `tfsdk:"layout_id"`
}

func (r *portalDefaultLayoutResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_portal_default_layout"
}

func (r *portalDefaultLayoutResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Selects the default portal layout of a realm (singleton per realm). A realm always has " +
			"at most one default and the API cannot unset it, so destroying this resource only removes it from " +
			"Terraform state and leaves the layout as default.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (equal to `realm`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm whose default layout is managed. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"layout_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the `ferriskey_portal_layout` to make the realm default.",
				Required:            true,
			},
		},
	}
}

func (r *portalDefaultLayoutResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *portalDefaultLayoutResource) set(ctx context.Context, plan *portalDefaultLayoutResourceModel) error {
	realm := plan.Realm.ValueString()
	if _, err := r.client.SetDefaultPortalLayout(ctx, realm, plan.LayoutID.ValueString()); err != nil {
		return err
	}
	plan.ID = types.StringValue(realm)
	return nil
}

func (r *portalDefaultLayoutResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portalDefaultLayoutResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.set(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error setting default portal layout", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *portalDefaultLayoutResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portalDefaultLayoutResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	layout, err := r.client.GetDefaultPortalLayout(ctx, realm)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading default portal layout", err.Error())
		return
	}

	state.ID = types.StringValue(realm)
	state.LayoutID = types.StringValue(layout.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *portalDefaultLayoutResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan portalDefaultLayoutResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.set(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error setting default portal layout", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *portalDefaultLayoutResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *portalDefaultLayoutResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
