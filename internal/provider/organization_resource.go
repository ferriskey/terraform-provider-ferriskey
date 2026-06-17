package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &organizationResource{}
	_ resource.ResourceWithConfigure   = &organizationResource{}
	_ resource.ResourceWithImportState = &organizationResource{}
)

// NewOrganizationResource is the resource factory.
func NewOrganizationResource() resource.Resource {
	return &organizationResource{}
}

type organizationResource struct {
	client *client.Client
}

type organizationResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Realm       types.String `tfsdk:"realm"`
	OrgUUID     types.String `tfsdk:"organization_uuid"`
	Name        types.String `tfsdk:"name"`
	Alias       types.String `tfsdk:"alias"`
	Description types.String `tfsdk:"description"`
	Domain      types.String `tfsdk:"domain"`
	RedirectURL types.String `tfsdk:"redirect_url"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

func (r *organizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *organizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an organization (multi-tenancy unit) within a FerrisKey realm.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{uuid}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the organization belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"organization_uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the organization.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable organization name.",
				Required:            true,
			},
			"alias": schema.StringAttribute{
				MarkdownDescription: "Stable, URL-safe identifier used for lookups and routing within the realm.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Organization description.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Associated email/identity domain.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"redirect_url": schema.StringAttribute{
				MarkdownDescription: "Redirect URL associated with the organization.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the organization is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *organizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *organizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	enabled := true
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		enabled = plan.Enabled.ValueBool()
	}

	created, err := r.client.CreateOrganization(ctx, realm, client.CreateOrganizationRequest{
		Name:        plan.Name.ValueString(),
		Alias:       plan.Alias.ValueString(),
		Description: strPtr(plan.Description),
		Domain:      strPtr(plan.Domain),
		RedirectURL: strPtr(plan.RedirectURL),
		Enabled:     enabled,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization", err.Error())
		return
	}

	r.flatten(realm, created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	org, err := r.client.GetOrganization(ctx, realm, state.OrgUUID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization", err.Error())
		return
	}

	r.flatten(realm, org, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state organizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	org, err := r.client.UpdateOrganization(ctx, realm, state.OrgUUID.ValueString(), client.UpdateOrganizationRequest{
		Name:        strPtr(plan.Name),
		Alias:       strPtr(plan.Alias),
		Description: strPtr(plan.Description),
		Domain:      strPtr(plan.Domain),
		RedirectURL: strPtr(plan.RedirectURL),
		Enabled:     boolPtr(plan.Enabled),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization", err.Error())
		return
	}

	r.flatten(realm, org, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteOrganization(ctx, state.Realm.ValueString(), state.OrgUUID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting organization", err.Error())
	}
}

func (r *organizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_uuid"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

func (r *organizationResource) flatten(realm string, org *client.Organization, m *organizationResourceModel) {
	m.Realm = types.StringValue(realm)
	m.OrgUUID = types.StringValue(org.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: org.ID}.String())
	m.Name = types.StringValue(org.Name)
	m.Alias = types.StringValue(org.Alias)
	m.Description = stringFromPtr(org.Description)
	m.Domain = stringFromPtr(org.Domain)
	m.RedirectURL = stringFromPtr(org.RedirectURL)
	m.Enabled = types.BoolValue(org.Enabled)
}
