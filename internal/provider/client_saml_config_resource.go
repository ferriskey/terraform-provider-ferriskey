package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &clientSamlConfigResource{}
	_ resource.ResourceWithConfigure   = &clientSamlConfigResource{}
	_ resource.ResourceWithImportState = &clientSamlConfigResource{}
)

const samlmNameIDEmail = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"

func samlmUUIDRealmPlan() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace()}
}

func samlmIDPlan() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
}

// NewClientSamlConfigResource is the resource factory.
func NewClientSamlConfigResource() resource.Resource {
	return &clientSamlConfigResource{}
}

type clientSamlConfigResource struct {
	client *client.Client
}

type clientSamlConfigResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Realm                   types.String `tfsdk:"realm"`
	ClientUUID              types.String `tfsdk:"client_uuid"`
	SpEntityID              types.String `tfsdk:"sp_entity_id"`
	AcsURL                  types.String `tfsdk:"acs_url"`
	NameIDFormat            types.String `tfsdk:"name_id_format"`
	SignAssertions          types.Bool   `tfsdk:"sign_assertions"`
	SignDocuments           types.Bool   `tfsdk:"sign_documents"`
	WantAuthnRequestsSigned types.Bool   `tfsdk:"want_authn_requests_signed"`
}

func (r *clientSamlConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_saml_config"
}

func (r *clientSamlConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the SAML service-provider configuration of a FerrisKey client (one per client, " +
			"the client must use the `saml` protocol). The API stores the whole configuration on every write, so " +
			"omitted optional attributes fall back to their defaults. The API has no delete endpoint for this " +
			"configuration: destroying the resource only removes it from Terraform state and leaves the " +
			"configuration in place on the server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier in the form `{realm}/{client_uuid}`.",
				Computed:            true,
				PlanModifiers:       samlmIDPlan(),
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm owning the client. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       samlmUUIDRealmPlan(),
			},
			"client_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the SAML client (`ferriskey_client.<name>.client_uuid`). Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       samlmUUIDRealmPlan(),
			},
			"sp_entity_id": schema.StringAttribute{
				MarkdownDescription: "Entity ID of the service provider (an absolute URI such as `urn:example:sp` or a URL).",
				Required:            true,
			},
			"acs_url": schema.StringAttribute{
				MarkdownDescription: "Assertion Consumer Service URL of the service provider (absolute `http` or `https` URL).",
				Required:            true,
			},
			"name_id_format": schema.StringAttribute{
				MarkdownDescription: "SAML NameID format URN. One of `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress` " +
					"(default), `urn:oasis:names:tc:SAML:2.0:nameid-format:persistent`, " +
					"`urn:oasis:names:tc:SAML:2.0:nameid-format:transient` or " +
					"`urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified`.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(samlmNameIDEmail),
				Validators: []validator.String{stringvalidator.OneOf(
					samlmNameIDEmail,
					"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
					"urn:oasis:names:tc:SAML:2.0:nameid-format:transient",
					"urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified",
				)},
			},
			"sign_assertions": schema.BoolAttribute{
				MarkdownDescription: "Whether SAML assertions are signed. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"sign_documents": schema.BoolAttribute{
				MarkdownDescription: "Whether whole SAML response documents are signed. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"want_authn_requests_signed": schema.BoolAttribute{
				MarkdownDescription: "Whether the service provider is required to sign its authentication requests. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *clientSamlConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientSamlConfigResource) upsert(ctx context.Context, plan *clientSamlConfigResourceModel) error {
	cfg, err := r.client.SetClientSamlConfig(ctx, plan.Realm.ValueString(), plan.ClientUUID.ValueString(), client.SetClientSamlConfigRequest{
		SpEntityID:              plan.SpEntityID.ValueString(),
		AcsURL:                  plan.AcsURL.ValueString(),
		NameIDFormat:            plan.NameIDFormat.ValueString(),
		SignAssertions:          plan.SignAssertions.ValueBool(),
		SignDocuments:           plan.SignDocuments.ValueBool(),
		WantAuthnRequestsSigned: plan.WantAuthnRequestsSigned.ValueBool(),
	})
	if err != nil {
		return err
	}
	r.flatten(plan.Realm.ValueString(), cfg, plan)
	return nil
}

func (r *clientSamlConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientSamlConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error creating client SAML config", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientSamlConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientSamlConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetClientSamlConfig(ctx, state.Realm.ValueString(), state.ClientUUID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading client SAML config", err.Error())
		return
	}
	r.flatten(state.Realm.ValueString(), cfg, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientSamlConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clientSamlConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating client SAML config", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientSamlConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *clientSamlConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, "realm", "client_uuid")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_uuid"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *clientSamlConfigResource) flatten(realm string, cfg *client.ClientSamlConfig, m *clientSamlConfigResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ClientUUID = types.StringValue(cfg.ClientID)
	m.ID = types.StringValue(realm + "/" + cfg.ClientID)
	m.SpEntityID = types.StringValue(cfg.SpEntityID)
	m.AcsURL = types.StringValue(cfg.AcsURL)
	m.NameIDFormat = types.StringValue(cfg.NameIDFormat)
	m.SignAssertions = types.BoolValue(cfg.SignAssertions)
	m.SignDocuments = types.BoolValue(cfg.SignDocuments)
	m.WantAuthnRequestsSigned = types.BoolValue(cfg.WantAuthnRequestsSigned)
}
