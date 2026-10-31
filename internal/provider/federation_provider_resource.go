package provider

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

const rxMaskedSecret = "********"

var (
	_ resource.Resource                = &federationProviderResource{}
	_ resource.ResourceWithConfigure   = &federationProviderResource{}
	_ resource.ResourceWithImportState = &federationProviderResource{}
)

func NewFederationProviderResource() resource.Resource {
	return &federationProviderResource{}
}

type federationProviderResource struct {
	client *client.Client
}

type federationProviderResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Realm               types.String `tfsdk:"realm"`
	ProviderID          types.String `tfsdk:"provider_id"`
	Name                types.String `tfsdk:"name"`
	ProviderType        types.String `tfsdk:"provider_type"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	Priority            types.Int64  `tfsdk:"priority"`
	Config              types.String `tfsdk:"config"`
	BindPassword        types.String `tfsdk:"bind_password"`
	SyncEnabled         types.Bool   `tfsdk:"sync_enabled"`
	SyncMode            types.String `tfsdk:"sync_mode"`
	SyncIntervalMinutes types.Int64  `tfsdk:"sync_interval_minutes"`
}

func (r *federationProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_provider"
}

func (r *federationProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a user federation provider (LDAP, Active Directory, Kerberos or custom) within a FerrisKey realm. " +
			"User synchronization and connection tests are not managed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{uuid}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the provider belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"provider_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the provider.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Provider name.",
				Required:            true,
			},
			"provider_type": schema.StringAttribute{
				MarkdownDescription: "Provider type: `Ldap`, `ActiveDirectory`, `Kerberos`, or any custom type string.",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the provider is active. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "Lookup priority among providers. Defaults to `0`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"config": schema.StringAttribute{
				MarkdownDescription: "Provider configuration as a JSON document (use `jsonencode`). For LDAP this has the objects `connection`, `bind` (`bind_dn`), `search` and `attributes`. " +
					"Do not put the bind password here: the API masks secrets, so use `bind_password` instead.",
				Required: true,
			},
			"bind_password": schema.StringAttribute{
				MarkdownDescription: "Password used to bind to the directory. It is injected into `config` as `bind.bind_password_encrypted`. The API never returns it, so it is tracked from configuration only.",
				Optional:            true,
				Sensitive:           true,
			},
			"sync_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether periodic user synchronization is enabled. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sync_mode": schema.StringAttribute{
				MarkdownDescription: "Synchronization mode: `Import`, `Force` or `LinkOnly`. Defaults to `LinkOnly`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("LinkOnly"),
			},
			"sync_interval_minutes": schema.Int64Attribute{
				MarkdownDescription: "Interval between synchronizations in minutes.",
				Optional:            true,
			},
		},
	}
}

func (r *federationProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *federationProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan federationProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := rxBuildProviderConfig(plan.Config.ValueString(), plan.BindPassword)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("config"), "Invalid config", err.Error())
		return
	}

	realm := plan.Realm.ValueString()
	created, err := r.client.CreateFederationProvider(ctx, realm, client.CreateFederationProviderRequest{
		Name:                plan.Name.ValueString(),
		ProviderType:        plan.ProviderType.ValueString(),
		Enabled:             plan.Enabled.ValueBool(),
		Priority:            plan.Priority.ValueInt64(),
		Config:              cfg,
		SyncEnabled:         plan.SyncEnabled.ValueBool(),
		SyncMode:            plan.SyncMode.ValueString(),
		SyncIntervalMinutes: int64Ptr(plan.SyncIntervalMinutes),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating federation provider", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *federationProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state federationProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	p, err := r.client.GetFederationProvider(ctx, realm, state.ProviderID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading federation provider", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, p, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *federationProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state federationProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := rxBuildProviderConfig(plan.Config.ValueString(), plan.BindPassword)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("config"), "Invalid config", err.Error())
		return
	}

	realm := plan.Realm.ValueString()
	updated, err := r.client.UpdateFederationProvider(ctx, realm, state.ProviderID.ValueString(), client.UpdateFederationProviderRequest{
		Name:                plan.Name.ValueString(),
		ProviderType:        plan.ProviderType.ValueString(),
		Enabled:             plan.Enabled.ValueBool(),
		Priority:            plan.Priority.ValueInt64(),
		Config:              cfg,
		SyncEnabled:         plan.SyncEnabled.ValueBool(),
		SyncMode:            plan.SyncMode.ValueString(),
		SyncIntervalMinutes: int64Ptr(plan.SyncIntervalMinutes),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating federation provider", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, updated, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *federationProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state federationProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteFederationProvider(ctx, state.Realm.ValueString(), state.ProviderID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting federation provider", err.Error())
	}
}

func (r *federationProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("provider_id"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

func (r *federationProviderResource) flatten(realm string, p *client.FederationProvider, m *federationProviderResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Realm = types.StringValue(realm)
	m.ProviderID = types.StringValue(p.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: p.ID}.String())
	m.Name = types.StringValue(p.Name)
	m.ProviderType = types.StringValue(p.ProviderType)
	m.Enabled = types.BoolValue(p.Enabled)
	m.Priority = types.Int64Value(p.Priority)
	m.SyncEnabled = types.BoolValue(p.SyncEnabled)
	m.SyncMode = types.StringValue(p.SyncMode)
	m.SyncIntervalMinutes = int64FromPtr(p.SyncIntervalMinutes)

	var server any
	if err := json.Unmarshal(p.Config, &server); err != nil {
		diags.AddError("Invalid provider config returned by the API", err.Error())
		return diags
	}
	pruned, _ := json.Marshal(rxPruneMasked(server))
	if m.Config.IsNull() || m.Config.IsUnknown() || !rxJSONEqual(m.Config.ValueString(), pruned) {
		m.Config = types.StringValue(string(pruned))
	}
	return diags
}

func rxBuildProviderConfig(config string, bindPassword types.String) (json.RawMessage, error) {
	if bindPassword.IsNull() || bindPassword.IsUnknown() {
		if !json.Valid([]byte(config)) {
			return nil, errors.New("config must be a valid JSON document")
		}
		return json.RawMessage(config), nil
	}

	var cfg map[string]any
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return nil, errors.New("config must be a JSON object when bind_password is set")
	}
	bind, ok := cfg["bind"].(map[string]any)
	if !ok {
		bind = map[string]any{}
		cfg["bind"] = bind
	}
	bind["bind_password_encrypted"] = bindPassword.ValueString()
	out, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func rxPruneMasked(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if s, ok := val.(string); ok && s == rxMaskedSecret {
				continue
			}
			out[k] = rxPruneMasked(val)
		}
		return out
	default:
		return v
	}
}
