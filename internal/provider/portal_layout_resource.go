package provider

import (
	"context"
	"encoding/json"
	"errors"

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
	_ resource.Resource                = &portalLayoutResource{}
	_ resource.ResourceWithConfigure   = &portalLayoutResource{}
	_ resource.ResourceWithImportState = &portalLayoutResource{}
)

func NewPortalLayoutResource() resource.Resource {
	return &portalLayoutResource{}
}

type portalLayoutResource struct {
	client *client.Client
}

type portalLayoutResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Realm     types.String `tfsdk:"realm"`
	LayoutID  types.String `tfsdk:"layout_id"`
	Name      types.String `tfsdk:"name"`
	Tree      types.String `tfsdk:"tree"`
	IsDefault types.Bool   `tfsdk:"is_default"`
}

func (r *portalLayoutResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_portal_layout"
}

func (r *portalLayoutResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a login portal layout (the frame around themed pages) in a FerrisKey realm. " +
			"The first layout created in a realm automatically becomes its default; use " +
			"`ferriskey_portal_default_layout` to choose the default explicitly. The API refuses to delete the " +
			"default layout: destroying it emits a warning and only removes it from Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{layout_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the layout belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"layout_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the layout.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Layout name.",
				Required:            true,
			},
			"tree": schema.StringAttribute{
				MarkdownDescription: "Layout component tree as a JSON string, typically built with `jsonencode`.",
				Required:            true,
				Validators:          []validator.String{portalJSONValidator{}},
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Whether this layout is the realm default. Read-only; see `ferriskey_portal_default_layout`.",
				Computed:            true,
			},
		},
	}
}

func (r *portalLayoutResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *portalLayoutResource) request(plan *portalLayoutResourceModel) client.PortalLayoutRequest {
	return client.PortalLayoutRequest{
		Name: plan.Name.ValueString(),
		Tree: json.RawMessage(plan.Tree.ValueString()),
	}
}

func (r *portalLayoutResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portalLayoutResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	created, err := r.client.CreatePortalLayout(ctx, realm, r.request(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating portal layout", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *portalLayoutResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portalLayoutResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	layout, err := r.client.GetPortalLayout(ctx, realm, state.LayoutID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading portal layout", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, layout, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *portalLayoutResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state portalLayoutResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	layout, err := r.client.UpdatePortalLayout(ctx, realm, state.LayoutID.ValueString(), r.request(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating portal layout", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, layout, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *portalLayoutResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state portalLayoutResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePortalLayout(ctx, state.Realm.ValueString(), state.LayoutID.ValueString())
	switch {
	case err == nil, errors.Is(err, client.ErrNotFound):
	case portalAPIMessageContains(err, "realm default"):
		resp.Diagnostics.AddWarning("Default portal layout not deleted",
			"The FerrisKey API cannot delete the default portal layout. It was removed from Terraform state only and "+
				"remains on the server until another layout is made default or the realm is deleted.")
	default:
		resp.Diagnostics.AddError("Error deleting portal layout", err.Error())
	}
}

func (r *portalLayoutResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("layout_id"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

func (r *portalLayoutResource) flatten(realm string, layout *client.PortalLayout, m *portalLayoutResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Realm = types.StringValue(realm)
	m.LayoutID = types.StringValue(layout.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: layout.ID}.String())
	m.Name = types.StringValue(layout.Name)
	m.IsDefault = types.BoolValue(layout.IsDefault)

	tree, err := portalReconcileJSON(m.Tree, layout.Tree, false)
	if err != nil {
		diags.AddError("Invalid portal layout tree returned by the API", err.Error())
		return diags
	}
	m.Tree = tree
	return diags
}
