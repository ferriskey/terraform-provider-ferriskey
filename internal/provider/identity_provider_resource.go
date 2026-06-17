package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &identityProviderResource{}
	_ resource.ResourceWithConfigure   = &identityProviderResource{}
	_ resource.ResourceWithImportState = &identityProviderResource{}
)

// NewIdentityProviderResource is the resource factory.
func NewIdentityProviderResource() resource.Resource {
	return &identityProviderResource{}
}

type identityProviderResource struct {
	client *client.Client
}

type identityProviderResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Realm                     types.String `tfsdk:"realm"`
	Alias                     types.String `tfsdk:"alias"`
	InternalID                types.String `tfsdk:"internal_id"`
	ProviderID                types.String `tfsdk:"provider_id"`
	DisplayName               types.String `tfsdk:"display_name"`
	Enabled                   types.Bool   `tfsdk:"enabled"`
	StoreToken                types.Bool   `tfsdk:"store_token"`
	AddReadTokenRoleOnCreate  types.Bool   `tfsdk:"add_read_token_role_on_create"`
	TrustEmail                types.Bool   `tfsdk:"trust_email"`
	LinkOnly                  types.Bool   `tfsdk:"link_only"`
	FirstBrokerLoginFlowAlias types.String `tfsdk:"first_broker_login_flow_alias"`
	PostBrokerLoginFlowAlias  types.String `tfsdk:"post_broker_login_flow_alias"`
	Config                    types.Map    `tfsdk:"config"`
}

func (r *identityProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_provider"
}

func (r *identityProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	rr := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	boolFlag := func(d string) schema.Attribute {
		return schema.BoolAttribute{MarkdownDescription: d, Optional: true, Computed: true,
			Default: booldefault.StaticBool(false), PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a federation identity provider (broker) within a FerrisKey realm.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{alias}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm the identity provider belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       rr,
			},
			"alias": schema.StringAttribute{
				MarkdownDescription: "Unique alias of the identity provider. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       rr,
			},
			"internal_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"provider_id": schema.StringAttribute{
				MarkdownDescription: "Provider type, e.g. `oidc` or `saml`. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       rr,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable display name.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the identity provider is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"store_token":                   boolFlag("Whether tokens from the IdP are stored."),
			"add_read_token_role_on_create": boolFlag("Whether to grant the read-token role on account creation."),
			"trust_email":                   boolFlag("Whether the IdP's email is trusted (skips verification)."),
			"link_only":                     boolFlag("Whether the IdP can only be used to link existing accounts."),
			"first_broker_login_flow_alias": schema.StringAttribute{
				MarkdownDescription: "Authentication flow alias for first broker login.",
				Optional:            true,
			},
			"post_broker_login_flow_alias": schema.StringAttribute{
				MarkdownDescription: "Authentication flow alias for post broker login.",
				Optional:            true,
			},
			"config": schema.MapAttribute{
				MarkdownDescription: "Provider-specific configuration (e.g. `clientId`, `clientSecret`, " +
					"`authorizationUrl`). Only the keys you set are managed; keys the server adds are ignored.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *identityProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *identityProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan identityProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, d := stringMap(ctx, plan.Config)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	idp, err := r.client.CreateIdentityProvider(ctx, plan.Realm.ValueString(), client.CreateIdentityProviderRequest{
		Alias:                     plan.Alias.ValueString(),
		ProviderID:                plan.ProviderID.ValueString(),
		DisplayName:               strPtr(plan.DisplayName),
		Enabled:                   boolPtr(plan.Enabled),
		Config:                    cfg,
		FirstBrokerLoginFlowAlias: strPtr(plan.FirstBrokerLoginFlowAlias),
		PostBrokerLoginFlowAlias:  strPtr(plan.PostBrokerLoginFlowAlias),
		StoreToken:                boolPtr(plan.StoreToken),
		AddReadTokenRoleOnCreate:  boolPtr(plan.AddReadTokenRoleOnCreate),
		TrustEmail:                boolPtr(plan.TrustEmail),
		LinkOnly:                  boolPtr(plan.LinkOnly),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating identity provider", err.Error())
		return
	}
	// Keep the configured `config` verbatim on write: Terraform requires the
	// post-apply value to equal the plan, and the server may echo it differently.
	r.flattenScalars(idp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *identityProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state identityProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	idp, err := r.client.GetIdentityProvider(ctx, state.Realm.ValueString(), state.Alias.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading identity provider", err.Error())
		return
	}
	r.flattenScalars(idp, &state)
	// On read, reconcile config from the server but keep only the keys we manage
	// (the server augments config with extra null-valued keys).
	cfg, d := filterStringMap(state.Config, idp.Config)
	resp.Diagnostics.Append(d...)
	state.Config = cfg
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *identityProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan identityProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, d := stringMap(ctx, plan.Config)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateIdentityProvider(ctx, plan.Realm.ValueString(), plan.Alias.ValueString(), client.UpdateIdentityProviderRequest{
		DisplayName:               strPtr(plan.DisplayName),
		Enabled:                   boolPtr(plan.Enabled),
		Config:                    cfg,
		FirstBrokerLoginFlowAlias: strPtr(plan.FirstBrokerLoginFlowAlias),
		PostBrokerLoginFlowAlias:  strPtr(plan.PostBrokerLoginFlowAlias),
		StoreToken:                boolPtr(plan.StoreToken),
		AddReadTokenRoleOnCreate:  boolPtr(plan.AddReadTokenRoleOnCreate),
		TrustEmail:                boolPtr(plan.TrustEmail),
		LinkOnly:                  boolPtr(plan.LinkOnly),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating identity provider", err.Error())
		return
	}
	// The PUT response body is empty/partial, so re-fetch the full object.
	idp, err := r.client.GetIdentityProvider(ctx, plan.Realm.ValueString(), plan.Alias.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading identity provider after update", err.Error())
		return
	}
	r.flattenScalars(idp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *identityProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state identityProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteIdentityProvider(ctx, state.Realm.ValueString(), state.Alias.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting identity provider", err.Error())
	}
}

func (r *identityProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			err.Error()+" (expected \"{realm}/{alias}\")")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("alias"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

// flattenScalars maps the non-config fields of the API object onto the model.
// Config is handled separately by each operation (kept verbatim on write,
// filtered from the server on read).
func (r *identityProviderResource) flattenScalars(idp *client.IdentityProvider, m *identityProviderResourceModel) {
	m.Alias = types.StringValue(idp.Alias)
	m.ID = types.StringValue(realmScopedID{Realm: m.Realm.ValueString(), ID: idp.Alias}.String())
	m.InternalID = types.StringValue(idp.InternalID)
	m.ProviderID = types.StringValue(idp.ProviderID)
	m.DisplayName = stringFromPtr(idp.DisplayName)
	m.Enabled = types.BoolValue(idp.Enabled)
	m.StoreToken = types.BoolValue(idp.StoreToken)
	m.AddReadTokenRoleOnCreate = types.BoolValue(idp.AddReadTokenRoleOnCreate)
	m.TrustEmail = types.BoolValue(idp.TrustEmail)
	m.LinkOnly = types.BoolValue(idp.LinkOnly)
	m.FirstBrokerLoginFlowAlias = stringFromPtr(idp.FirstBrokerLoginFlowAlias)
	m.PostBrokerLoginFlowAlias = stringFromPtr(idp.PostBrokerLoginFlowAlias)
}
