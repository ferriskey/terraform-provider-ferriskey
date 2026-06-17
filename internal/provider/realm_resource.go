package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &realmResource{}
	_ resource.ResourceWithConfigure   = &realmResource{}
	_ resource.ResourceWithImportState = &realmResource{}
)

// NewRealmResource is the resource factory.
func NewRealmResource() resource.Resource {
	return &realmResource{}
}

type realmResource struct {
	client *client.Client
}

type realmResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	RealmID   types.String `tfsdk:"realm_id"`
	Settings  types.Object `tfsdk:"settings"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *realmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_realm"
}

func (r *realmResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a FerrisKey realm. A realm is the root container for clients, users, " +
			"roles and organizations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Terraform identifier for the realm (equal to `name`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Unique realm name. Renaming the realm in place is supported.",
				Required:            true,
			},
			"realm_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the realm.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Server-managed realm settings (read-only). Configure these through the " +
					"FerrisKey admin surfaces; they are surfaced here for reference.",
				Computed:   true,
				Attributes: realmSettingsSchemaAttributes(),
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 creation timestamp.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 last-update timestamp.",
				Computed:            true,
			},
		},
	}
}

func (r *realmResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *realmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan realmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm, err := r.client.CreateRealm(ctx, client.CreateRealmRequest{Name: plan.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error creating realm", err.Error())
		return
	}

	r.flatten(realm, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *realmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state realmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm, err := r.client.GetRealm(ctx, state.Name.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading realm", err.Error())
		return
	}

	r.flatten(realm, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *realmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state realmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The realm is addressed by its current (state) name; the new name is in
	// the plan.
	realm, err := r.client.UpdateRealm(ctx, state.Name.ValueString(), client.UpdateRealmRequest{Name: plan.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error updating realm", err.Error())
		return
	}

	r.flatten(realm, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *realmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state realmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRealm(ctx, state.Name.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting realm", err.Error())
	}
}

func (r *realmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Realms are imported by name.
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// flatten maps an API realm onto the Terraform model.
func (r *realmResource) flatten(realm *client.Realm, m *realmResourceModel) {
	m.ID = types.StringValue(realm.Name)
	m.Name = types.StringValue(realm.Name)
	m.RealmID = types.StringValue(realm.ID)
	m.CreatedAt = types.StringValue(realm.CreatedAt.Format(timeLayout))
	m.UpdatedAt = types.StringValue(realm.UpdatedAt.Format(timeLayout))
	m.Settings = flattenRealmSettings(realm.Settings)
}

// realmSettingsAttrTypes is the attribute-type map for the settings object.
func realmSettingsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                           types.StringType,
		"realm_id":                     types.StringType,
		"default_signing_algorithm":    types.StringType,
		"user_registration_enabled":    types.BoolType,
		"forgot_password_enabled":      types.BoolType,
		"remember_me_enabled":          types.BoolType,
		"magic_link_enabled":           types.BoolType,
		"magic_link_ttl":               types.Int64Type,
		"passkey_enabled":              types.BoolType,
		"compass_enabled":              types.BoolType,
		"access_token_lifetime":        types.Int64Type,
		"refresh_token_lifetime":       types.Int64Type,
		"id_token_lifetime":            types.Int64Type,
		"temporary_token_lifetime":     types.Int64Type,
		"email_verification_enabled":   types.BoolType,
		"email_verification_ttl_hours": types.Int64Type,
		"updated_at":                   types.StringType,
	}
}

func flattenRealmSettings(s *client.RealmSetting) types.Object {
	if s == nil {
		return types.ObjectNull(realmSettingsAttrTypes())
	}
	signing := types.StringNull()
	if s.DefaultSigningAlgorithm != nil {
		signing = types.StringValue(*s.DefaultSigningAlgorithm)
	}
	obj, _ := types.ObjectValue(realmSettingsAttrTypes(), map[string]attr.Value{
		"id":                           types.StringValue(s.ID),
		"realm_id":                     types.StringValue(s.RealmID),
		"default_signing_algorithm":    signing,
		"user_registration_enabled":    types.BoolValue(s.UserRegistrationEnabled),
		"forgot_password_enabled":      types.BoolValue(s.ForgotPasswordEnabled),
		"remember_me_enabled":          types.BoolValue(s.RememberMeEnabled),
		"magic_link_enabled":           types.BoolValue(s.MagicLinkEnabled),
		"magic_link_ttl":               types.Int64Value(s.MagicLinkTTL),
		"passkey_enabled":              types.BoolValue(s.PasskeyEnabled),
		"compass_enabled":              types.BoolValue(s.CompassEnabled),
		"access_token_lifetime":        types.Int64Value(s.AccessTokenLifetime),
		"refresh_token_lifetime":       types.Int64Value(s.RefreshTokenLifetime),
		"id_token_lifetime":            types.Int64Value(s.IDTokenLifetime),
		"temporary_token_lifetime":     types.Int64Value(s.TemporaryTokenLifetime),
		"email_verification_enabled":   types.BoolValue(s.EmailVerificationEnabled),
		"email_verification_ttl_hours": types.Int64Value(s.EmailVerificationTTL),
		"updated_at":                   types.StringValue(s.UpdatedAt.Format(timeLayout)),
	})
	return obj
}

// realmSettingsSchemaAttributes builds the (all-computed) schema for the nested
// settings object, shared by the realm resource and data source.
func realmSettingsSchemaAttributes() map[string]schema.Attribute {
	str := func(d string) schema.Attribute { return schema.StringAttribute{Computed: true, MarkdownDescription: d} }
	b := func(d string) schema.Attribute { return schema.BoolAttribute{Computed: true, MarkdownDescription: d} }
	i := func(d string) schema.Attribute { return schema.Int64Attribute{Computed: true, MarkdownDescription: d} }
	return map[string]schema.Attribute{
		"id":                           str("Settings UUID."),
		"realm_id":                     str("Owning realm UUID."),
		"default_signing_algorithm":    str("Default token signing algorithm."),
		"user_registration_enabled":    b("Whether self-service user registration is enabled."),
		"forgot_password_enabled":      b("Whether the forgot-password flow is enabled."),
		"remember_me_enabled":          b("Whether 'remember me' is enabled."),
		"magic_link_enabled":           b("Whether magic-link login is enabled."),
		"magic_link_ttl":               i("Magic-link time-to-live (seconds)."),
		"passkey_enabled":              b("Whether passkey login is enabled."),
		"compass_enabled":              b("Whether Compass analytics are enabled."),
		"access_token_lifetime":        i("Access token lifetime (seconds)."),
		"refresh_token_lifetime":       i("Refresh token lifetime (seconds)."),
		"id_token_lifetime":            i("ID token lifetime (seconds)."),
		"temporary_token_lifetime":     i("Temporary token lifetime (seconds)."),
		"email_verification_enabled":   b("Whether email verification is enabled."),
		"email_verification_ttl_hours": i("Email verification token lifetime (hours)."),
		"updated_at":                   str("RFC3339 last-update timestamp of the settings."),
	}
}
