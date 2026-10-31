package provider

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

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
	_ resource.Resource                = &emailTemplateResource{}
	_ resource.ResourceWithConfigure   = &emailTemplateResource{}
	_ resource.ResourceWithImportState = &emailTemplateResource{}
)

func NewEmailTemplateResource() resource.Resource {
	return &emailTemplateResource{}
}

type emailTemplateResource struct {
	client *client.Client
}

type emailTemplateResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Realm      types.String `tfsdk:"realm"`
	TemplateID types.String `tfsdk:"template_id"`
	Name       types.String `tfsdk:"name"`
	EmailType  types.String `tfsdk:"email_type"`
	Structure  types.String `tfsdk:"structure"`
	MJML       types.String `tfsdk:"mjml"`
}

func (r *emailTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_template"
}

func (r *emailTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an email template within a FerrisKey realm. Template import/export is not managed.",
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
				MarkdownDescription: "Template name.",
				Required:            true,
			},
			"email_type": schema.StringAttribute{
				MarkdownDescription: "Email the template applies to: `reset_password`, `magic_link` or `email_verification`. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"structure": schema.StringAttribute{
				MarkdownDescription: "Template body as a JSON document (use `jsonencode`), shaped like `{\"children\": [{\"type\": \"mj-section\", ...}]}`. Node types must be `mj-*` elements. The server sanitizes node content, so sanitized values show up as drift.",
				Required:            true,
			},
			"mjml": schema.StringAttribute{
				MarkdownDescription: "MJML markup rendered by the server from `structure`.",
				Computed:            true,
			},
		},
	}
}

func (r *emailTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *emailTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan emailTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !json.Valid([]byte(plan.Structure.ValueString())) {
		resp.Diagnostics.AddAttributeError(path.Root("structure"), "Invalid JSON", "structure must be a valid JSON document.")
		return
	}

	realm := plan.Realm.ValueString()
	created, err := r.client.CreateEmailTemplate(ctx, realm, client.CreateEmailTemplateRequest{
		Name:      plan.Name.ValueString(),
		EmailType: plan.EmailType.ValueString(),
		Structure: json.RawMessage(plan.Structure.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating email template", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *emailTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state emailTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	tpl, err := r.client.GetEmailTemplate(ctx, realm, state.TemplateID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading email template", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, tpl, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *emailTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state emailTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !json.Valid([]byte(plan.Structure.ValueString())) {
		resp.Diagnostics.AddAttributeError(path.Root("structure"), "Invalid JSON", "structure must be a valid JSON document.")
		return
	}

	realm := plan.Realm.ValueString()
	updated, err := r.client.UpdateEmailTemplate(ctx, realm, state.TemplateID.ValueString(), client.UpdateEmailTemplateRequest{
		Name:      plan.Name.ValueString(),
		Structure: json.RawMessage(plan.Structure.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating email template", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, updated, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *emailTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state emailTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteEmailTemplate(ctx, state.Realm.ValueString(), state.TemplateID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting email template", err.Error())
	}
}

func (r *emailTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_id"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

func (r *emailTemplateResource) flatten(realm string, t *client.EmailTemplate, m *emailTemplateResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Realm = types.StringValue(realm)
	m.TemplateID = types.StringValue(t.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: t.ID}.String())
	m.Name = types.StringValue(t.Name)
	m.EmailType = types.StringValue(t.EmailType)
	m.MJML = types.StringValue(t.MJML)

	if m.Structure.IsNull() || m.Structure.IsUnknown() || !rxJSONEqual(m.Structure.ValueString(), t.Structure) {
		var server any
		if err := json.Unmarshal(t.Structure, &server); err != nil {
			diags.AddError("Invalid template structure returned by the API", err.Error())
			return diags
		}
		canonical, _ := json.Marshal(server)
		m.Structure = types.StringValue(string(canonical))
	}
	return diags
}

func rxJSONEqual(a string, b []byte) bool {
	var av, bv any
	if json.Unmarshal([]byte(a), &av) != nil || json.Unmarshal(b, &bv) != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}
