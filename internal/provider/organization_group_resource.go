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
	_ resource.Resource                = &organizationGroupResource{}
	_ resource.ResourceWithConfigure   = &organizationGroupResource{}
	_ resource.ResourceWithImportState = &organizationGroupResource{}
)

// NewOrganizationGroupResource is the resource factory.
func NewOrganizationGroupResource() resource.Resource {
	return &organizationGroupResource{}
}

type organizationGroupResource struct {
	client *client.Client
}

type organizationGroupResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Realm          types.String `tfsdk:"realm"`
	OrganizationID types.String `tfsdk:"organization_id"`
	GroupUUID      types.String `tfsdk:"group_uuid"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	ParentGroupID  types.String `tfsdk:"parent_group_id"`
}

func (r *organizationGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_group"
}

func (r *organizationGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a group inside an organization. Groups form a tree through `parent_group_id`. " +
			"Group names must be unique within an organization.",
		Attributes: map[string]schema.Attribute{
			"id":              orgExtIDAttribute("{realm}/{organization_id}/{group_uuid}"),
			"realm":           orgExtReplaceAttribute("Realm the organization belongs to."),
			"organization_id": orgExtReplaceAttribute("UUID of the organization."),
			"group_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the group, assigned by FerrisKey.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the group.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the group. Do not set an empty string: FerrisKey stores an empty description as unset.",
				Optional:            true,
			},
			"parent_group_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the parent group; omit for a top-level group. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *organizationGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *organizationGroupResource) flatten(g *client.OrgGroup, m *organizationGroupResourceModel) {
	m.ID = types.StringValue(m.Realm.ValueString() + "/" + g.OrganizationID + "/" + g.ID)
	m.OrganizationID = types.StringValue(g.OrganizationID)
	m.GroupUUID = types.StringValue(g.ID)
	m.Name = types.StringValue(g.Name)
	m.Description = types.StringNull()
	if g.Description != nil && *g.Description != "" {
		m.Description = types.StringValue(*g.Description)
	}
	m.ParentGroupID = types.StringNull()
	if g.ParentGroupID != nil {
		m.ParentGroupID = types.StringValue(*g.ParentGroupID)
	}
}

func (r *organizationGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := client.CreateOrgGroupRequest{Name: plan.Name.ValueString()}
	if !plan.Description.IsNull() {
		body.Description = plan.Description.ValueStringPointer()
	}
	if !plan.ParentGroupID.IsNull() {
		body.ParentGroupID = plan.ParentGroupID.ValueStringPointer()
	}
	g, err := r.client.CreateOrganizationGroup(ctx, plan.Realm.ValueString(), plan.OrganizationID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization group", err.Error())
		return
	}
	r.flatten(g, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	g, err := r.client.GetOrganizationGroup(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.GroupUUID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization group", err.Error())
		return
	}
	r.flatten(g, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state organizationGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	description := plan.Description.ValueString()
	g, err := r.client.UpdateOrganizationGroup(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.GroupUUID.ValueString(), client.UpdateOrgGroupRequest{
		Name:        plan.Name.ValueStringPointer(),
		Description: &description,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization group", err.Error())
		return
	}
	plan.GroupUUID = state.GroupUUID
	r.flatten(g, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteOrganizationGroup(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.GroupUUID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting organization group", err.Error())
	}
}

func (r *organizationGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, "realm", "organization_id", "group_uuid")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_uuid"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
