package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &clientResource{}
	_ resource.ResourceWithConfigure   = &clientResource{}
	_ resource.ResourceWithImportState = &clientResource{}
)

// NewClientResource is the resource factory.
func NewClientResource() resource.Resource {
	return &clientResource{}
}

type clientResource struct {
	client *client.Client
}

type clientResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Realm                     types.String `tfsdk:"realm"`
	ClientUUID                types.String `tfsdk:"client_uuid"`
	ClientID                  types.String `tfsdk:"client_id"`
	Name                      types.String `tfsdk:"name"`
	ClientType                types.String `tfsdk:"client_type"`
	Protocol                  types.String `tfsdk:"protocol"`
	Enabled                   types.Bool   `tfsdk:"enabled"`
	PublicClient              types.Bool   `tfsdk:"public_client"`
	ServiceAccountEnabled     types.Bool   `tfsdk:"service_account_enabled"`
	ServiceAccountUserID      types.String `tfsdk:"service_account_user_id"`
	DirectAccessGrantsEnabled types.Bool   `tfsdk:"direct_access_grants_enabled"`
	Secret                    types.String `tfsdk:"secret"`
	AccessTokenLifetime       types.Int64  `tfsdk:"access_token_lifetime"`
	RefreshTokenLifetime      types.Int64  `tfsdk:"refresh_token_lifetime"`
	IDTokenLifetime           types.Int64  `tfsdk:"id_token_lifetime"`
	TemporaryTokenLifetime    types.Int64  `tfsdk:"temporary_token_lifetime"`
	RedirectURIs              types.Set    `tfsdk:"redirect_uris"`
}

func (r *clientResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client"
}

func (r *clientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a client (OAuth2/OIDC application) within a FerrisKey realm.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{uuid}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the client belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"client_uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the client.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client identifier (the public-facing `client_id`).",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable client name. Defaults to the `client_id` server-side when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"client_type": schema.StringAttribute{
				MarkdownDescription: "Client type: `confidential`, `public`, or `system`. Changing this forces a new resource.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("confidential", "public", "system"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "Client protocol. Defaults to `openid-connect`. Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("openid-connect"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the client is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"public_client": schema.BoolAttribute{
				MarkdownDescription: "Whether the client is a public client (no secret). Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplace(),
				},
			},
			"service_account_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether a service account is enabled for this client (client credentials grant). " +
					"Changing this forces a new resource.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplace(),
				},
			},
			"service_account_user_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the client's service account user (present only when " +
					"`service_account_enabled` is true). Assign roles to this user to grant the service " +
					"account permissions — see `ferriskey_user_role`.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"direct_access_grants_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the direct access grants (resource owner password) flow is enabled.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Client secret (confidential clients only). Server-generated and stored in state.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"access_token_lifetime": schema.Int64Attribute{
				MarkdownDescription: "Access token lifetime override (seconds). Server default when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"refresh_token_lifetime": schema.Int64Attribute{
				MarkdownDescription: "Refresh token lifetime override (seconds). Server default when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"id_token_lifetime": schema.Int64Attribute{
				MarkdownDescription: "ID token lifetime override (seconds). Server default when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"temporary_token_lifetime": schema.Int64Attribute{
				MarkdownDescription: "Temporary token lifetime override (seconds). Server default when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"redirect_uris": schema.SetAttribute{
				MarkdownDescription: "Set of allowed redirect URIs. Order is not significant. Omit to leave " +
					"existing redirect URIs unmanaged; set to `[]` to remove all.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *clientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()

	// The API requires a non-empty client name; mirror the documented server
	// behaviour of defaulting it to the client_id when the user omits it.
	name := plan.Name.ValueString()
	if plan.Name.IsNull() || plan.Name.IsUnknown() || name == "" {
		name = plan.ClientID.ValueString()
	}

	createReq := client.CreateClientRequest{
		ClientType:                plan.ClientType.ValueString(),
		ClientID:                  plan.ClientID.ValueString(),
		Name:                      name,
		Protocol:                  plan.Protocol.ValueString(),
		Enabled:                   boolPtr(plan.Enabled),
		PublicClient:              boolPtr(plan.PublicClient),
		ServiceAccountEnabled:     boolPtr(plan.ServiceAccountEnabled),
		DirectAccessGrantsEnabled: boolPtr(plan.DirectAccessGrantsEnabled),
	}

	created, err := r.client.CreateClient(ctx, realm, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating client", err.Error())
		return
	}

	// Token lifetimes are not part of the create body; apply them with a
	// follow-up PATCH when the configuration sets any of them.
	if needsLifetimePatch(plan) {
		updated, err := r.client.UpdateClient(ctx, realm, created.ID, client.UpdateClientRequest{
			AccessTokenLifetime:    int64Ptr(plan.AccessTokenLifetime),
			RefreshTokenLifetime:   int64Ptr(plan.RefreshTokenLifetime),
			IDTokenLifetime:        int64Ptr(plan.IDTokenLifetime),
			TemporaryTokenLifetime: int64Ptr(plan.TemporaryTokenLifetime),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error setting client token lifetimes", err.Error())
			return
		}
		created = updated
	}

	// Reconcile redirect URIs only when the configuration manages them (a known
	// set, possibly empty). A null/unknown set leaves them untouched.
	if !plan.RedirectURIs.IsNull() && !plan.RedirectURIs.IsUnknown() {
		desired, d := stringSlice(ctx, plan.RedirectURIs)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		uris, err := r.client.SetRedirectURIs(ctx, realm, created.ID, desired)
		if err != nil {
			resp.Diagnostics.AddError("Error setting client redirect URIs", err.Error())
			return
		}
		created.RedirectURIs = uris
	}

	resp.Diagnostics.Append(r.flatten(ctx, realm, created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	cl, err := r.client.GetClient(ctx, realm, state.ClientUUID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading client", err.Error())
		return
	}

	// The GET client payload may omit redirect URIs; fetch them explicitly so
	// Read is faithful and import is complete.
	uris, err := r.client.ListRedirectURIs(ctx, realm, cl.ID)
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Error reading client redirect URIs", err.Error())
		return
	}
	cl.RedirectURIs = uris

	resp.Diagnostics.Append(r.flatten(ctx, realm, cl, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state clientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	uuid := state.ClientUUID.ValueString()

	updated, err := r.client.UpdateClient(ctx, realm, uuid, client.UpdateClientRequest{
		ClientID:                  strPtr(plan.ClientID),
		Name:                      strPtr(plan.Name),
		Enabled:                   boolPtr(plan.Enabled),
		DirectAccessGrantsEnabled: boolPtr(plan.DirectAccessGrantsEnabled),
		AccessTokenLifetime:       int64Ptr(plan.AccessTokenLifetime),
		RefreshTokenLifetime:      int64Ptr(plan.RefreshTokenLifetime),
		IDTokenLifetime:           int64Ptr(plan.IDTokenLifetime),
		TemporaryTokenLifetime:    int64Ptr(plan.TemporaryTokenLifetime),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating client", err.Error())
		return
	}

	if !plan.RedirectURIs.IsNull() && !plan.RedirectURIs.IsUnknown() {
		desired, d := stringSlice(ctx, plan.RedirectURIs)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		uris, err := r.client.SetRedirectURIs(ctx, realm, uuid, desired)
		if err != nil {
			resp.Diagnostics.AddError("Error updating client redirect URIs", err.Error())
			return
		}
		updated.RedirectURIs = uris
	} else {
		// Preserve whatever the server reports so state stays faithful.
		uris, err := r.client.ListRedirectURIs(ctx, realm, uuid)
		if err != nil && !errors.Is(err, client.ErrNotFound) {
			resp.Diagnostics.AddError("Error reading client redirect URIs", err.Error())
			return
		}
		updated.RedirectURIs = uris
	}

	resp.Diagnostics.Append(r.flatten(ctx, realm, updated, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteClient(ctx, state.Realm.ValueString(), state.ClientUUID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting client", err.Error())
	}
}

func (r *clientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_uuid"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

// flatten maps an API client onto the Terraform model.
func (r *clientResource) flatten(ctx context.Context, realm string, cl *client.APIClient, m *clientResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.Realm = types.StringValue(realm)
	m.ClientUUID = types.StringValue(cl.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: cl.ID}.String())
	m.ClientID = types.StringValue(cl.ClientID)
	m.Name = types.StringValue(cl.Name)
	m.ClientType = types.StringValue(cl.ClientType)
	m.Protocol = types.StringValue(cl.Protocol)
	m.Enabled = types.BoolValue(cl.Enabled)
	m.PublicClient = types.BoolValue(cl.PublicClient)
	m.ServiceAccountEnabled = types.BoolValue(cl.ServiceAccountEnabled)
	m.DirectAccessGrantsEnabled = types.BoolValue(cl.DirectAccessGrantsEnabled)

	// Resolve the service account user when enabled, so callers can grant the
	// service account roles. The username is deterministic.
	m.ServiceAccountUserID = types.StringNull()
	if cl.ServiceAccountEnabled {
		sa, err := r.client.FindUserByUsername(ctx, realm, client.ServiceAccountUsername(cl.ClientID))
		if err == nil {
			m.ServiceAccountUserID = types.StringValue(sa.ID)
		} else if !errors.Is(err, client.ErrNotFound) {
			diags.AddError("Error resolving service account user", err.Error())
		}
	}
	m.Secret = stringFromPtr(cl.Secret)
	m.AccessTokenLifetime = int64FromPtr(cl.AccessTokenLifetime)
	m.RefreshTokenLifetime = int64FromPtr(cl.RefreshTokenLifetime)
	m.IDTokenLifetime = int64FromPtr(cl.IDTokenLifetime)
	m.TemporaryTokenLifetime = int64FromPtr(cl.TemporaryTokenLifetime)

	values := make([]string, 0, len(cl.RedirectURIs))
	for _, u := range cl.RedirectURIs {
		values = append(values, u.Value)
	}
	set, d := stringSetValue(values)
	diags.Append(d...)
	m.RedirectURIs = set

	return diags
}

func needsLifetimePatch(m clientResourceModel) bool {
	return int64Ptr(m.AccessTokenLifetime) != nil ||
		int64Ptr(m.RefreshTokenLifetime) != nil ||
		int64Ptr(m.IDTokenLifetime) != nil ||
		int64Ptr(m.TemporaryTokenLifetime) != nil
}
