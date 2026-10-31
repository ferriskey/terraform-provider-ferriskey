package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &clientTokenExchangePolicyResource{}
	_ resource.ResourceWithConfigure   = &clientTokenExchangePolicyResource{}
	_ resource.ResourceWithImportState = &clientTokenExchangePolicyResource{}
)

func NewClientTokenExchangePolicyResource() resource.Resource {
	return &clientTokenExchangePolicyResource{}
}

type clientTokenExchangePolicyResource struct {
	client *client.Client
}

type clientTokenExchangePolicyResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Realm              types.String `tfsdk:"realm"`
	ClientID           types.String `tfsdk:"client_id"`
	PolicyID           types.String `tfsdk:"policy_id"`
	TargetAudience     types.String `tfsdk:"target_audience"`
	AllowedScopes      types.Set    `tfsdk:"allowed_scopes"`
	AllowImpersonation types.Bool   `tfsdk:"allow_impersonation"`
	AllowDelegation    types.Bool   `tfsdk:"allow_delegation"`
}

func (r *clientTokenExchangePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_token_exchange_policy"
}

func (r *clientTokenExchangePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a token exchange policy of a FerrisKey client, allowing it to exchange tokens for a target audience. The API has no update endpoint, so any change forces a new resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{client_id}/{policy_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the client belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the requesting client the policy belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"policy_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the policy.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"target_audience": schema.StringAttribute{
				MarkdownDescription: "The `client_id` (not the UUID) of another client in the same realm that tokens may be exchanged for. It cannot be the requesting client itself. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"allowed_scopes": schema.SetAttribute{
				MarkdownDescription: "Ceiling of scopes the exchanged token may carry. Omit for no ceiling; an empty set is rejected by the API. Changing this forces a new resource.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
			"allow_impersonation": schema.BoolAttribute{
				MarkdownDescription: "Whether impersonation exchanges are allowed. Defaults to `false`. Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"allow_delegation": schema.BoolAttribute{
				MarkdownDescription: "Whether delegation exchanges are allowed. Defaults to `false`. Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *clientTokenExchangePolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientTokenExchangePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientTokenExchangePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes, d := stringSlice(ctx, plan.AllowedScopes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	clientID := plan.ClientID.ValueString()
	created, err := r.client.CreateClientTokenExchangePolicy(ctx, realm, clientID, client.CreateClientTokenExchangePolicyRequest{
		TargetAudience:     plan.TargetAudience.ValueString(),
		AllowedScopes:      scopes,
		AllowImpersonation: plan.AllowImpersonation.ValueBool(),
		AllowDelegation:    plan.AllowDelegation.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating token exchange policy", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, clientID, created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientTokenExchangePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientTokenExchangePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	clientID := state.ClientID.ValueString()
	item, err := r.client.GetClientTokenExchangePolicy(ctx, realm, clientID, state.PolicyID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading token exchange policy", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, clientID, item, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientTokenExchangePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clientTokenExchangePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientTokenExchangePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientTokenExchangePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteClientTokenExchangePolicy(ctx, state.Realm.ValueString(), state.ClientID.ValueString(), state.PolicyID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting token exchange policy", err.Error())
	}
}

func (r *clientTokenExchangePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	clientExtImport(ctx, req.ID, "policy_id", resp)
}

func (r *clientTokenExchangePolicyResource) flatten(realm, clientID string, item *client.ClientTokenExchangePolicy, m *clientTokenExchangePolicyResourceModel) diag.Diagnostics {
	m.Realm = types.StringValue(realm)
	m.ClientID = types.StringValue(clientID)
	m.PolicyID = types.StringValue(item.ID)
	m.ID = types.StringValue(clientExtID{Realm: realm, ClientID: clientID, ID: item.ID}.String())
	m.TargetAudience = types.StringValue(item.TargetAudience)
	m.AllowImpersonation = types.BoolValue(item.AllowImpersonation)
	m.AllowDelegation = types.BoolValue(item.AllowDelegation)
	if item.AllowedScopes == nil {
		m.AllowedScopes = types.SetNull(types.StringType)
		return nil
	}
	set, d := stringSetValue(item.AllowedScopes)
	m.AllowedScopes = set
	return d
}
