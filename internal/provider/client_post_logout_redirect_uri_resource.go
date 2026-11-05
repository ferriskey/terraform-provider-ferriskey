package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &clientPostLogoutRedirectURIResource{}
	_ resource.ResourceWithConfigure   = &clientPostLogoutRedirectURIResource{}
	_ resource.ResourceWithImportState = &clientPostLogoutRedirectURIResource{}
)

func NewClientPostLogoutRedirectURIResource() resource.Resource {
	return &clientPostLogoutRedirectURIResource{}
}

type clientPostLogoutRedirectURIResource struct {
	client *client.Client
}

type clientPostLogoutRedirectURIResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Realm    types.String `tfsdk:"realm"`
	ClientID types.String `tfsdk:"client_id"`
	URIID    types.String `tfsdk:"uri_id"`
	Value    types.String `tfsdk:"value"`
	Enabled  types.Bool   `tfsdk:"enabled"`
}

func (r *clientPostLogoutRedirectURIResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_post_logout_redirect_uri"
}

func (r *clientPostLogoutRedirectURIResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single post-logout redirect URI of a FerrisKey client. Only `enabled` can be updated in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{client_id}/{uri_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the client belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the client the redirect URI belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"uri_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the post-logout redirect URI.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The redirect URI. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the redirect URI is enabled. Defaults to `false`, matching the API default.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *clientPostLogoutRedirectURIResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientPostLogoutRedirectURIResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientPostLogoutRedirectURIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	clientID := plan.ClientID.ValueString()
	created, err := r.client.CreateClientPostLogoutRedirectURI(ctx, realm, clientID, client.CreateClientPostLogoutRedirectURIRequest{
		Value:   plan.Value.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating post-logout redirect URI", err.Error())
		return
	}

	r.flatten(realm, clientID, created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientPostLogoutRedirectURIResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientPostLogoutRedirectURIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	clientID := state.ClientID.ValueString()
	item, err := r.client.GetClientPostLogoutRedirectURI(ctx, realm, clientID, state.URIID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading post-logout redirect URI", err.Error())
		return
	}

	r.flatten(realm, clientID, item, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientPostLogoutRedirectURIResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state clientPostLogoutRedirectURIResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	clientID := plan.ClientID.ValueString()
	updated, err := r.client.UpdateClientPostLogoutRedirectURI(ctx, realm, clientID, state.URIID.ValueString(), client.UpdateClientPostLogoutRedirectURIRequest{
		Enabled: plan.Enabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating post-logout redirect URI", err.Error())
		return
	}

	r.flatten(realm, clientID, updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientPostLogoutRedirectURIResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientPostLogoutRedirectURIResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteClientPostLogoutRedirectURI(ctx, state.Realm.ValueString(), state.ClientID.ValueString(), state.URIID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting post-logout redirect URI", err.Error())
	}
}

func (r *clientPostLogoutRedirectURIResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	clientExtImport(ctx, req.ID, "uri_id", resp)
}

func (r *clientPostLogoutRedirectURIResource) flatten(realm, clientID string, item *client.ClientPostLogoutRedirectURI, m *clientPostLogoutRedirectURIResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ClientID = types.StringValue(clientID)
	m.URIID = types.StringValue(item.ID)
	m.ID = types.StringValue(clientExtID{Realm: realm, ClientID: clientID, ID: item.ID}.String())
	m.Value = types.StringValue(item.Value)
	m.Enabled = types.BoolValue(item.Enabled)
}
