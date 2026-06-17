package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &realmSettingsResource{}
	_ resource.ResourceWithConfigure   = &realmSettingsResource{}
	_ resource.ResourceWithImportState = &realmSettingsResource{}
)

// NewRealmSettingsResource is the resource factory.
func NewRealmSettingsResource() resource.Resource {
	return &realmSettingsResource{}
}

type realmSettingsResource struct {
	client *client.Client
}

type realmSettingsResourceModel struct {
	Realm                     types.String `tfsdk:"realm"`
	ID                        types.String `tfsdk:"id"`
	AccessTokenLifetime       types.Int64  `tfsdk:"access_token_lifetime"`
	RefreshTokenLifetime      types.Int64  `tfsdk:"refresh_token_lifetime"`
	IDTokenLifetime           types.Int64  `tfsdk:"id_token_lifetime"`
	TemporaryTokenLifetime    types.Int64  `tfsdk:"temporary_token_lifetime"`
	UserRegistrationEnabled   types.Bool   `tfsdk:"user_registration_enabled"`
	ForgotPasswordEnabled     types.Bool   `tfsdk:"forgot_password_enabled"`
	RememberMeEnabled         types.Bool   `tfsdk:"remember_me_enabled"`
	MagicLinkEnabled          types.Bool   `tfsdk:"magic_link_enabled"`
	MagicLinkTTL              types.Int64  `tfsdk:"magic_link_ttl"`
	PasskeyEnabled            types.Bool   `tfsdk:"passkey_enabled"`
	CompassEnabled            types.Bool   `tfsdk:"compass_enabled"`
	EmailVerificationEnabled  types.Bool   `tfsdk:"email_verification_enabled"`
	EmailVerificationTTLHours types.Int64  `tfsdk:"email_verification_ttl_hours"`
	DefaultSigningAlgorithm   types.String `tfsdk:"default_signing_algorithm"`
	UpdatedAt                 types.String `tfsdk:"updated_at"`
}

func (r *realmSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_realm_settings"
}

func (r *realmSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	boolAttr := func(d string) schema.Attribute {
		return schema.BoolAttribute{
			MarkdownDescription: d, Optional: true, Computed: true,
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		}
	}
	intAttr := func(d string) schema.Attribute {
		return schema.Int64Attribute{
			MarkdownDescription: d, Optional: true, Computed: true,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the security settings of a FerrisKey realm (token lifetimes, login flows, " +
			"email verification). Updates are a partial merge: a value omitted from the configuration keeps its " +
			"current server value rather than resetting. Deleting this resource stops managing the settings but " +
			"does not change them (a realm always has settings).",
		Attributes: map[string]schema.Attribute{
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm whose settings are managed. Also the resource ID. " +
					"Changing it forces a new resource.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (equal to `realm`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"access_token_lifetime":        intAttr("Access token lifetime (seconds)."),
			"refresh_token_lifetime":       intAttr("Refresh token lifetime (seconds)."),
			"id_token_lifetime":            intAttr("ID token lifetime (seconds)."),
			"temporary_token_lifetime":     intAttr("Temporary token lifetime (seconds)."),
			"user_registration_enabled":    boolAttr("Whether self-service registration is enabled."),
			"forgot_password_enabled":      boolAttr("Whether the forgot-password flow is enabled."),
			"remember_me_enabled":          boolAttr("Whether 'remember me' is enabled."),
			"magic_link_enabled":           boolAttr("Whether magic-link login is enabled."),
			"magic_link_ttl":               intAttr("Magic-link time-to-live (seconds)."),
			"passkey_enabled":              boolAttr("Whether passkey login is enabled."),
			"compass_enabled":              boolAttr("Whether Compass analytics are enabled."),
			"email_verification_enabled":   boolAttr("Whether email verification is enabled."),
			"email_verification_ttl_hours": intAttr("Email verification token lifetime (hours)."),
			"default_signing_algorithm": schema.StringAttribute{
				MarkdownDescription: "Default token signing algorithm (e.g. `RS256`).",
				Optional:            true,
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

func (r *realmSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

// buildRequest maps the plan into a partial update request (only known values).
func buildRealmSettingsRequest(m realmSettingsResourceModel) client.UpdateRealmSettingsRequest {
	req := client.UpdateRealmSettingsRequest{
		AccessTokenLifetime:       int64Ptr(m.AccessTokenLifetime),
		RefreshTokenLifetime:      int64Ptr(m.RefreshTokenLifetime),
		IDTokenLifetime:           int64Ptr(m.IDTokenLifetime),
		TemporaryTokenLifetime:    int64Ptr(m.TemporaryTokenLifetime),
		UserRegistrationEnabled:   boolPtr(m.UserRegistrationEnabled),
		ForgotPasswordEnabled:     boolPtr(m.ForgotPasswordEnabled),
		RememberMeEnabled:         boolPtr(m.RememberMeEnabled),
		MagicLinkEnabled:          boolPtr(m.MagicLinkEnabled),
		PasskeyEnabled:            boolPtr(m.PasskeyEnabled),
		CompassEnabled:            boolPtr(m.CompassEnabled),
		EmailVerificationEnabled:  boolPtr(m.EmailVerificationEnabled),
		EmailVerificationTTLHours: int64Ptr(m.EmailVerificationTTLHours),
		DefaultSigningAlgorithm:   strPtr(m.DefaultSigningAlgorithm),
	}
	if v := int64Ptr(m.MagicLinkTTL); v != nil {
		ttl := int32(*v)
		req.MagicLinkTTL = &ttl
	}
	return req
}

func (r *realmSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan realmSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	settings, err := r.client.UpdateRealmSettings(ctx, plan.Realm.ValueString(), buildRealmSettingsRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating realm settings", err.Error())
		return
	}
	r.flatten(plan.Realm.ValueString(), settings, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *realmSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state realmSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	settings, err := r.client.GetRealmSettings(ctx, state.Realm.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading realm settings", err.Error())
		return
	}
	if settings == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.flatten(state.Realm.ValueString(), settings, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *realmSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan realmSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	settings, err := r.client.UpdateRealmSettings(ctx, plan.Realm.ValueString(), buildRealmSettingsRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating realm settings", err.Error())
		return
	}
	r.flatten(plan.Realm.ValueString(), settings, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op: a realm always has settings, so there is nothing to remove.
// The resource simply stops being managed.
func (r *realmSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *realmSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *realmSettingsResource) flatten(realm string, s *client.RealmSetting, m *realmSettingsResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ID = types.StringValue(realm)
	m.AccessTokenLifetime = types.Int64Value(s.AccessTokenLifetime)
	m.RefreshTokenLifetime = types.Int64Value(s.RefreshTokenLifetime)
	m.IDTokenLifetime = types.Int64Value(s.IDTokenLifetime)
	m.TemporaryTokenLifetime = types.Int64Value(s.TemporaryTokenLifetime)
	m.UserRegistrationEnabled = types.BoolValue(s.UserRegistrationEnabled)
	m.ForgotPasswordEnabled = types.BoolValue(s.ForgotPasswordEnabled)
	m.RememberMeEnabled = types.BoolValue(s.RememberMeEnabled)
	m.MagicLinkEnabled = types.BoolValue(s.MagicLinkEnabled)
	m.MagicLinkTTL = types.Int64Value(int64(s.MagicLinkTTL))
	m.PasskeyEnabled = types.BoolValue(s.PasskeyEnabled)
	m.CompassEnabled = types.BoolValue(s.CompassEnabled)
	m.EmailVerificationEnabled = types.BoolValue(s.EmailVerificationEnabled)
	m.EmailVerificationTTLHours = types.Int64Value(s.EmailVerificationTTL)
	m.DefaultSigningAlgorithm = stringFromPtr(s.DefaultSigningAlgorithm)
	m.UpdatedAt = types.StringValue(s.UpdatedAt.Format(timeLayout))
}
