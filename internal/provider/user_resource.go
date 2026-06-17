package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

// NewUserResource is the resource factory.
func NewUserResource() resource.Resource {
	return &userResource{}
}

type userResource struct {
	client *client.Client
}

type userResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Realm           types.String `tfsdk:"realm"`
	UserUUID        types.String `tfsdk:"user_uuid"`
	Username        types.String `tfsdk:"username"`
	Email           types.String `tfsdk:"email"`
	EmailVerified   types.Bool   `tfsdk:"email_verified"`
	Firstname       types.String `tfsdk:"firstname"`
	Lastname        types.String `tfsdk:"lastname"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	RequiredActions types.Set    `tfsdk:"required_actions"`
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a user within a FerrisKey realm.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{uuid}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the user belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"user_uuid": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the user.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address. Once set it cannot be cleared by removing it from the " +
					"configuration (the API treats an omitted field as unchanged); set a new value to change it.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"email_verified": schema.BoolAttribute{
				MarkdownDescription: "Whether the email address is verified. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"firstname": schema.StringAttribute{
				MarkdownDescription: "First name.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"lastname": schema.StringAttribute{
				MarkdownDescription: "Last name.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the user is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"required_actions": schema.SetAttribute{
				MarkdownDescription: "Set of required actions the user must complete at next login " +
					"(`configure_otp`, `verify_email`, `update_password`, `configure_passkey`).",
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	created, err := r.client.CreateUser(ctx, realm, client.CreateUserRequest{
		Username:      plan.Username.ValueString(),
		Email:         strPtr(plan.Email),
		EmailVerified: boolPtr(plan.EmailVerified),
		Firstname:     strPtr(plan.Firstname),
		Lastname:      strPtr(plan.Lastname),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	// `enabled` and `required_actions` are not accepted by the create endpoint,
	// so apply them with a follow-up update. The update is a full replacement,
	// so we must resend every field (otherwise unspecified fields are nulled).
	requiredActions, d := stringSlice(ctx, plan.RequiredActions)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if boolPtr(plan.Enabled) != nil || requiredActions != nil {
		updated, err := r.client.UpdateUser(ctx, realm, created.ID, client.UpdateUserRequest{
			Email:           strPtr(plan.Email),
			EmailVerified:   boolPtr(plan.EmailVerified),
			Firstname:       strPtr(plan.Firstname),
			Lastname:        strPtr(plan.Lastname),
			Enabled:         boolPtr(plan.Enabled),
			RequiredActions: requiredActions,
		})
		if err != nil {
			resp.Diagnostics.AddError("Error finalizing user creation", err.Error())
			return
		}
		created = updated
	}

	resp.Diagnostics.Append(r.flatten(realm, created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	u, err := r.client.GetUser(ctx, realm, state.UserUUID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, u, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := plan.Realm.ValueString()
	requiredActions, d := stringSlice(ctx, plan.RequiredActions)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateUser(ctx, realm, state.UserUUID.ValueString(), client.UpdateUserRequest{
		Email:           strPtr(plan.Email),
		EmailVerified:   boolPtr(plan.EmailVerified),
		Firstname:       strPtr(plan.Firstname),
		Lastname:        strPtr(plan.Lastname),
		Enabled:         boolPtr(plan.Enabled),
		RequiredActions: requiredActions,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, updated, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteUser(ctx, state.Realm.ValueString(), state.UserUUID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_uuid"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

func (r *userResource) flatten(realm string, u *client.User, m *userResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Realm = types.StringValue(realm)
	m.UserUUID = types.StringValue(u.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: u.ID}.String())
	m.Username = types.StringValue(u.Username)
	m.Email = stringFromPtr(u.Email)
	m.EmailVerified = types.BoolValue(u.EmailVerified)
	m.Firstname = stringFromPtr(u.Firstname)
	m.Lastname = stringFromPtr(u.Lastname)
	m.Enabled = types.BoolValue(u.Enabled)

	set, d := stringSetValue(u.RequiredActions)
	diags.Append(d...)
	m.RequiredActions = set
	return diags
}
