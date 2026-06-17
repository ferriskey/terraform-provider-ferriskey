package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &userRoleResource{}
	_ resource.ResourceWithConfigure   = &userRoleResource{}
	_ resource.ResourceWithImportState = &userRoleResource{}
)

// NewUserRoleResource is the resource factory.
func NewUserRoleResource() resource.Resource {
	return &userRoleResource{}
}

type userRoleResource struct {
	client *client.Client
}

type userRoleResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Realm  types.String `tfsdk:"realm"`
	UserID types.String `tfsdk:"user_id"`
	RoleID types.String `tfsdk:"role_id"`
}

func (r *userRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_role"
}

func (r *userRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Assigns a realm role to a user (or a client's service account user). The " +
			"assignment is a pure link: any change forces it to be recreated. Commonly used to grant a " +
			"Terraform service account its scoped admin roles during bootstrap.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{user_id}/{role_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm the user and role belong to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       requiresReplace,
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the user (e.g. a client's `service_account_user_id`). " +
					"Changing this forces a new resource.",
				Required:      true,
				PlanModifiers: requiresReplace,
			},
			"role_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the role to assign. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       requiresReplace,
			},
		},
	}
}

func (r *userRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *userRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.AssignRoleToUser(ctx, plan.Realm.ValueString(), plan.UserID.ValueString(), plan.RoleID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error assigning role to user", err.Error())
		return
	}

	plan.ID = types.StringValue(userRoleID(plan.Realm.ValueString(), plan.UserID.ValueString(), plan.RoleID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := r.client.ListUserRoles(ctx, state.Realm.ValueString(), state.UserID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user roles", err.Error())
		return
	}

	roleID := state.RoleID.ValueString()
	found := false
	for _, role := range roles {
		if role.ID == roleID {
			found = true
			break
		}
	}
	if !found {
		// The assignment no longer exists; drop it from state so it is recreated.
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(userRoleID(state.Realm.ValueString(), state.UserID.ValueString(), roleID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable: every attribute forces replacement. It is required to
// satisfy the resource.Resource interface.
func (r *userRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.RemoveRoleFromUser(ctx, state.Realm.ValueString(), state.UserID.ValueString(), state.RoleID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error removing role from user", err.Error())
	}
}

func (r *userRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("invalid import ID %q: expected \"{realm}/{user_id}/{role_id}\"", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func userRoleID(realm, userID, roleID string) string {
	return realm + "/" + userID + "/" + roleID
}
