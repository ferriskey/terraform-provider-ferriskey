package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &clientScopeResource{}
	_ resource.ResourceWithConfigure   = &clientScopeResource{}
	_ resource.ResourceWithImportState = &clientScopeResource{}
)

// NewClientScopeResource is the resource factory.
func NewClientScopeResource() resource.Resource {
	return &clientScopeResource{}
}

type clientScopeResource struct {
	client *client.Client
}

type clientScopeResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	Realm                      types.String `tfsdk:"realm"`
	ScopeID                    types.String `tfsdk:"scope_id"`
	Name                       types.String `tfsdk:"name"`
	Description                types.String `tfsdk:"description"`
	Protocol                   types.String `tfsdk:"protocol"`
	IsDefault                  types.Bool   `tfsdk:"is_default"`
	DynamicRegistrationAllowed types.Bool   `tfsdk:"dynamic_registration_allowed"`
}

func (r *clientScopeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_scope"
}

func (r *clientScopeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a client scope within a FerrisKey realm. Realms are seeded with built-in " +
			"scopes (`openid`, `profile`, `email`, ...); this resource manages additional scopes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{scope_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the scope belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"scope_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the client scope.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Scope name (the value used in the OAuth2 `scope` parameter).",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Scope description.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "Protocol of the scope. Defaults to `openid-connect`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("openid-connect"),
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Whether the scope is a realm-level default scope (`DEFAULT` rather than `OPTIONAL`). Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"dynamic_registration_allowed": schema.BoolAttribute{
				MarkdownDescription: "Whether the scope may be used by dynamically registered clients. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *clientScopeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientScopeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientScopeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	created, err := r.client.CreateClientScope(ctx, realm, client.CreateClientScopeRequest{
		Name:                       plan.Name.ValueString(),
		Description:                strPtr(plan.Description),
		Protocol:                   plan.Protocol.ValueString(),
		IsDefault:                  plan.IsDefault.ValueBool(),
		DynamicRegistrationAllowed: plan.DynamicRegistrationAllowed.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating client scope", err.Error())
		return
	}

	scopeFlatten(realm, created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientScopeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientScopeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	scope, err := r.client.GetClientScope(ctx, realm, state.ScopeID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading client scope", err.Error())
		return
	}

	scopeFlatten(realm, scope, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientScopeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state clientScopeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	updated, err := r.client.UpdateClientScope(ctx, realm, state.ScopeID.ValueString(), client.UpdateClientScopeRequest{
		Name:                       strPtr(plan.Name),
		Description:                strPtr(plan.Description),
		Protocol:                   strPtr(plan.Protocol),
		IsDefault:                  boolPtr(plan.IsDefault),
		DynamicRegistrationAllowed: boolPtr(plan.DynamicRegistrationAllowed),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating client scope", err.Error())
		return
	}

	scopeFlatten(realm, updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientScopeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientScopeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteClientScope(ctx, state.Realm.ValueString(), state.ScopeID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting client scope", err.Error())
	}
}

func (r *clientScopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope_id"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

func scopeFlatten(realm string, scope *client.ClientScope, m *clientScopeResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ScopeID = types.StringValue(scope.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: scope.ID}.String())
	m.Name = types.StringValue(scope.Name)
	m.Description = stringFromPtr(scope.Description)
	m.Protocol = types.StringValue(scope.Protocol)
	m.IsDefault = types.BoolValue(scope.DefaultScopeType == "DEFAULT")
	m.DynamicRegistrationAllowed = types.BoolValue(scope.DynamicRegistrationAllowed)
}
