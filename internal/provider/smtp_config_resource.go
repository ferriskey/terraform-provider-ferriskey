package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &smtpConfigResource{}
	_ resource.ResourceWithConfigure   = &smtpConfigResource{}
	_ resource.ResourceWithImportState = &smtpConfigResource{}
)

// NewSmtpConfigResource is the resource factory.
func NewSmtpConfigResource() resource.Resource {
	return &smtpConfigResource{}
}

type smtpConfigResource struct {
	client *client.Client
}

type smtpConfigResourceModel struct {
	Realm      types.String `tfsdk:"realm"`
	ID         types.String `tfsdk:"id"`
	Host       types.String `tfsdk:"host"`
	Port       types.Int64  `tfsdk:"port"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	FromEmail  types.String `tfsdk:"from_email"`
	FromName   types.String `tfsdk:"from_name"`
	Encryption types.String `tfsdk:"encryption"`
}

func (r *smtpConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_smtp_config"
}

func (r *smtpConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a FerrisKey realm's SMTP configuration. The `password` is write-only — the " +
			"API never returns it, so it is kept from configuration and cannot be detected as drifted.",
		Attributes: map[string]schema.Attribute{
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm whose SMTP config is managed. Also the resource ID. Changing it forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (equal to `realm`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"host":       schema.StringAttribute{MarkdownDescription: "SMTP server host.", Required: true},
			"port":       schema.Int64Attribute{MarkdownDescription: "SMTP server port.", Required: true},
			"username":   schema.StringAttribute{MarkdownDescription: "SMTP username.", Required: true},
			"password":   schema.StringAttribute{MarkdownDescription: "SMTP password (write-only).", Required: true, Sensitive: true},
			"from_email": schema.StringAttribute{MarkdownDescription: "From address.", Required: true},
			"from_name":  schema.StringAttribute{MarkdownDescription: "From display name.", Required: true},
			"encryption": schema.StringAttribute{MarkdownDescription: "Transport encryption (e.g. `tls`, `ssl`, `none`).", Required: true},
		},
	}
}

func (r *smtpConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *smtpConfigResource) upsert(ctx context.Context, plan *smtpConfigResourceModel) error {
	cfg, err := r.client.UpsertSmtpConfig(ctx, plan.Realm.ValueString(), client.UpsertSmtpConfigRequest{
		Host:       plan.Host.ValueString(),
		Port:       plan.Port.ValueInt64(),
		Username:   plan.Username.ValueString(),
		Password:   plan.Password.ValueString(),
		FromEmail:  plan.FromEmail.ValueString(),
		FromName:   plan.FromName.ValueString(),
		Encryption: plan.Encryption.ValueString(),
	})
	if err != nil {
		return err
	}
	// Preserve the write-only password from the plan; flatten the rest.
	password := plan.Password
	r.flatten(plan.Realm.ValueString(), cfg, plan)
	plan.Password = password
	return nil
}

func (r *smtpConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan smtpConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error creating SMTP config", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *smtpConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state smtpConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetSmtpConfig(ctx, state.Realm.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading SMTP config", err.Error())
		return
	}
	// Password is never returned; keep the value already in state.
	password := state.Password
	r.flatten(state.Realm.ValueString(), cfg, &state)
	state.Password = password
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *smtpConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan smtpConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating SMTP config", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *smtpConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state smtpConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSmtpConfig(ctx, state.Realm.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting SMTP config", err.Error())
	}
}

func (r *smtpConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Note: password cannot be imported (write-only); set it in config after import.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *smtpConfigResource) flatten(realm string, c *client.SmtpConfig, m *smtpConfigResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ID = types.StringValue(realm)
	m.Host = types.StringValue(c.Host)
	m.Port = types.Int64Value(c.Port)
	m.Username = types.StringValue(c.Username)
	m.FromEmail = types.StringValue(c.FromEmail)
	m.FromName = types.StringValue(c.FromName)
	m.Encryption = types.StringValue(c.Encryption)
}
