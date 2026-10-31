package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &clientScopeAssignmentResource{}
	_ resource.ResourceWithConfigure   = &clientScopeAssignmentResource{}
	_ resource.ResourceWithImportState = &clientScopeAssignmentResource{}
)

// NewClientScopeAssignmentResource is the resource factory.
func NewClientScopeAssignmentResource() resource.Resource {
	return &clientScopeAssignmentResource{}
}

type clientScopeAssignmentResource struct {
	client *client.Client
}

type clientScopeAssignmentResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Realm    types.String `tfsdk:"realm"`
	ClientID types.String `tfsdk:"client_id"`
	ScopeID  types.String `tfsdk:"scope_id"`
	Type     types.String `tfsdk:"type"`
}

func (r *clientScopeAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_scope_assignment"
}

func (r *clientScopeAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches a client scope to a client as a `default` or `optional` scope. The " +
			"assignment is a pure link: any change forces it to be recreated. New clients are automatically " +
			"given the realm's default scopes, so assigning one of those fails because it is already attached.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{client_id}/{scope_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm the client and scope belong to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       requiresReplace,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the client. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       requiresReplace,
			},
			"scope_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the client scope. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       requiresReplace,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Assignment type: `default` or `optional`. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       requiresReplace,
				Validators: []validator.String{
					stringvalidator.OneOf(client.ScopeAssignmentDefault, client.ScopeAssignmentOptional),
				},
			},
		},
	}
}

func (r *clientScopeAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientScopeAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientScopeAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.AssignClientScope(ctx, plan.Realm.ValueString(), plan.ClientID.ValueString(),
		plan.ScopeID.ValueString(), plan.Type.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error assigning client scope", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.Realm.ValueString() + "/" + plan.ClientID.ValueString() + "/" + plan.ScopeID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientScopeAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientScopeAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopeType, err := r.client.GetClientScopeAssignment(ctx, state.Realm.ValueString(),
		state.ClientID.ValueString(), state.ScopeID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading client scope assignment", err.Error())
		return
	}

	state.Type = types.StringValue(strings.ToLower(scopeType))
	state.ID = types.StringValue(state.Realm.ValueString() + "/" + state.ClientID.ValueString() + "/" + state.ScopeID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientScopeAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clientScopeAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientScopeAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientScopeAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UnassignClientScope(ctx, state.Realm.ValueString(), state.ClientID.ValueString(),
		state.ScopeID.ValueString(), state.Type.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error unassigning client scope", err.Error())
	}
}

func (r *clientScopeAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("invalid import ID %q: expected \"{realm}/{client_id}/{scope_id}\"", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
