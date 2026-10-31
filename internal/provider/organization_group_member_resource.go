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
	_ resource.Resource                = &organizationGroupMemberResource{}
	_ resource.ResourceWithConfigure   = &organizationGroupMemberResource{}
	_ resource.ResourceWithImportState = &organizationGroupMemberResource{}
)

// NewOrganizationGroupMemberResource is the resource factory.
func NewOrganizationGroupMemberResource() resource.Resource {
	return &organizationGroupMemberResource{}
}

type organizationGroupMemberResource struct {
	client *client.Client
}

type organizationGroupMemberResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Realm          types.String `tfsdk:"realm"`
	OrganizationID types.String `tfsdk:"organization_id"`
	GroupID        types.String `tfsdk:"group_id"`
	UserID         types.String `tfsdk:"user_id"`
}

func (r *organizationGroupMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_group_member"
}

func (r *organizationGroupMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Adds a user to an organization group. A pure link: any change recreates it. " +
			"Group membership is recursive: a member of a group is effectively a member of all its ancestors.",
		Attributes: map[string]schema.Attribute{
			"id":              orgExtIDAttribute("{realm}/{organization_id}/{group_id}/{user_id}"),
			"realm":           orgExtReplaceAttribute("Realm the organization belongs to."),
			"organization_id": orgExtReplaceAttribute("UUID of the organization."),
			"group_id":        orgExtReplaceAttribute("UUID of the organization group (`group_uuid` of `ferriskey_organization_group`)."),
			"user_id":         orgExtReplaceAttribute("UUID of the user to add."),
		},
	}
}

func (r *organizationGroupMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func orgExtHasGroupMember(members []client.OrgGroupMember, userID string) bool {
	for _, m := range members {
		if m.UserID == userID {
			return true
		}
	}
	return false
}

func (r *organizationGroupMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationGroupMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	realm, orgID, groupID, userID := plan.Realm.ValueString(), plan.OrganizationID.ValueString(), plan.GroupID.ValueString(), plan.UserID.ValueString()
	existing, err := r.client.ListOrganizationGroupMembers(ctx, realm, orgID, groupID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization group members", err.Error())
		return
	}
	if !orgExtHasGroupMember(existing, userID) {
		if err := r.client.AddOrganizationGroupMember(ctx, realm, orgID, groupID, userID); err != nil {
			resp.Diagnostics.AddError("Error adding organization group member", err.Error())
			return
		}
	}
	plan.ID = types.StringValue(realm + "/" + orgID + "/" + groupID + "/" + userID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationGroupMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationGroupMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	members, err := r.client.ListOrganizationGroupMembers(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.GroupID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization group members", err.Error())
		return
	}
	if !orgExtHasGroupMember(members, state.UserID.ValueString()) {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable (all attributes force replacement).
func (r *organizationGroupMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationGroupMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationGroupMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationGroupMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveOrganizationGroupMember(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.GroupID.ValueString(), state.UserID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error removing organization group member", err.Error())
	}
}

func (r *organizationGroupMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, "realm", "organization_id", "group_id", "user_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[3])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
