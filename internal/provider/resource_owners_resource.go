package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &resourceOwnersResource{}
	_ resource.ResourceWithConfigure   = &resourceOwnersResource{}
	_ resource.ResourceWithImportState = &resourceOwnersResource{}
)

var authzOwnerObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"uri":       types.StringType,
	"client_id": types.StringType,
}}

func NewResourceOwnersResource() resource.Resource {
	return &resourceOwnersResource{}
}

type resourceOwnersResource struct {
	client *client.Client
}

type authzOwnerModel struct {
	URI      types.String `tfsdk:"uri"`
	ClientID types.String `tfsdk:"client_id"`
}

type resourceOwnersResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Realm  types.String `tfsdk:"realm"`
	Owners types.Set    `tfsdk:"owners"`
}

func (r *resourceOwnersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_owners"
}

func (r *resourceOwnersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages which client owns each protected resource of a FerrisKey realm (singleton per " +
			"realm). A protected resource is identified by its URI, which must already be listed in the realm's " +
			"`allowed_resources` setting; its owner is a confidential, admin-registered client of the same realm. " +
			"The API replaces the whole list on every write, so this resource is **authoritative** for the realm: " +
			"owners not listed in `owners` are removed, and destroying the resource clears all owners.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (equal to `realm`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm whose resource owners are managed. Changing it forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"owners": schema.SetNestedAttribute{
				MarkdownDescription: "Resource ownerships. Each `uri` can appear only once.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uri": schema.StringAttribute{
							MarkdownDescription: "URI of the protected resource; must be one of the realm's `allowed_resources`.",
							Required:            true,
						},
						"client_id": schema.StringAttribute{
							MarkdownDescription: "UUID of the owning client (`client_uuid` of `ferriskey_client`). Must be confidential and registered by an admin.",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (r *resourceOwnersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *resourceOwnersResource) replace(ctx context.Context, realm string, owners []client.ResourceOwner) ([]client.ResourceOwner, error) {
	return r.client.ReplaceResourceOwners(ctx, realm, owners)
}

func (r *resourceOwnersResource) expand(ctx context.Context, set types.Set) ([]client.ResourceOwner, diag.Diagnostics) {
	var models []authzOwnerModel
	diags := set.ElementsAs(ctx, &models, false)
	owners := make([]client.ResourceOwner, 0, len(models))
	for _, m := range models {
		owners = append(owners, client.ResourceOwner{URI: m.URI.ValueString(), ClientID: m.ClientID.ValueString()})
	}
	return owners, diags
}

func (r *resourceOwnersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceOwnersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	owners, d := r.expand(ctx, plan.Owners)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.replace(ctx, plan.Realm.ValueString(), owners)
	if err != nil {
		resp.Diagnostics.AddError("Error setting resource owners", err.Error())
		return
	}
	resp.Diagnostics.Append(r.flatten(ctx, plan.Realm.ValueString(), got, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceOwnersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceOwnersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.GetResourceOwners(ctx, state.Realm.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading resource owners", err.Error())
		return
	}
	resp.Diagnostics.Append(r.flatten(ctx, state.Realm.ValueString(), got, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceOwnersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceOwnersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	owners, d := r.expand(ctx, plan.Owners)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.replace(ctx, plan.Realm.ValueString(), owners)
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource owners", err.Error())
		return
	}
	resp.Diagnostics.Append(r.flatten(ctx, plan.Realm.ValueString(), got, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceOwnersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceOwnersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.replace(ctx, state.Realm.ValueString(), nil); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error clearing resource owners", err.Error())
	}
}

func (r *resourceOwnersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *resourceOwnersResource) flatten(ctx context.Context, realm string, owners []client.ResourceOwner, m *resourceOwnersResourceModel) diag.Diagnostics {
	m.Realm = types.StringValue(realm)
	m.ID = types.StringValue(realm)
	models := make([]authzOwnerModel, 0, len(owners))
	for _, o := range owners {
		models = append(models, authzOwnerModel{URI: types.StringValue(o.URI), ClientID: types.StringValue(o.ClientID)})
	}
	set, diags := types.SetValueFrom(ctx, authzOwnerObjectType, models)
	m.Owners = set
	return diags
}
