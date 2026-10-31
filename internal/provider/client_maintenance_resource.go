package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &clientMaintenanceResource{}
	_ resource.ResourceWithConfigure   = &clientMaintenanceResource{}
	_ resource.ResourceWithImportState = &clientMaintenanceResource{}
)

// NewClientMaintenanceResource is the resource factory.
func NewClientMaintenanceResource() resource.Resource {
	return &clientMaintenanceResource{}
}

type clientMaintenanceResource struct {
	client *client.Client
}

type clientMaintenanceResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Realm           types.String `tfsdk:"realm"`
	ClientUUID      types.String `tfsdk:"client_uuid"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	Reason          types.String `tfsdk:"reason"`
	SessionStrategy types.String `tfsdk:"session_strategy"`
}

func (r *clientMaintenanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_maintenance"
}

func (r *clientMaintenanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the maintenance mode of a FerrisKey client (one per client). While enabled, " +
			"logins to the client are refused except for whitelisted users and roles (see " +
			"`ferriskey_client_maintenance_whitelist_entry`). The state is written with a single toggle endpoint and " +
			"read back from the client. Destroying the resource disables maintenance mode on the client.",
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
				MarkdownDescription: "UUID of the client. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       samlmUUIDRealmPlan(),
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether maintenance mode is on. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"reason": schema.StringAttribute{
				MarkdownDescription: "Free-text reason for the maintenance, shown to users. Unset when omitted.",
				Optional:            true,
			},
			"session_strategy": schema.StringAttribute{
				MarkdownDescription: "What happens to existing sessions when maintenance starts: `expire` (default, " +
					"sessions end at their natural expiry) or `terminate` (sessions are ended immediately).",
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString("expire"),
				Validators: []validator.String{stringvalidator.OneOf("expire", "terminate")},
			},
		},
	}
}

func (r *clientMaintenanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientMaintenanceResource) apply(ctx context.Context, plan *clientMaintenanceResourceModel) error {
	realm, clientUUID := plan.Realm.ValueString(), plan.ClientUUID.ValueString()
	err := r.client.ToggleClientMaintenance(ctx, realm, clientUUID, client.ToggleClientMaintenanceRequest{
		Enabled:         plan.Enabled.ValueBool(),
		Reason:          strPtr(plan.Reason),
		SessionStrategy: strPtr(plan.SessionStrategy),
	})
	if err != nil {
		return err
	}
	st, err := r.client.GetClientMaintenance(ctx, realm, clientUUID)
	if err != nil {
		return err
	}
	r.flatten(realm, clientUUID, st, plan)
	return nil
}

func (r *clientMaintenanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientMaintenanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error setting client maintenance", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientMaintenanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientMaintenanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	realm, clientUUID := state.Realm.ValueString(), state.ClientUUID.ValueString()
	st, err := r.client.GetClientMaintenance(ctx, realm, clientUUID)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading client maintenance", err.Error())
		return
	}
	r.flatten(realm, clientUUID, st, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clientMaintenanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clientMaintenanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating client maintenance", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientMaintenanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientMaintenanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.ToggleClientMaintenance(ctx, state.Realm.ValueString(), state.ClientUUID.ValueString(), client.ToggleClientMaintenanceRequest{Enabled: false})
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Error disabling client maintenance", err.Error())
	}
}

func (r *clientMaintenanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := samlmSplitID(req.ID, "realm", "client_uuid")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_uuid"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *clientMaintenanceResource) flatten(realm, clientUUID string, st *client.ClientMaintenanceState, m *clientMaintenanceResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ClientUUID = types.StringValue(clientUUID)
	m.ID = types.StringValue(realm + "/" + clientUUID)
	m.Enabled = types.BoolValue(st.Enabled)
	m.Reason = stringFromPtr(st.Reason)
	m.SessionStrategy = types.StringValue(st.SessionStrategy)
}
