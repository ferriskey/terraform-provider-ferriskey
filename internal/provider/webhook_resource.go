package provider

import (
	"context"
	"errors"

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
	_ resource.Resource                = &webhookResource{}
	_ resource.ResourceWithConfigure   = &webhookResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
	_ resource.ResourceWithModifyPlan  = &webhookResource{}
)

func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

type webhookResource struct {
	client *client.Client
}

type webhookRetryPolicyModel struct {
	MaxAttempts     types.Int64 `tfsdk:"max_attempts"`
	BaseDelayMs     types.Int64 `tfsdk:"base_delay_ms"`
	MaxDelayMs      types.Int64 `tfsdk:"max_delay_ms"`
	MaxTotalDelayMs types.Int64 `tfsdk:"max_total_delay_ms"`
}

type webhookResourceModel struct {
	ID                  types.String             `tfsdk:"id"`
	Realm               types.String             `tfsdk:"realm"`
	WebhookID           types.String             `tfsdk:"webhook_id"`
	Name                types.String             `tfsdk:"name"`
	Description         types.String             `tfsdk:"description"`
	Endpoint            types.String             `tfsdk:"endpoint"`
	Subscribers         types.Set                `tfsdk:"subscribers"`
	Headers             types.Map                `tfsdk:"headers"`
	RetryPolicy         *webhookRetryPolicyModel `tfsdk:"retry_policy"`
	Secret              types.String             `tfsdk:"secret"`
	RotateSecretTrigger types.String             `tfsdk:"rotate_secret_trigger"`
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a webhook subscription within a FerrisKey realm. " +
			"The signing secret is never returned by the API except when it is rotated, so `secret` is only populated " +
			"after setting or changing `rotate_secret_trigger`. Delivery history and retries are not managed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier `{realm}/{webhook_id}`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm the webhook belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"webhook_id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned UUID of the webhook.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable webhook name.",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Webhook description.",
				Optional:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "URL that receives the webhook deliveries.",
				Required:            true,
			},
			"subscribers": schema.SetAttribute{
				MarkdownDescription: "Set of event triggers the webhook subscribes to, e.g. `user.created`, `client.updated`, `realm.settings.updated`.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"headers": schema.MapAttribute{
				MarkdownDescription: "Extra HTTP headers sent with every delivery. The API never returns them, so the value is tracked from configuration only and drift cannot be detected. Removing the attribute clears the stored headers.",
				Optional:            true,
				Sensitive:           true,
				ElementType:         types.StringType,
			},
			"retry_policy": schema.SingleNestedAttribute{
				MarkdownDescription: "Per-webhook retry policy overriding the realm defaults. Unset fields fall back to the realm settings.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"max_attempts": schema.Int64Attribute{
						MarkdownDescription: "Maximum number of delivery attempts.",
						Optional:            true,
					},
					"base_delay_ms": schema.Int64Attribute{
						MarkdownDescription: "Initial retry delay in milliseconds.",
						Optional:            true,
					},
					"max_delay_ms": schema.Int64Attribute{
						MarkdownDescription: "Upper bound for a single retry delay in milliseconds.",
						Optional:            true,
					},
					"max_total_delay_ms": schema.Int64Attribute{
						MarkdownDescription: "Upper bound for the cumulative retry delay in milliseconds.",
						Optional:            true,
					},
				},
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Signing secret returned by the last rotation. Null until `rotate_secret_trigger` has been set.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"rotate_secret_trigger": schema.StringAttribute{
				MarkdownDescription: "Arbitrary value; setting it on creation or changing it afterwards rotates the signing secret in place (the webhook is not replaced) and refreshes `secret`.",
				Optional:            true,
			},
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *webhookResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan, state types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("rotate_secret_trigger"), &plan)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("rotate_secret_trigger"), &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.IsNull() && !plan.Equal(state) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("secret"), types.StringUnknown())...)
	}
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subscribers, d := stringSlice(ctx, plan.Subscribers)
	resp.Diagnostics.Append(d...)
	headers, d := stringMap(ctx, plan.Headers)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if subscribers == nil {
		subscribers = []string{}
	}

	realm := plan.Realm.ValueString()
	created, err := r.client.CreateWebhook(ctx, realm, client.CreateWebhookRequest{
		Endpoint:    plan.Endpoint.ValueString(),
		Name:        strPtr(plan.Name),
		Description: strPtr(plan.Description),
		Headers:     headers,
		Subscribers: subscribers,
		RetryPolicy: webhookRetryPolicyToAPI(plan.RetryPolicy),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook", err.Error())
		return
	}

	plan.Secret = types.StringNull()
	resp.Diagnostics.Append(r.flatten(realm, created, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.RotateSecretTrigger.IsNull() {
		secret, err := r.client.RotateWebhookSecret(ctx, realm, created.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error rotating webhook secret", err.Error())
			return
		}
		plan.Secret = types.StringValue(secret)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	realm := state.Realm.ValueString()
	webhook, err := r.client.GetWebhook(ctx, realm, state.WebhookID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading webhook", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(realm, webhook, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subscribers, d := stringSlice(ctx, plan.Subscribers)
	resp.Diagnostics.Append(d...)
	headers, d := stringMap(ctx, plan.Headers)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if subscribers == nil {
		subscribers = []string{}
	}
	if headers == nil && !state.Headers.IsNull() {
		headers = map[string]string{}
	}

	retry := webhookRetryPolicyToAPI(plan.RetryPolicy)
	if retry == nil {
		retry = &client.WebhookRetryPolicy{}
	}

	realm := plan.Realm.ValueString()
	id := state.WebhookID.ValueString()
	updated, err := r.client.UpdateWebhook(ctx, realm, id, client.UpdateWebhookRequest{
		Endpoint:    plan.Endpoint.ValueString(),
		Name:        strPtr(plan.Name),
		Description: strPtr(plan.Description),
		Headers:     headers,
		Subscribers: subscribers,
		RetryPolicy: retry,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook", err.Error())
		return
	}

	plan.Secret = state.Secret
	if !plan.RotateSecretTrigger.IsNull() && !plan.RotateSecretTrigger.Equal(state.RotateSecretTrigger) {
		secret, err := r.client.RotateWebhookSecret(ctx, realm, id)
		if err != nil {
			resp.Diagnostics.AddError("Error rotating webhook secret", err.Error())
			return
		}
		plan.Secret = types.StringValue(secret)
	}

	resp.Diagnostics.Append(r.flatten(realm, updated, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteWebhook(ctx, state.Realm.ValueString(), state.WebhookID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting webhook", err.Error())
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, err := parseRealmScopedID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parsed.Realm)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("webhook_id"), parsed.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parsed.String())...)
}

func (r *webhookResource) flatten(realm string, w *client.Webhook, m *webhookResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Realm = types.StringValue(realm)
	m.WebhookID = types.StringValue(w.ID)
	m.ID = types.StringValue(realmScopedID{Realm: realm, ID: w.ID}.String())
	m.Name = stringFromPtr(w.Name)
	m.Description = stringFromPtr(w.Description)
	m.Endpoint = types.StringValue(w.Endpoint)

	names := make([]string, 0, len(w.Subscribers))
	for _, s := range w.Subscribers {
		names = append(names, s.Name)
	}
	set, d := stringSetValue(names)
	diags.Append(d...)
	m.Subscribers = set

	retry := webhookRetryPolicyFromAPI(w.RetryPolicy)
	if retry == nil && m.RetryPolicy != nil {
		retry = &webhookRetryPolicyModel{
			MaxAttempts:     types.Int64Null(),
			BaseDelayMs:     types.Int64Null(),
			MaxDelayMs:      types.Int64Null(),
			MaxTotalDelayMs: types.Int64Null(),
		}
	}
	m.RetryPolicy = retry
	if m.Secret.IsUnknown() {
		m.Secret = types.StringNull()
	}
	return diags
}

func webhookRetryPolicyToAPI(p *webhookRetryPolicyModel) *client.WebhookRetryPolicy {
	if p == nil {
		return nil
	}
	return &client.WebhookRetryPolicy{
		MaxAttempts:     int64Ptr(p.MaxAttempts),
		BaseDelayMs:     int64Ptr(p.BaseDelayMs),
		MaxDelayMs:      int64Ptr(p.MaxDelayMs),
		MaxTotalDelayMs: int64Ptr(p.MaxTotalDelayMs),
	}
}

func webhookRetryPolicyFromAPI(p *client.WebhookRetryPolicy) *webhookRetryPolicyModel {
	if p == nil {
		return nil
	}
	if p.MaxAttempts == nil && p.BaseDelayMs == nil && p.MaxDelayMs == nil && p.MaxTotalDelayMs == nil {
		return nil
	}
	return &webhookRetryPolicyModel{
		MaxAttempts:     int64FromPtr(p.MaxAttempts),
		BaseDelayMs:     int64FromPtr(p.BaseDelayMs),
		MaxDelayMs:      int64FromPtr(p.MaxDelayMs),
		MaxTotalDelayMs: int64FromPtr(p.MaxTotalDelayMs),
	}
}
