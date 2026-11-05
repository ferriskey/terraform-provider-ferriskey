package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

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

	ResetPasswordTemplateID     types.String `tfsdk:"reset_password_template_id"`
	MagicLinkTemplateID         types.String `tfsdk:"magic_link_template_id"`
	EmailVerificationTemplateID types.String `tfsdk:"email_verification_template_id"`
	LoginAliases                types.List   `tfsdk:"login_aliases"`
	DefaultLocale               types.String `tfsdk:"default_locale"`
	SupportedLocales            types.Set    `tfsdk:"supported_locales"`
	RequireMFA                  types.Bool   `tfsdk:"require_mfa"`
	EditUsernameEnabled         types.Bool   `tfsdk:"edit_username_enabled"`
	LockoutThreshold            types.Int64  `tfsdk:"lockout_threshold"`
	LockoutDurationSeconds      types.Int64  `tfsdk:"lockout_duration_seconds"`
	SeawatchPIIMode             types.String `tfsdk:"seawatch_pii_mode"`
	SeawatchPseudoKey           types.String `tfsdk:"seawatch_pseudo_key"`
	WebhookRetryMaxAttempts     types.Int64  `tfsdk:"webhook_retry_max_attempts"`
	WebhookRetryBaseDelayMs     types.Int64  `tfsdk:"webhook_retry_base_delay_ms"`
	WebhookRetryMaxDelayMs      types.Int64  `tfsdk:"webhook_retry_max_delay_ms"`
	WebhookRetryMaxTotalDelayMs types.Int64  `tfsdk:"webhook_retry_max_total_delay_ms"`
	CimdEnabled                 types.Bool   `tfsdk:"cimd_enabled"`
	DcrEnabled                  types.Bool   `tfsdk:"dcr_enabled"`
	CimdAllowedHosts            types.Set    `tfsdk:"cimd_allowed_hosts"`
	AllowedResources            types.Set    `tfsdk:"allowed_resources"`
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
	strAttr := func(d string, v ...validator.String) schema.Attribute {
		return schema.StringAttribute{
			MarkdownDescription: d, Optional: true, Computed: true, Validators: v,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	setAttr := func(d string) schema.Attribute {
		return schema.SetAttribute{
			MarkdownDescription: d, Optional: true, Computed: true, ElementType: types.StringType,
			PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
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
			"reset_password_template_id":     strAttr("UUID of the `ferriskey_email_template` used for password reset emails."),
			"magic_link_template_id":         strAttr("UUID of the `ferriskey_email_template` used for magic-link emails."),
			"email_verification_template_id": strAttr("UUID of the `ferriskey_email_template` used for email verification."),
			"login_aliases": schema.ListAttribute{
				MarkdownDescription: "Ordered login identifiers the realm accepts (`username`, `email`). The first alias " +
					"that resolves a user wins.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.OneOf("username", "email")),
				},
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"default_locale":           strAttr("Default UI locale (e.g. `en`)."),
			"supported_locales":        setAttr("Locales users can choose from."),
			"require_mfa":              boolAttr("Whether every user must enroll a second factor."),
			"edit_username_enabled":    boolAttr("Whether users may change their own username from the account console."),
			"lockout_threshold":        intAttr("Failed login attempts before an account is locked."),
			"lockout_duration_seconds": intAttr("How long a locked account stays locked (seconds)."),
			"seawatch_pii_mode": strAttr("How personal data appears in Seawatch security events: `off`, `mask` or `pseudonymise`.",
				stringvalidator.OneOf("off", "mask", "pseudonymise")),
			"seawatch_pseudo_key": schema.StringAttribute{
				MarkdownDescription: "HMAC key used when `seawatch_pii_mode` is `pseudonymise`. Write-only: the API never " +
					"returns it, so changes made outside Terraform are not detected.",
				Optional:  true,
				Sensitive: true,
			},
			"webhook_retry_max_attempts":       intAttr("Maximum webhook delivery attempts. Server default when unset."),
			"webhook_retry_base_delay_ms":      intAttr("Base delay between webhook retries (milliseconds)."),
			"webhook_retry_max_delay_ms":       intAttr("Maximum delay between webhook retries (milliseconds)."),
			"webhook_retry_max_total_delay_ms": intAttr("Maximum total time spent retrying a webhook delivery (milliseconds)."),
			"cimd_enabled":                     boolAttr("Whether clients may register through Client ID Metadata Documents."),
			"dcr_enabled":                      boolAttr("Whether OAuth 2.0 Dynamic Client Registration is enabled."),
			"cimd_allowed_hosts":               setAttr("Hosts allowed to serve Client ID Metadata Documents."),
			"allowed_resources":                setAttr("Resource indicators (RFC 8707) clients may request. Required before using `ferriskey_resource_owners`."),
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

// buildRealmSettingsRequest maps the plan into a partial update request (only
// known values).
func buildRealmSettingsRequest(ctx context.Context, m realmSettingsResourceModel) (client.UpdateRealmSettingsRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
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

		ResetPasswordTemplateID:     strPtr(m.ResetPasswordTemplateID),
		MagicLinkTemplateID:         strPtr(m.MagicLinkTemplateID),
		EmailVerificationTemplateID: strPtr(m.EmailVerificationTemplateID),
		DefaultLocale:               strPtr(m.DefaultLocale),
		RequireMFA:                  boolPtr(m.RequireMFA),
		EditUsernameEnabled:         boolPtr(m.EditUsernameEnabled),
		LockoutThreshold:            int64Ptr(m.LockoutThreshold),
		LockoutDurationSeconds:      int64Ptr(m.LockoutDurationSeconds),
		SeawatchPIIMode:             strPtr(m.SeawatchPIIMode),
		SeawatchPseudoKey:           strPtr(m.SeawatchPseudoKey),
		WebhookRetryMaxAttempts:     int64Ptr(m.WebhookRetryMaxAttempts),
		WebhookRetryBaseDelayMs:     int64Ptr(m.WebhookRetryBaseDelayMs),
		WebhookRetryMaxDelayMs:      int64Ptr(m.WebhookRetryMaxDelayMs),
		WebhookRetryMaxTotalDelayMs: int64Ptr(m.WebhookRetryMaxTotalDelayMs),
		CimdEnabled:                 boolPtr(m.CimdEnabled),
		DcrEnabled:                  boolPtr(m.DcrEnabled),
	}
	if v := int64Ptr(m.MagicLinkTTL); v != nil {
		ttl := int32(*v)
		req.MagicLinkTTL = &ttl
	}
	if !m.LoginAliases.IsNull() && !m.LoginAliases.IsUnknown() {
		aliases := []string{}
		diags.Append(m.LoginAliases.ElementsAs(ctx, &aliases, false)...)
		req.LoginAliases = &aliases
	}
	req.SupportedLocales = realmSettingsSetPtr(ctx, m.SupportedLocales, &diags)
	req.CimdAllowedHosts = realmSettingsSetPtr(ctx, m.CimdAllowedHosts, &diags)
	req.AllowedResources = realmSettingsSetPtr(ctx, m.AllowedResources, &diags)
	return req, diags
}

// realmSettingsSetPtr returns nil for a null/unknown set (leave unchanged) and
// a pointer to a possibly empty slice otherwise, so `[]` clears the list.
func realmSettingsSetPtr(ctx context.Context, set types.Set, diags *diag.Diagnostics) *[]string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	values, d := stringSlice(ctx, set)
	diags.Append(d...)
	if values == nil {
		values = []string{}
	}
	return &values
}

func (r *realmSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan realmSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, d := buildRealmSettingsRequest(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	settings, err := r.client.UpdateRealmSettings(ctx, plan.Realm.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating realm settings", err.Error())
		return
	}
	resp.Diagnostics.Append(r.flatten(plan.Realm.ValueString(), settings, &plan)...)
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
	resp.Diagnostics.Append(r.flatten(state.Realm.ValueString(), settings, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *realmSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan realmSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, d := buildRealmSettingsRequest(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	settings, err := r.client.UpdateRealmSettings(ctx, plan.Realm.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating realm settings", err.Error())
		return
	}
	resp.Diagnostics.Append(r.flatten(plan.Realm.ValueString(), settings, &plan)...)
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

func (r *realmSettingsResource) flatten(realm string, s *client.RealmSetting, m *realmSettingsResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
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

	m.ResetPasswordTemplateID = stringFromPtr(s.ResetPasswordTemplateID)
	m.MagicLinkTemplateID = stringFromPtr(s.MagicLinkTemplateID)
	m.EmailVerificationTemplateID = stringFromPtr(s.EmailVerificationTemplateID)
	m.DefaultLocale = types.StringValue(s.DefaultLocale)
	m.RequireMFA = types.BoolValue(s.RequireMFA)
	m.EditUsernameEnabled = types.BoolValue(s.EditUsernameEnabled)
	m.LockoutThreshold = types.Int64Value(s.LockoutThreshold)
	m.LockoutDurationSeconds = types.Int64Value(s.LockoutDurationSeconds)
	m.SeawatchPIIMode = types.StringValue(s.SeawatchPIIMode)
	m.WebhookRetryMaxAttempts = int64FromPtr(s.WebhookRetryMaxAttempts)
	m.WebhookRetryBaseDelayMs = int64FromPtr(s.WebhookRetryBaseDelayMs)
	m.WebhookRetryMaxDelayMs = int64FromPtr(s.WebhookRetryMaxDelayMs)
	m.WebhookRetryMaxTotalDelayMs = int64FromPtr(s.WebhookRetryMaxTotalDelayMs)
	m.CimdEnabled = types.BoolValue(s.CimdEnabled)
	m.DcrEnabled = types.BoolValue(s.DcrEnabled)

	aliases := make([]attr.Value, 0, len(s.LoginAliases))
	for _, a := range s.LoginAliases {
		aliases = append(aliases, types.StringValue(a))
	}
	list, d := types.ListValue(types.StringType, aliases)
	diags.Append(d...)
	m.LoginAliases = list
	for _, f := range []struct {
		dst    *types.Set
		values []string
	}{
		{&m.SupportedLocales, s.SupportedLocales},
		{&m.CimdAllowedHosts, s.CimdAllowedHosts},
		{&m.AllowedResources, s.AllowedResources},
	} {
		set, d := stringSetValue(f.values)
		diags.Append(d...)
		*f.dst = set
	}
	return diags
}
