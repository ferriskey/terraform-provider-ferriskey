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
	_ resource.Resource                = &authzStateResource{}
	_ resource.ResourceWithConfigure   = &authzStateResource{}
	_ resource.ResourceWithImportState = &authzStateResource{}
)

func NewAuthzStateResource() resource.Resource {
	return &authzStateResource{}
}

type authzStateResource struct {
	client *client.Client
}

type authzStateResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Realm   types.String `tfsdk:"realm"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Version types.Int64  `tfsdk:"version"`
}

func (r *authzStateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authz_state"
}

func (r *authzStateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Enables or disables Cedar authorization for a FerrisKey realm (singleton per realm). " +
			"Destroying the resource sets `enabled` back to `false`, the default state of a realm. The API requires " +
			"the server to run with `AUTHZ_ENABLED=true`; otherwise every authorization endpoint answers 404.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (equal to `realm`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm whose authorization state is managed. Changing it forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the realm answers authorization questions.",
				Required:            true,
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "Counter the server increments every time the realm's authorization rules change.",
				Computed:            true,
			},
		},
	}
}

func (r *authzStateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *authzStateResource) apply(ctx context.Context, plan *authzStateResourceModel) error {
	realm := plan.Realm.ValueString()
	authzWriteMu.Lock()
	s, err := r.client.SetAuthzState(ctx, realm, plan.Enabled.ValueBool())
	authzWriteMu.Unlock()
	if err != nil {
		return err
	}
	r.flatten(realm, s, plan)
	return nil
}

func (r *authzStateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan authzStateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error setting authorization state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *authzStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state authzStateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, err := r.client.GetAuthzState(ctx, state.Realm.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading authorization state", err.Error())
		return
	}
	r.flatten(state.Realm.ValueString(), s, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *authzStateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan authzStateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error setting authorization state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *authzStateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state authzStateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	authzWriteMu.Lock()
	_, err := r.client.SetAuthzState(ctx, state.Realm.ValueString(), false)
	authzWriteMu.Unlock()
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error disabling authorization", err.Error())
	}
}

func (r *authzStateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *authzStateResource) flatten(realm string, s *client.AuthzState, m *authzStateResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ID = types.StringValue(realm)
	m.Enabled = types.BoolValue(s.Enabled)
	m.Version = types.Int64Value(s.Version)
}
