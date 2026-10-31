package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &organizationGroupRoleResource{}
	_ resource.ResourceWithConfigure   = &organizationGroupRoleResource{}
	_ resource.ResourceWithImportState = &organizationGroupRoleResource{}
)

// NewOrganizationGroupRoleResource is the resource factory.
func NewOrganizationGroupRoleResource() resource.Resource {
	return &organizationGroupRoleResource{}
}

type organizationGroupRoleResource struct {
	client *client.Client
}

type organizationGroupRoleResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Realm          types.String `tfsdk:"realm"`
	OrganizationID types.String `tfsdk:"organization_id"`
	GroupID        types.String `tfsdk:"group_id"`
	RoleID         types.String `tfsdk:"role_id"`
}

func (r *organizationGroupRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_group_role"
}

func (r *organizationGroupRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Assigns a role to an organization group. A pure link: any change recreates it. " +
			"Organization roles are not a separate kind of role: `role_id` is the UUID of an ordinary realm role (`ferriskey_role` without `client_id`) " +
			"or client role (`ferriskey_role` with `client_id`); both are accepted.",
		Attributes: map[string]schema.Attribute{
			"id":              orgExtIDAttribute("{realm}/{organization_id}/{group_id}/{role_id}"),
			"realm":           orgExtReplaceAttribute("Realm the organization belongs to."),
			"organization_id": orgExtReplaceAttribute("UUID of the organization."),
			"group_id":        orgExtReplaceAttribute("UUID of the organization group (`group_uuid` of `ferriskey_organization_group`)."),
			"role_id":         orgExtReplaceAttribute("UUID of the realm or client role to assign (`role_uuid` of `ferriskey_role`)."),
		},
	}
}

func (r *organizationGroupRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *organizationGroupRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationGroupRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	realm, orgID, groupID, roleID := plan.Realm.ValueString(), plan.OrganizationID.ValueString(), plan.GroupID.ValueString(), plan.RoleID.ValueString()
	existing, err := r.client.ListOrganizationGroupRoles(ctx, realm, orgID, groupID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization group roles", err.Error())
		return
	}
	if !orgExtHasRole(existing, roleID) {
		if err := r.client.AssignOrganizationGroupRole(ctx, realm, orgID, groupID, roleID); err != nil {
			resp.Diagnostics.AddError("Error assigning organization group role", err.Error())
			return
		}
	}
	plan.ID = types.StringValue(realm + "/" + orgID + "/" + groupID + "/" + roleID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationGroupRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationGroupRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	roles, err := r.client.ListOrganizationGroupRoles(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.GroupID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization group roles", err.Error())
		return
	}
	if !orgExtHasRole(roles, state.RoleID.ValueString()) {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable (all attributes force replacement).
func (r *organizationGroupRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationGroupRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationGroupRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationGroupRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveOrganizationGroupRole(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.GroupID.ValueString(), state.RoleID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error removing organization group role", err.Error())
	}
}

func (r *organizationGroupRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, "realm", "organization_id", "group_id", "role_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), parts[3])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
