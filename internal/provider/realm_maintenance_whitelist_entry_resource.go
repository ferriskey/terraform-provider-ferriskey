package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                     = &realmMaintenanceWhitelistEntryResource{}
	_ resource.ResourceWithConfigure        = &realmMaintenanceWhitelistEntryResource{}
	_ resource.ResourceWithImportState      = &realmMaintenanceWhitelistEntryResource{}
	_ resource.ResourceWithConfigValidators = &realmMaintenanceWhitelistEntryResource{}
)

// NewRealmMaintenanceWhitelistEntryResource is the resource factory.
func NewRealmMaintenanceWhitelistEntryResource() resource.Resource {
	return &realmMaintenanceWhitelistEntryResource{}
}

type realmMaintenanceWhitelistEntryResource struct {
	client *client.Client
}

type realmMaintenanceWhitelistEntryResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Realm   types.String `tfsdk:"realm"`
	EntryID types.String `tfsdk:"entry_id"`
	UserID  types.String `tfsdk:"user_id"`
	RoleID  types.String `tfsdk:"role_id"`
}

func (r *realmMaintenanceWhitelistEntryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_realm_maintenance_whitelist_entry"
}

func (r *realmMaintenanceWhitelistEntryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	rr := samlmUUIDRealmPlan()
	resp.Schema = schema.Schema{
		MarkdownDescription: "Allows a user or every holder of a role to keep logging in to a FerrisKey realm while its " +
			"maintenance mode is enabled. Exactly one of `user_id` or `role_id` must be set. Entries cannot be " +
			"updated, so every attribute change forces a new resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier in the form `{realm}/{entry_id}`.",
				Computed:            true,
				PlanModifiers:       samlmIDPlan(),
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       rr,
			},
			"entry_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the whitelist entry.",
				Computed:            true,
				PlanModifiers:       samlmIDPlan(),
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the whitelisted user. Conflicts with `role_id`. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers:       rr,
			},
			"role_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the whitelisted role. Conflicts with `user_id`. Changing this forces a new resource.",
				Optional:            true,
				PlanModifiers:       rr,
			},
		},
	}
}

func (r *realmMaintenanceWhitelistEntryResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("user_id"), path.MatchRoot("role_id")),
	}
}

func (r *realmMaintenanceWhitelistEntryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *realmMaintenanceWhitelistEntryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan realmMaintenanceWhitelistEntryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	e, err := r.client.AddRealmMaintenanceWhitelistEntry(ctx, plan.Realm.ValueString(), client.AddMaintenanceWhitelistEntryRequest{
		UserID: strPtr(plan.UserID),
		RoleID: strPtr(plan.RoleID),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error adding realm maintenance whitelist entry", err.Error())
		return
	}
	r.flatten(plan.Realm.ValueString(), e, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *realmMaintenanceWhitelistEntryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state realmMaintenanceWhitelistEntryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	entries, err := r.client.ListRealmMaintenanceWhitelist(ctx, state.Realm.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading realm maintenance whitelist", err.Error())
		return
	}
	for i := range entries {
		if entries[i].ID == state.EntryID.ValueString() {
			r.flatten(state.Realm.ValueString(), &entries[i], &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *realmMaintenanceWhitelistEntryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan realmMaintenanceWhitelistEntryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *realmMaintenanceWhitelistEntryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state realmMaintenanceWhitelistEntryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.RemoveRealmMaintenanceWhitelistEntry(ctx, state.Realm.ValueString(), state.EntryID.ValueString())
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Error removing realm maintenance whitelist entry", err.Error())
	}
}

func (r *realmMaintenanceWhitelistEntryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entry_id"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *realmMaintenanceWhitelistEntryResource) flatten(realm string, e *client.RealmMaintenanceWhitelistEntry, m *realmMaintenanceWhitelistEntryResourceModel) {
	m.Realm = types.StringValue(realm)
	m.EntryID = types.StringValue(e.ID)
	m.ID = types.StringValue(realm + "/" + e.ID)
	m.UserID = stringFromPtr(e.UserID)
	m.RoleID = stringFromPtr(e.RoleID)
}
