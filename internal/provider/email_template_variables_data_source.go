package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ datasource.DataSource              = &emailTemplateVariablesDataSource{}
	_ datasource.DataSourceWithConfigure = &emailTemplateVariablesDataSource{}
)

func NewEmailTemplateVariablesDataSource() datasource.DataSource {
	return &emailTemplateVariablesDataSource{}
}

type emailTemplateVariablesDataSource struct {
	client *client.Client
}

type emailTemplateVariableModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type emailTemplateVariablesDataSourceModel struct {
	ID        types.String                 `tfsdk:"id"`
	EmailType types.String                 `tfsdk:"email_type"`
	Variables []emailTemplateVariableModel `tfsdk:"variables"`
}

func (d *emailTemplateVariablesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_template_variables"
}

func (d *emailTemplateVariablesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the variables that can be referenced in an email template of a given type. The list is global and not realm-scoped.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Equal to `email_type`.",
				Computed:            true,
			},
			"email_type": schema.StringAttribute{
				MarkdownDescription: "Email type: `reset_password`, `magic_link` or `email_verification`.",
				Required:            true,
			},
			"variables": schema.ListNestedAttribute{
				MarkdownDescription: "Available template variables.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Variable name, e.g. `user.first_name`.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Human-readable description of the variable.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *emailTemplateVariablesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	d.client = c
}

func (d *emailTemplateVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg emailTemplateVariablesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vars, err := d.client.GetEmailTemplateVariables(ctx, cfg.EmailType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading email template variables", err.Error())
		return
	}

	cfg.ID = cfg.EmailType
	cfg.Variables = make([]emailTemplateVariableModel, 0, len(vars))
	for _, v := range vars {
		cfg.Variables = append(cfg.Variables, emailTemplateVariableModel{
			Name:        types.StringValue(v.Name),
			Description: types.StringValue(v.Description),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
