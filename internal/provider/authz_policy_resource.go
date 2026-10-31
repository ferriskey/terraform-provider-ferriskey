package provider

import (
	"context"
	"errors"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                     = &authzPolicyResource{}
	_ resource.ResourceWithConfigure        = &authzPolicyResource{}
	_ resource.ResourceWithImportState      = &authzPolicyResource{}
	_ resource.ResourceWithConfigValidators = &authzPolicyResource{}
)

var authzWriteMu sync.Mutex

var authzLinkObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"type": types.StringType,
	"id":   types.StringType,
}}

func NewAuthzPolicyResource() resource.Resource {
	return &authzPolicyResource{}
}

type authzPolicyResource struct {
	client *client.Client
}

type authzLinkModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
}

type authzPolicyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Realm       types.String `tfsdk:"realm"`
	PolicyID    types.String `tfsdk:"policy_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	ManagedBy   types.String `tfsdk:"managed_by"`
	Origin      types.String `tfsdk:"origin"`
	Source      types.String `tfsdk:"source"`
	TemplateID  types.String `tfsdk:"template_id"`
	Links       types.Map    `tfsdk:"links"`
}

func (r *authzPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authz_policy"
}

func (r *authzPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Cedar authorization policy of a FerrisKey realm. A policy is either " +
			"**static** (`source` holds Cedar text) or **linked** (`template_id` plus `links` instantiate a " +
			"`ferriskey_authz_template`). The server validates every enabled policy against the realm schema on " +
			"each write, so a policy that references types or actions declared in a `ferriskey_authz_schema` " +
			"must depend on it (reference it or use `depends_on`). The Cedar text is stored and returned verbatim, " +
			"so there is no normalization drift. Writes to a realm's authorization rules are serialized by the provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{uuid}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the policy belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"policy_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the policy.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Policy name, unique within the realm. Letters, digits, `_`, `-`, `.` and `:` only (max 128 characters).",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Policy description.",
				Optional:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the policy takes part in evaluation. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"managed_by": schema.StringAttribute{
				MarkdownDescription: "Optional manager label (lowercase letters, digits, `_`, `-`, `.`; max 64 characters) " +
					"stamped on the policy so several tools can own disjoint policy sets. Changing it forces a new resource.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"origin": schema.StringAttribute{
				MarkdownDescription: "How the policy was authored, as reported by the API (`custom` for static, `generated` for linked, `builtin`).",
				Computed:            true,
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Cedar policy text of a static policy. Exactly one of `source` or `template_id` must be set.",
				Optional:            true,
			},
			"template_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the `ferriskey_authz_template` a linked policy instantiates. Requires `links`.",
				Optional:            true,
			},
			"links": schema.MapNestedAttribute{
				MarkdownDescription: "Template slot assignments of a linked policy, keyed by slot (`?principal` or `?resource`).",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							MarkdownDescription: "Cedar entity type, e.g. `FerrisKey::Role`.",
							Required:            true,
						},
						"id": schema.StringAttribute{
							MarkdownDescription: "Entity identifier.",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (r *authzPolicyResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("source"), path.MatchRoot("template_id")),
		resourcevalidator.RequiredTogether(path.MatchRoot("template_id"), path.MatchRoot("links")),
		resourcevalidator.Conflicting(path.MatchRoot("source"), path.MatchRoot("links")),
	}
}

func (r *authzPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *authzPolicyResource) expand(ctx context.Context, m *authzPolicyResourceModel) (client.AuthzPolicyRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := client.AuthzPolicyRequest{
		Name:        m.Name.ValueString(),
		Description: strPtr(m.Description),
		Enabled:     m.Enabled.ValueBool(),
	}
	if !m.TemplateID.IsNull() {
		links := map[string]authzLinkModel{}
		diags.Append(m.Links.ElementsAs(ctx, &links, false)...)
		body := client.AuthzPolicyBody{Type: "linked", TemplateID: strPtr(m.TemplateID), Links: map[string]client.AuthzEntityRef{}}
		for slot, l := range links {
			body.Links[slot] = client.AuthzEntityRef{Type: l.Type.ValueString(), ID: l.ID.ValueString()}
		}
		req.Body = body
		return req, diags
	}
	req.Body = client.AuthzPolicyBody{Type: "static", Source: strPtr(m.Source)}
	return req, diags
}

func (r *authzPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan authzPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, d := r.expand(ctx, &plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	authzWriteMu.Lock()
	created, err := r.client.CreateAuthzPolicy(ctx, plan.Realm.ValueString(), plan.ManagedBy.ValueString(), body)
	authzWriteMu.Unlock()
	if err != nil {
		resp.Diagnostics.AddError("Error creating authorization policy", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, plan.Realm.ValueString(), created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *authzPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state authzPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	policy, err := r.client.GetAuthzPolicy(ctx, realm, state.PolicyID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading authorization policy", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, realm, policy, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *authzPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state authzPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, d := r.expand(ctx, &plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	authzWriteMu.Lock()
	updated, err := r.client.UpdateAuthzPolicy(ctx, realm, state.PolicyID.ValueString(), plan.ManagedBy.ValueString(), body)
	authzWriteMu.Unlock()
	if err != nil {
		resp.Diagnostics.AddError("Error updating authorization policy", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, realm, updated, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *authzPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state authzPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	authzWriteMu.Lock()
	err := r.client.DeleteAuthzPolicy(ctx, state.Realm.ValueString(), state.PolicyID.ValueString())
	authzWriteMu.Unlock()
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting authorization policy", err.Error())
	}
}

func (r *authzPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_id"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

func (r *authzPolicyResource) flatten(ctx context.Context, realm string, p *client.AuthzPolicy, m *authzPolicyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Realm = types.StringValue(realm)
	m.PolicyID = types.StringValue(p.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: p.ID}.String())
	m.Name = types.StringValue(p.Name)
	m.Description = stringFromPtr(p.Description)
	m.Enabled = types.BoolValue(p.Enabled)
	m.ManagedBy = stringFromPtr(p.ManagedBy)
	m.Origin = types.StringValue(p.Origin)
	m.Source = stringFromPtr(p.Body.Source)
	m.TemplateID = stringFromPtr(p.Body.TemplateID)

	if p.Body.Type != "linked" {
		m.Links = types.MapNull(authzLinkObjectType)
		return diags
	}
	links := make(map[string]authzLinkModel, len(p.Body.Links))
	for slot, l := range p.Body.Links {
		links[slot] = authzLinkModel{Type: types.StringValue(l.Type), ID: types.StringValue(l.ID)}
	}
	m.Links, diags = types.MapValueFrom(ctx, authzLinkObjectType, links)
	return diags
}
