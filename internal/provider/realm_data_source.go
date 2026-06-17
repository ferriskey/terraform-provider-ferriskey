package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ datasource.DataSource              = &realmDataSource{}
	_ datasource.DataSourceWithConfigure = &realmDataSource{}
)

// NewRealmDataSource is the data source factory.
func NewRealmDataSource() datasource.DataSource {
	return &realmDataSource{}
}

type realmDataSource struct {
	client *client.Client
}

type realmDataSourceModel struct {
	Name      types.String `tfsdk:"name"`
	RealmID   types.String `tfsdk:"realm_id"`
	Settings  types.Object `tfsdk:"settings"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (d *realmDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_realm"
}

func (d *realmDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing FerrisKey realm by name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the realm to read.",
				Required:            true,
			},
			"realm_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the realm.",
				Computed:            true,
			},
			"settings": schema.SingleNestedAttribute{
				MarkdownDescription: "Server-managed realm settings.",
				Computed:            true,
				Attributes:          realmSettingsSchemaAttributesDataSource(),
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 creation timestamp.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 last-update timestamp.",
				Computed:            true,
			},
		},
	}
}

func (d *realmDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	d.client = c
}

func (d *realmDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data realmDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm, err := d.client.GetRealm(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading realm", err.Error())
		return
	}

	data.Name = types.StringValue(realm.Name)
	data.RealmID = types.StringValue(realm.ID)
	data.CreatedAt = types.StringValue(realm.CreatedAt.Format(timeLayout))
	data.UpdatedAt = types.StringValue(realm.UpdatedAt.Format(timeLayout))
	data.Settings = flattenRealmSettings(realm.Settings)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// realmSettingsSchemaAttributesDataSource mirrors realmSettingsSchemaAttributes
// but for the data source schema package (the attribute types differ between
// the resource and data source schema packages, so they cannot be shared).
func realmSettingsSchemaAttributesDataSource() map[string]schema.Attribute {
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
