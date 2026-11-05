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
	_ resource.Resource                = &authzTemplateResource{}
	_ resource.ResourceWithConfigure   = &authzTemplateResource{}
	_ resource.ResourceWithImportState = &authzTemplateResource{}
)

func NewAuthzTemplateResource() resource.Resource {
	return &authzTemplateResource{}
}

type authzTemplateResource struct {
	client *client.Client
}

type authzTemplateResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Realm       types.String `tfsdk:"realm"`
	TemplateID  types.String `tfsdk:"template_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Source      types.String `tfsdk:"source"`
}

func (r *authzTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authz_template"
}

func (r *authzTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Cedar policy template of a FerrisKey realm. A template is Cedar text with " +
			"`?principal` and/or `?resource` slots that `ferriskey_authz_policy` resources instantiate through `links`. " +
			"A template cannot be deleted while a policy links it (the API answers 409), so linked policies must " +
			"reference the template to be destroyed first. The Cedar text is stored verbatim. Writes to a realm's " +
			"authorization rules are serialized by the provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{uuid}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the template belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"template_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the template.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Template name, unique within the realm. Letters, digits, `_`, `-`, `.` and `:` only (max 128 characters).",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Template description.",
				Optional:            true,
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Cedar template text containing `?principal` and/or `?resource` slots.",
				Required:            true,
			},
		},
	}
}

func (r *authzTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *authzTemplateResource) request(m *authzTemplateResourceModel) client.AuthzTemplateRequest {
	return client.AuthzTemplateRequest{
		Name:        m.Name.ValueString(),
		Description: strPtr(m.Description),
		Source:      m.Source.ValueString(),
	}
}

func (r *authzTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan authzTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	authzWriteMu.Lock()
	created, err := r.client.CreateAuthzTemplate(ctx, plan.Realm.ValueString(), r.request(&plan))
	authzWriteMu.Unlock()
	if err != nil {
		resp.Diagnostics.AddError("Error creating authorization template", err.Error())
		return
	}

	r.flatten(plan.Realm.ValueString(), created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *authzTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state authzTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	tpl, err := r.client.GetAuthzTemplate(ctx, realm, state.TemplateID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading authorization template", err.Error())
		return
	}

	r.flatten(realm, tpl, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *authzTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state authzTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	authzWriteMu.Lock()
	updated, err := r.client.UpdateAuthzTemplate(ctx, realm, state.TemplateID.ValueString(), r.request(&plan))
	authzWriteMu.Unlock()
	if err != nil {
		resp.Diagnostics.AddError("Error updating authorization template", err.Error())
		return
	}

	r.flatten(realm, updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *authzTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state authzTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	authzWriteMu.Lock()
	err := r.client.DeleteAuthzTemplate(ctx, state.Realm.ValueString(), state.TemplateID.ValueString())
	authzWriteMu.Unlock()
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting authorization template", err.Error())
	}
}

func (r *authzTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_id"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

func (r *authzTemplateResource) flatten(realm string, t *client.AuthzTemplate, m *authzTemplateResourceModel) {
	m.Realm = types.StringValue(realm)
	m.TemplateID = types.StringValue(t.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: t.ID}.String())
	m.Name = types.StringValue(t.Name)
	m.Description = stringFromPtr(t.Description)
	m.Source = types.StringValue(t.Source)
}
