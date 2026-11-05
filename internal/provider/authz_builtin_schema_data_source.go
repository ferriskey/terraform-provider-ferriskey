package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ datasource.DataSource              = &authzBuiltinSchemaDataSource{}
	_ datasource.DataSourceWithConfigure = &authzBuiltinSchemaDataSource{}
)

func NewAuthzBuiltinSchemaDataSource() datasource.DataSource {
	return &authzBuiltinSchemaDataSource{}
}

type authzBuiltinSchemaDataSource struct {
	client *client.Client
}

type authzBuiltinSchemaDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Realm     types.String `tfsdk:"realm"`
	Namespace types.String `tfsdk:"namespace"`
	Source    types.String `tfsdk:"source"`
}

func (d *authzBuiltinSchemaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authz_builtin_schema"
}

func (d *authzBuiltinSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the built-in `FerrisKey` Cedar schema (users, groups, roles, organizations, agents " +
			"and MCP tools) that every realm schema is merged with. Useful to reference its types when writing a " +
			"`ferriskey_authz_schema`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Data source identifier (equal to `realm`).",
				Computed:            true,
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm to query.",
				Required:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Namespace of the built-in schema (`FerrisKey`).",
				Computed:            true,
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Cedar text of the built-in schema.",
				Computed:            true,
			},
		},
	}
}

func (d *authzBuiltinSchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	d.client = c
}

func (d *authzBuiltinSchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data authzBuiltinSchemaDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	s, err := d.client.GetAuthzBuiltinSchema(ctx, data.Realm.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading built-in authorization schema", err.Error())
		return
	}

	data.ID = data.Realm
	data.Namespace = types.StringValue(s.Namespace)
	data.Source = types.StringValue(s.Source)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
