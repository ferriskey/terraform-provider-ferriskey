package provider

import (
	"context"
	"errors"

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
	_ resource.Resource                = &passwordPolicyResource{}
	_ resource.ResourceWithConfigure   = &passwordPolicyResource{}
	_ resource.ResourceWithImportState = &passwordPolicyResource{}
)

// NewPasswordPolicyResource is the resource factory.
func NewPasswordPolicyResource() resource.Resource {
	return &passwordPolicyResource{}
}

type passwordPolicyResource struct {
	client *client.Client
}

type passwordPolicyResourceModel struct {
	Realm            types.String `tfsdk:"realm"`
	ID               types.String `tfsdk:"id"`
	MinLength        types.Int64  `tfsdk:"min_length"`
	RequireUppercase types.Bool   `tfsdk:"require_uppercase"`
	RequireLowercase types.Bool   `tfsdk:"require_lowercase"`
	RequireNumber    types.Bool   `tfsdk:"require_number"`
	RequireSpecial   types.Bool   `tfsdk:"require_special"`
	MaxAgeDays       types.Int64  `tfsdk:"max_age_days"`
}

func (r *passwordPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password_policy"
}

func (r *passwordPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	boolDefault := func(d string, def bool) schema.Attribute {
		return schema.BoolAttribute{MarkdownDescription: d, Optional: true, Computed: true, Default: booldefault.StaticBool(def)}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a FerrisKey realm's password policy. A realm always has a policy, so " +
			"deleting this resource stops managing it but does not remove it.",
		Attributes: map[string]schema.Attribute{
			"realm": schema.StringAttribute{
				MarkdownDescription: "Realm whose password policy is managed. Also the resource ID. Changing it forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (equal to `realm`).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"min_length": schema.Int64Attribute{
				MarkdownDescription: "Minimum password length.",
				Optional:            true,
				Computed:            true,
			},
			"require_uppercase": boolDefault("Require at least one uppercase character.", false),
			"require_lowercase": boolDefault("Require at least one lowercase character.", false),
			"require_number":    boolDefault("Require at least one digit.", false),
			"require_special":   boolDefault("Require at least one special character.", false),
			"max_age_days": schema.Int64Attribute{
				MarkdownDescription: "Maximum password age in days (null/unset means no expiry).",
				Optional:            true,
			},
		},
	}
}

func (r *passwordPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *passwordPolicyResource) apply(ctx context.Context, plan *passwordPolicyResourceModel) error {
	policy, err := r.client.UpdatePasswordPolicy(ctx, plan.Realm.ValueString(), client.UpdatePasswordPolicyRequest{
		MinLength:        int64Ptr(plan.MinLength),
		RequireUppercase: boolPtr(plan.RequireUppercase),
		RequireLowercase: boolPtr(plan.RequireLowercase),
		RequireNumber:    boolPtr(plan.RequireNumber),
		RequireSpecial:   boolPtr(plan.RequireSpecial),
		MaxAgeDays:       int64Ptr(plan.MaxAgeDays),
	})
	if err != nil {
		return err
	}
	r.flatten(plan.Realm.ValueString(), policy, plan)
	return nil
}

func (r *passwordPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan passwordPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error setting password policy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *passwordPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state passwordPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	policy, err := r.client.GetPasswordPolicy(ctx, state.Realm.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading password policy", err.Error())
		return
	}
	r.flatten(state.Realm.ValueString(), policy, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *passwordPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan passwordPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error updating password policy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op: the realm always has a password policy.
func (r *passwordPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *passwordPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *passwordPolicyResource) flatten(realm string, p *client.PasswordPolicy, m *passwordPolicyResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ID = types.StringValue(realm)
	m.MinLength = types.Int64Value(p.MinLength)
	m.RequireUppercase = types.BoolValue(p.RequireUppercase)
	m.RequireLowercase = types.BoolValue(p.RequireLowercase)
	m.RequireNumber = types.BoolValue(p.RequireNumber)
	m.RequireSpecial = types.BoolValue(p.RequireSpecial)
	m.MaxAgeDays = int64FromPtr(p.MaxAgeDays)
}
