package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

// Ensure FerrisKeyProvider satisfies the provider.Provider interface.
var _ provider.Provider = &FerrisKeyProvider{}

// Environment variables that override the corresponding provider arguments.
const (
	envURL          = "FERRISKEY_URL"
	envRealm        = "FERRISKEY_REALM"
	envUsername     = "FERRISKEY_USERNAME"
	envPassword     = "FERRISKEY_PASSWORD" //nolint:gosec // env var name, not a credential
	envClientID     = "FERRISKEY_CLIENT_ID"
	envClientSecret = "FERRISKEY_CLIENT_SECRET" //nolint:gosec // env var name, not a credential
	envScope        = "FERRISKEY_SCOPE"
	envCACert       = "FERRISKEY_CA_CERT"
	envTLSInsecure  = "FERRISKEY_TLS_INSECURE"
)

// FerrisKeyProvider is the provider implementation.
type FerrisKeyProvider struct {
	version string
}

// FerrisKeyProviderModel maps the provider configuration block.
type FerrisKeyProviderModel struct {
	URL          types.String `tfsdk:"url"`
	Realm        types.String `tfsdk:"realm"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Scope        types.String `tfsdk:"scope"`
	CACert       types.String `tfsdk:"ca_cert"`
	TLSInsecure  types.Bool   `tfsdk:"tls_insecure_skip_verify"`
}

// New returns a provider factory bound to a build version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &FerrisKeyProvider{version: version}
	}
}

func (p *FerrisKeyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ferriskey"
	resp.Version = p.version
}

func (p *FerrisKeyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The FerrisKey provider manages the configuration of a FerrisKey CIAM/IAM " +
			"instance (realms, clients, users, roles, organizations, ...) through its REST API. " +
			"It authenticates with OAuth2 using either the password grant (bootstrap) or the " +
			"client credentials grant (steady state).",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the FerrisKey instance, e.g. `https://auth.example.com`. " +
					"May also be set with the `" + envURL + "` environment variable.",
				Optional: true,
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm used for authentication (often the admin/`master` realm). This is " +
					"the realm whose token endpoint issues the access token, not necessarily the realm the " +
					"managed resources live in. May also be set with the `" + envRealm + "` environment variable.",
				Optional: true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for the OAuth2 password grant (bootstrap phase). Requires " +
					"`client_id` and `password`. May also be set with the `" + envUsername + "` environment variable.",
				Optional: true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for the OAuth2 password grant. May also be set with the `" +
					envPassword + "` environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client ID. With `username`/`password` this is the public client " +
					"(e.g. `admin-cli`); with `client_secret` this is the confidential service account client. " +
					"May also be set with the `" + envClientID + "` environment variable.",
				Optional: true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client secret for the client credentials grant (steady state). " +
					"Presence selects the client credentials grant. May also be set with the `" +
					envClientSecret + "` environment variable.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					// client_secret and password select mutually exclusive
					// grants; reject configuring both at once.
					stringvalidator.ConflictsWith(path.MatchRoot("password")),
				},
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "Optional OAuth2 scope requested when fetching tokens. May also be set " +
					"with the `" + envScope + "` environment variable.",
				Optional: true,
			},
			"ca_cert": schema.StringAttribute{
				MarkdownDescription: "Optional PEM-encoded CA certificate (or bundle) to trust, for FerrisKey " +
					"instances served by a private CA. Use `file(\"ca.pem\")` to load from disk. May also be set " +
					"with the `" + envCACert + "` environment variable.",
				Optional: true,
			},
			"tls_insecure_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "Disable TLS certificate verification. **Development only** — never use " +
					"against production. May also be set with the `" + envTLSInsecure + "` environment variable.",
				Optional: true,
			},
		},
	}
}

func (p *FerrisKeyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg FerrisKeyProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve each value: explicit config wins, otherwise fall back to the
	// environment variable. This keeps secrets out of .tf files and makes CI
	// configuration trivial.
	url := resolve(cfg.URL, envURL)
	realm := resolve(cfg.Realm, envRealm)
	username := resolve(cfg.Username, envUsername)
	password := resolve(cfg.Password, envPassword)
	clientID := resolve(cfg.ClientID, envClientID)
	clientSecret := resolve(cfg.ClientSecret, envClientSecret)
	scope := resolve(cfg.Scope, envScope)
	caCert := resolve(cfg.CACert, envCACert)
	tlsInsecure := resolveBool(cfg.TLSInsecure, envTLSInsecure)

	if url == "" {
		resp.Diagnostics.AddAttributeError(path.Root("url"),
			"Missing FerrisKey URL",
			"The provider requires a FerrisKey instance URL, set either in the `url` argument or the "+envURL+" environment variable.")
	}
	if realm == "" {
		resp.Diagnostics.AddAttributeError(path.Root("realm"),
			"Missing authentication realm",
			"The provider requires an authentication realm, set either in the `realm` argument or the "+envRealm+" environment variable.")
	}
	if clientID == "" {
		resp.Diagnostics.AddAttributeError(path.Root("client_id"),
			"Missing client_id",
			"The provider requires a `client_id`, set either in the argument or the "+envClientID+" environment variable.")
	}

	// Decide the grant type from the supplied credentials.
	var grant client.GrantType
	switch {
	case clientSecret != "" && password == "":
		grant = client.GrantClientCredentials
	case password != "" && clientSecret == "":
		grant = client.GrantPassword
		if username == "" {
			resp.Diagnostics.AddAttributeError(path.Root("username"),
				"Missing username for password grant",
				"When authenticating with the password grant, `username` is required (or the "+envUsername+" environment variable).")
		}
	case password != "" && clientSecret != "":
		resp.Diagnostics.AddError(
			"Ambiguous authentication configuration",
			"Both `password` and `client_secret` are set. Provide either `username`+`password` (password grant) "+
				"or `client_id`+`client_secret` (client credentials grant), not both.")
	default:
		resp.Diagnostics.AddError(
			"Missing credentials",
			"No usable credentials were found. Provide either `username`+`password` (password grant) or "+
				"`client_id`+`client_secret` (client credentials grant), via arguments or "+envPassword+"/"+envClientSecret+" environment variables.")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Build a TLS-aware HTTP client when custom CA / insecure verification is
	// requested; otherwise leave it nil so the client uses its default.
	var httpClient *http.Client
	tlsOpts := client.TLSOptions{CACertPEM: caCert, InsecureSkipVerify: tlsInsecure}
	if !tlsOpts.IsZero() {
		if tlsInsecure {
			resp.Diagnostics.AddWarning(
				"TLS certificate verification disabled",
				"`tls_insecure_skip_verify` is enabled; FerrisKey's TLS certificate will not be verified. "+
					"Use this only in development.")
		}
		hc, err := client.HTTPClientWithTLS(tlsOpts)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("ca_cert"), "Invalid CA certificate", err.Error())
			return
		}
		httpClient = hc
	}

	c := client.New(client.Config{
		URL: url,
		Auth: client.AuthConfig{
			Realm:        realm,
			GrantType:    grant,
			Username:     username,
			Password:     password,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scope:        scope,
		},
		HTTPClient: httpClient,
		UserAgent:  fmt.Sprintf("terraform-provider-ferriskey/%s", p.version),
	})

	// Fail fast with a clear error if credentials or connectivity are wrong,
	// rather than surfacing the failure on the first resource operation.
	if err := c.Ping(ctx, realm); err != nil {
		resp.Diagnostics.AddError(
			"Unable to authenticate with FerrisKey",
			fmt.Sprintf("Could not obtain an access token from %s (realm %q): %s", url, realm, err))
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *FerrisKeyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRealmResource,
		NewRealmSettingsResource,
		NewClientResource,
		NewUserResource,
		NewRoleResource,
		NewOrganizationResource,
		NewUserRoleResource,
		NewPasswordPolicyResource,
		NewSmtpConfigResource,
		NewOrganizationMemberResource,
		NewIdentityProviderResource,
	}
}

func (p *FerrisKeyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRealmDataSource,
		NewRoleDataSource,
		NewOpenIDConfigurationDataSource,
	}
}

// resolve returns the configured value if known and non-null, otherwise the
// environment variable's value.
func resolve(v types.String, env string) string {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueString()
	}
	return strings.TrimSpace(os.Getenv(env))
}

// resolveBool resolves a bool argument, falling back to the environment
// variable (parsed leniently: "1"/"true"/"yes" are true).
func resolveBool(v types.Bool, env string) bool {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueBool()
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(env))) {
	case "1", "true", "yes", "on":
		return true
	default:
		if b, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(env))); err == nil {
			return b
		}
		return false
	}
}
