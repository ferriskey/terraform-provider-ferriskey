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
	_ resource.Resource                = &authzSchemaResource{}
	_ resource.ResourceWithConfigure   = &authzSchemaResource{}
	_ resource.ResourceWithImportState = &authzSchemaResource{}
)

func NewAuthzSchemaResource() resource.Resource {
	return &authzSchemaResource{}
}

type authzSchemaResource struct {
	client *client.Client
}

type authzSchemaResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Realm     types.String `tfsdk:"realm"`
	Namespace types.String `tfsdk:"namespace"`
	Source    types.String `tfsdk:"source"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *authzSchemaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authz_schema"
}

func (r *authzSchemaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the Cedar schema of a FerrisKey realm (singleton per realm). The schema must " +
			"declare its types and actions inside its `namespace` (default `App`, never `FerrisKey`) and is merged " +
			"server-side with the built-in schema (see the `ferriskey_authz_builtin_schema` data source). The API " +
			"refuses a schema that makes an enabled policy or template invalid, so create policies after the schema " +
			"and update them before narrowing it. The API has **no delete endpoint** for schemas: destroying this " +
			"resource only removes it from Terraform state and leaves the schema stored on the server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (equal to `realm`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm whose schema is managed. Changing it forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Cedar namespace the schema declares its types in. A Cedar identifier other than " +
					"`FerrisKey`. Defaults to `App` server-side.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Cedar schema text (the realm's own schema, without the built-in types). Stored and returned verbatim.",
				Required:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last schema replacement, as reported by the API.",
				Computed:            true,
			},
		},
	}
}

func (r *authzSchemaResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *authzSchemaResource) apply(ctx context.Context, plan *authzSchemaResourceModel) error {
	realm := plan.Realm.ValueString()
	authzWriteMu.Lock()
	s, err := r.client.PutAuthzSchema(ctx, realm, client.ReplaceAuthzSchemaRequest{
		Namespace: strPtr(plan.Namespace),
		Source:    plan.Source.ValueString(),
	})
	authzWriteMu.Unlock()
	if err != nil {
		return err
	}
	r.flatten(realm, s, plan)
	return nil
}

func (r *authzSchemaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan authzSchemaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error creating authorization schema", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *authzSchemaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state authzSchemaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, err := r.client.GetAuthzSchema(ctx, state.Realm.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading authorization schema", err.Error())
		return
	}
	r.flatten(state.Realm.ValueString(), s, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *authzSchemaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan authzSchemaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating authorization schema", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *authzSchemaResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *authzSchemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *authzSchemaResource) flatten(realm string, s *client.AuthzSchema, m *authzSchemaResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ID = types.StringValue(realm)
	m.Namespace = types.StringValue(s.Namespace)
	m.Source = types.StringValue(s.Source)
	m.UpdatedAt = types.StringValue(s.UpdatedAt)
}
