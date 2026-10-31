package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	_ resource.Resource                = &clientRoleResource{}
	_ resource.ResourceWithConfigure   = &clientRoleResource{}
	_ resource.ResourceWithImportState = &clientRoleResource{}
)

func NewClientRoleResource() resource.Resource {
	return &clientRoleResource{}
}

type clientRoleResource struct {
	client *client.Client
}

type clientRoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Realm       types.String `tfsdk:"realm"`
	ClientID    types.String `tfsdk:"client_id"`
	RoleID      types.String `tfsdk:"role_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permissions types.Set    `tfsdk:"permissions"`
}

type clientExtID struct {
	Realm    string
	ClientID string
	ID       string
}

func (i clientExtID) String() string {
	return i.Realm + "/" + i.ClientID + "/" + i.ID
}

func clientExtParseID(s string) (clientExtID, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return clientExtID{}, fmt.Errorf(
			"invalid import ID %q: expected the format \"{realm}/{client_id}/{id}\", e.g. \"master/3a8c6128-.../9b1d0c4e-...\"", s)
	}
	return clientExtID{Realm: parts[0], ClientID: parts[1], ID: parts[2]}, nil
}

func clientExtImport(ctx context.Context, id, leaf string, state *resource.ImportStateResponse) {
	parsed, err := clientExtParseID(id)
	if err != nil {
		state.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	state.Diagnostics.Append(state.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	state.Diagnostics.Append(state.State.SetAttribute(ctx, path.Root("client_id"), parsed.ClientID)...)
	state.Diagnostics.Append(state.State.SetAttribute(ctx, path.Root(leaf), parsed.ID)...)
	state.Diagnostics.Append(state.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

func (r *clientRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_role"
}

func (r *clientRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a role owned by a FerrisKey client (a client role). The role is created through the client roles endpoint and then managed through the realm role endpoints.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{client_id}/{role_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the client belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the client that owns the role. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the role.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Role name.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Role description.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"permissions": schema.SetAttribute{
				MarkdownDescription: "Set of permission identifiers granted by the role.",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *clientRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissions, d := stringSlice(ctx, plan.Permissions)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if permissions == nil {
		permissions = []string{}
	}

	realm := plan.Realm.ValueString()
	created, err := r.client.CreateClientRole(ctx, realm, plan.ClientID.ValueString(), client.CreateRoleRequest{
		Name:        plan.Name.ValueString(),
		Description: strPtr(plan.Description),
		Permissions: permissions,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating client role", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, plan.ClientID.ValueString(), created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	role, err := r.client.GetRole(ctx, realm, state.RoleID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading client role", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, state.ClientID.ValueString(), role, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state clientRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	id := state.RoleID.ValueString()

	role, err := r.client.UpdateRole(ctx, realm, id, client.UpdateRoleRequest{
		Name:        strPtr(plan.Name),
		Description: strPtr(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating client role", err.Error())
		return
	}

	if !plan.Permissions.Equal(state.Permissions) {
		permissions, d := stringSlice(ctx, plan.Permissions)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if permissions == nil {
			permissions = []string{}
		}
		role, err = r.client.UpdateRolePermissions(ctx, realm, id, permissions)
		if err != nil {
			resp.Diagnostics.AddError("Error updating client role permissions", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(r.flatten(realm, plan.ClientID.ValueString(), role, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRole(ctx, state.Realm.ValueString(), state.RoleID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting client role", err.Error())
	}
}

func (r *clientRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	clientExtImport(ctx, req.ID, "role_id", resp)
}

func (r *clientRoleResource) flatten(realm, clientID string, role *client.Role, m *clientRoleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Realm = types.StringValue(realm)
	m.ClientID = types.StringValue(clientID)
	m.RoleID = types.StringValue(role.ID)
	m.ID = types.StringValue(clientExtID{Realm: realm, ClientID: clientID, ID: role.ID}.String())
	m.Name = types.StringValue(role.Name)
	m.Description = stringFromPtr(role.Description)

	set, d := stringSetValue(role.Permissions)
	diags.Append(d...)
	m.Permissions = set
	return diags
}
