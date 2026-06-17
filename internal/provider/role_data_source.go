package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ datasource.DataSource              = &roleDataSource{}
	_ datasource.DataSourceWithConfigure = &roleDataSource{}
)

// NewRoleDataSource is the data source factory.
func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

type roleDataSource struct {
	client *client.Client
}

type roleDataSourceModel struct {
	Realm       types.String `tfsdk:"realm"`
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	Permissions types.Set    `tfsdk:"permissions"`
}

func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing realm role by name. Useful to reference built-in admin roles " +
			"(e.g. `staff`, `master-realm`) when granting a service account its permissions.",
		Attributes: map[string]schema.Attribute{
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm the role belongs to.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the role to look up.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the role.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Role description.",
				Computed:            true,
			},
			"permissions": schema.SetAttribute{
				MarkdownDescription: "Permissions granted by the role.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	d.client = c
}

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := d.client.FindRoleByName(ctx, data.Realm.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error looking up role", err.Error())
		return
	}

	data.ID = types.StringValue(role.ID)
	data.Description = stringFromPtr(role.Description)
	set, diags := stringSetValue(role.Permissions)
	resp.Diagnostics.Append(diags...)
	data.Permissions = set

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
