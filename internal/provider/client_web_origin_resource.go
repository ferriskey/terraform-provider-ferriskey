package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &clientWebOriginResource{}
	_ resource.ResourceWithConfigure   = &clientWebOriginResource{}
	_ resource.ResourceWithImportState = &clientWebOriginResource{}
)

func NewClientWebOriginResource() resource.Resource {
	return &clientWebOriginResource{}
}

type clientWebOriginResource struct {
	client *client.Client
}

type clientWebOriginResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Realm       types.String `tfsdk:"realm"`
	ClientID    types.String `tfsdk:"client_id"`
	WebOriginID types.String `tfsdk:"web_origin_id"`
	Value       types.String `tfsdk:"value"`
}

func (r *clientWebOriginResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_web_origin"
}

func (r *clientWebOriginResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single allowed web origin (CORS) of a FerrisKey client. The API has no update endpoint, so any change forces a new resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{client_id}/{web_origin_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the client belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the client the web origin belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"web_origin_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the web origin.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The allowed origin, e.g. `https://app.example.com`. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *clientWebOriginResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientWebOriginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientWebOriginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	clientID := plan.ClientID.ValueString()
	created, err := r.client.CreateClientWebOrigin(ctx, realm, clientID, client.CreateClientWebOriginRequest{
		Value: plan.Value.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating client web origin", err.Error())
		return
	}

	r.flatten(realm, clientID, created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientWebOriginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientWebOriginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	clientID := state.ClientID.ValueString()
	item, err := r.client.GetClientWebOrigin(ctx, realm, clientID, state.WebOriginID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading client web origin", err.Error())
		return
	}

	r.flatten(realm, clientID, item, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientWebOriginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clientWebOriginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientWebOriginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientWebOriginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteClientWebOrigin(ctx, state.Realm.ValueString(), state.ClientID.ValueString(), state.WebOriginID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting client web origin", err.Error())
	}
}

func (r *clientWebOriginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	clientExtImport(ctx, req.ID, "web_origin_id", resp)
}

func (r *clientWebOriginResource) flatten(realm, clientID string, item *client.ClientWebOrigin, m *clientWebOriginResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ClientID = types.StringValue(clientID)
	m.WebOriginID = types.StringValue(item.ID)
	m.ID = types.StringValue(clientExtID{Realm: realm, ClientID: clientID, ID: item.ID}.String())
	m.Value = types.StringValue(item.Value)
}
