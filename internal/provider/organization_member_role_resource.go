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
	_ resource.Resource                = &organizationMemberRoleResource{}
	_ resource.ResourceWithConfigure   = &organizationMemberRoleResource{}
	_ resource.ResourceWithImportState = &organizationMemberRoleResource{}
)

// NewOrganizationMemberRoleResource is the resource factory.
func NewOrganizationMemberRoleResource() resource.Resource {
	return &organizationMemberRoleResource{}
}

type organizationMemberRoleResource struct {
	client *client.Client
}

type organizationMemberRoleResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Realm          types.String `tfsdk:"realm"`
	OrganizationID types.String `tfsdk:"organization_id"`
	UserID         types.String `tfsdk:"user_id"`
	RoleID         types.String `tfsdk:"role_id"`
}

func (r *organizationMemberRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_member_role"
}

func (r *organizationMemberRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Assigns a role to a member of an organization. A pure link: any change recreates it. " +
			"Organization roles are not a separate kind of role: `role_id` is the UUID of an ordinary realm role (`ferriskey_role` without `client_id`) " +
			"or client role (`ferriskey_role` with `client_id`); both are accepted. " +
			"The user must already be a member of the organization (see `ferriskey_organization_member`).",
		Attributes: map[string]schema.Attribute{
			"id":              orgExtIDAttribute("{realm}/{organization_id}/{user_id}/{role_id}"),
			"realm":           orgExtReplaceAttribute("Realm the organization belongs to."),
			"organization_id": orgExtReplaceAttribute("UUID of the organization."),
			"user_id":         orgExtReplaceAttribute("UUID of the organization member."),
			"role_id":         orgExtReplaceAttribute("UUID of the realm or client role to assign (`role_uuid` of `ferriskey_role`)."),
		},
	}
}

func (r *organizationMemberRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func orgExtHasRole(roles []client.Role, id string) bool {
	for _, role := range roles {
		if role.ID == id {
			return true
		}
	}
	return false
}

func (r *organizationMemberRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationMemberRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	realm, orgID, userID, roleID := plan.Realm.ValueString(), plan.OrganizationID.ValueString(), plan.UserID.ValueString(), plan.RoleID.ValueString()
	existing, err := r.client.ListOrganizationMemberRoles(ctx, realm, orgID, userID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization member roles", err.Error())
		return
	}
	if !orgExtHasRole(existing, roleID) {
		if err := r.client.AssignOrganizationMemberRole(ctx, realm, orgID, userID, roleID); err != nil {
			resp.Diagnostics.AddError("Error assigning organization member role", err.Error())
			return
		}
	}
	plan.ID = types.StringValue(realm + "/" + orgID + "/" + userID + "/" + roleID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationMemberRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationMemberRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	roles, err := r.client.ListOrganizationMemberRoles(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.UserID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization member roles", err.Error())
		return
	}
	if !orgExtHasRole(roles, state.RoleID.ValueString()) {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable (all attributes force replacement).
func (r *organizationMemberRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationMemberRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationMemberRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationMemberRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveOrganizationMemberRole(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.UserID.ValueString(), state.RoleID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error removing organization member role", err.Error())
	}
}

func (r *organizationMemberRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, "realm", "organization_id", "user_id", "role_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), parts[3])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
