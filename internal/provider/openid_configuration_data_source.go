package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ datasource.DataSource              = &openIDConfigurationDataSource{}
	_ datasource.DataSourceWithConfigure = &openIDConfigurationDataSource{}
)

// NewOpenIDConfigurationDataSource is the data source factory.
func NewOpenIDConfigurationDataSource() datasource.DataSource {
	return &openIDConfigurationDataSource{}
}

type openIDConfigurationDataSource struct {
	client *client.Client
}

type openIDConfigurationDataSourceModel struct {
	Realm                             types.String `tfsdk:"realm"`
	Issuer                            types.String `tfsdk:"issuer"`
	AuthorizationEndpoint             types.String `tfsdk:"authorization_endpoint"`
	TokenEndpoint                     types.String `tfsdk:"token_endpoint"`
	RevocationEndpoint                types.String `tfsdk:"revocation_endpoint"`
	EndSessionEndpoint                types.String `tfsdk:"end_session_endpoint"`
	IntrospectionEndpoint             types.String `tfsdk:"introspection_endpoint"`
	UserinfoEndpoint                  types.String `tfsdk:"userinfo_endpoint"`
	JwksURI                           types.String `tfsdk:"jwks_uri"`
	GrantTypesSupported               types.List   `tfsdk:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported types.List   `tfsdk:"token_endpoint_auth_methods_supported"`
}

func (d *openIDConfigurationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_openid_configuration"
}

func (d *openIDConfigurationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	str := func(desc string) schema.Attribute {
		return schema.StringAttribute{Computed: true, MarkdownDescription: desc}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the public OIDC discovery document (`.well-known/openid-configuration`) for a realm.",
		Attributes: map[string]schema.Attribute{
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm whose discovery document is read.",
				Required:            true,
			},
			"issuer":                 str("OIDC issuer identifier."),
			"authorization_endpoint": str("Authorization endpoint URL."),
			"token_endpoint":         str("Token endpoint URL."),
			"revocation_endpoint":    str("Token revocation endpoint URL."),
			"end_session_endpoint":   str("End-session (logout) endpoint URL."),
			"introspection_endpoint": str("Token introspection endpoint URL."),
			"userinfo_endpoint":      str("UserInfo endpoint URL."),
			"jwks_uri":               str("JSON Web Key Set URL."),
			"grant_types_supported": schema.ListAttribute{
				MarkdownDescription: "Supported OAuth2 grant types.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"token_endpoint_auth_methods_supported": schema.ListAttribute{
				MarkdownDescription: "Supported token endpoint authentication methods.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *openIDConfigurationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	d.client = c
}

func (d *openIDConfigurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data openIDConfigurationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oidc, err := d.client.GetOpenIDConfiguration(ctx, data.Realm.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading OIDC discovery document", err.Error())
		return
	}

	data.Issuer = types.StringValue(oidc.Issuer)
	data.AuthorizationEndpoint = types.StringValue(oidc.AuthorizationEndpoint)
	data.TokenEndpoint = types.StringValue(oidc.TokenEndpoint)
	data.RevocationEndpoint = types.StringValue(oidc.RevocationEndpoint)
	data.EndSessionEndpoint = types.StringValue(oidc.EndSessionEndpoint)
	data.IntrospectionEndpoint = types.StringValue(oidc.IntrospectionEndpoint)
	data.UserinfoEndpoint = types.StringValue(oidc.UserinfoEndpoint)
	data.JwksURI = types.StringValue(oidc.JwksURI)

	grantTypes, diags := types.ListValueFrom(ctx, types.StringType, oidc.GrantTypesSupported)
	resp.Diagnostics.Append(diags...)
	data.GrantTypesSupported = grantTypes

	authMethods, diags := types.ListValueFrom(ctx, types.StringType, oidc.TokenEndpointAuthMethodsSupported)
	resp.Diagnostics.Append(diags...)
	data.TokenEndpointAuthMethodsSupported = authMethods

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
