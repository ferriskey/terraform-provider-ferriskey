package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &clientScopeProtocolMapperResource{}
	_ resource.ResourceWithConfigure   = &clientScopeProtocolMapperResource{}
	_ resource.ResourceWithImportState = &clientScopeProtocolMapperResource{}
)

// NewClientScopeProtocolMapperResource is the resource factory.
func NewClientScopeProtocolMapperResource() resource.Resource {
	return &clientScopeProtocolMapperResource{}
}

type clientScopeProtocolMapperResource struct {
	client *client.Client
}

type clientScopeProtocolMapperResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Realm      types.String `tfsdk:"realm"`
	ScopeID    types.String `tfsdk:"scope_id"`
	MapperID   types.String `tfsdk:"mapper_id"`
	Name       types.String `tfsdk:"name"`
	MapperType types.String `tfsdk:"mapper_type"`
	Config     types.Map    `tfsdk:"config"`
}

func (r *clientScopeProtocolMapperResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_scope_protocol_mapper"
}

func (r *clientScopeProtocolMapperResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a protocol mapper attached to a client scope. The API has no single-mapper " +
			"read endpoint, so the mapper is read through its parent client scope.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{scope_id}/{mapper_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the scope belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       requiresReplace,
			},
			"scope_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the client scope that owns the mapper. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       requiresReplace,
			},
			"mapper_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the protocol mapper.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Mapper name.",
				Required:            true,
			},
			"mapper_type": schema.StringAttribute{
				MarkdownDescription: "Mapper implementation type, e.g. `oidc-usermodel-attribute-mapper` or `oidc-audience-mapper`.",
				Required:            true,
			},
			"config": schema.MapAttribute{
				MarkdownDescription: "Mapper configuration as string key/value pairs, e.g. `claim.name`, " +
					"`user.attribute`, `id.token.claim`. Only the keys you specify are tracked; changing it " +
					"replaces the stored configuration.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *clientScopeProtocolMapperResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientScopeProtocolMapperResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientScopeProtocolMapperResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, d := scopeMapperConfig(ctx, plan.Config)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	scopeID := plan.ScopeID.ValueString()
	created, err := r.client.CreateProtocolMapper(ctx, realm, scopeID, client.CreateProtocolMapperRequest{
		Name:       plan.Name.ValueString(),
		MapperType: plan.MapperType.ValueString(),
		Config:     config,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating protocol mapper", err.Error())
		return
	}

	resp.Diagnostics.Append(scopeMapperFlatten(realm, scopeID, created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientScopeProtocolMapperResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientScopeProtocolMapperResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	scopeID := state.ScopeID.ValueString()
	mapper, err := r.client.GetProtocolMapper(ctx, realm, scopeID, state.MapperID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading protocol mapper", err.Error())
		return
	}

	resp.Diagnostics.Append(scopeMapperFlatten(realm, scopeID, mapper, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientScopeProtocolMapperResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state clientScopeProtocolMapperResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, d := scopeMapperConfig(ctx, plan.Config)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	scopeID := plan.ScopeID.ValueString()
	updated, err := r.client.UpdateProtocolMapper(ctx, realm, scopeID, state.MapperID.ValueString(), client.UpdateProtocolMapperRequest{
		Name:       strPtr(plan.Name),
		MapperType: strPtr(plan.MapperType),
		Config:     config,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating protocol mapper", err.Error())
		return
	}

	resp.Diagnostics.Append(scopeMapperFlatten(realm, scopeID, updated, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientScopeProtocolMapperResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientScopeProtocolMapperResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProtocolMapper(ctx, state.Realm.ValueString(), state.ScopeID.ValueString(), state.MapperID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting protocol mapper", err.Error())
	}
}

func (r *clientScopeProtocolMapperResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("invalid import ID %q: expected \"{realm}/{scope_id}/{mapper_id}\"", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mapper_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func scopeMapperConfig(ctx context.Context, m types.Map) (map[string]any, diag.Diagnostics) {
	values, diags := stringMap(ctx, m)
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out, diags
}

func scopeMapperFlatten(realm, scopeID string, mapper *client.ProtocolMapper, m *clientScopeProtocolMapperResourceModel) diag.Diagnostics {
	server := make(map[string]*string, len(mapper.Config))
	for k, v := range mapper.Config {
		if s, ok := v.(string); ok {
			server[k] = &s
			continue
		}
		encoded, err := json.Marshal(v)
		if err != nil {
			continue
		}
		s := string(encoded)
		server[k] = &s
	}

	config, diags := filterStringMap(m.Config, server)
	m.Config = config
	m.Realm = types.StringValue(realm)
	m.ScopeID = types.StringValue(scopeID)
	m.MapperID = types.StringValue(mapper.ID)
	m.ID = types.StringValue(realm + "/" + scopeID + "/" + mapper.ID)
	m.Name = types.StringValue(mapper.Name)
	m.MapperType = types.StringValue(mapper.MapperType)
	return diags
}
