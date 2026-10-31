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
	_ resource.Resource                = &organizationAttributeResource{}
	_ resource.ResourceWithConfigure   = &organizationAttributeResource{}
	_ resource.ResourceWithImportState = &organizationAttributeResource{}
)

// NewOrganizationAttributeResource is the resource factory.
func NewOrganizationAttributeResource() resource.Resource {
	return &organizationAttributeResource{}
}

type organizationAttributeResource struct {
	client *client.Client
}

type organizationAttributeResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Realm          types.String `tfsdk:"realm"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Key            types.String `tfsdk:"key"`
	Value          types.String `tfsdk:"value"`
}

func (r *organizationAttributeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_attribute"
}

func orgExtIDAttribute(format string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "Composite identifier `" + format + "`.",
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func orgExtReplaceAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: description + " Changing this forces a new resource.",
		Required:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
}

func (r *organizationAttributeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single key/value attribute on an organization. Setting the same key again overwrites the value; destroying the resource deletes the attribute.",
		Attributes: map[string]schema.Attribute{
			"id":              orgExtIDAttribute("{realm}/{organization_id}/{key}"),
			"realm":           orgExtReplaceAttribute("Realm the organization belongs to."),
			"organization_id": orgExtReplaceAttribute("UUID of the organization."),
			"key":             orgExtReplaceAttribute("Attribute key."),
			"value": schema.StringAttribute{
				MarkdownDescription: "Attribute value.",
				Required:            true,
			},
		},
	}
}

func (r *organizationAttributeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *organizationAttributeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationAttributeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attr, err := r.client.UpsertOrganizationAttribute(ctx, plan.Realm.ValueString(), plan.OrganizationID.ValueString(), plan.Key.ValueString(), plan.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error setting organization attribute", err.Error())
		return
	}
	plan.ID = types.StringValue(plan.Realm.ValueString() + "/" + plan.OrganizationID.ValueString() + "/" + plan.Key.ValueString())
	plan.Value = types.StringValue(attr.Value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationAttributeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationAttributeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attrs, err := r.client.ListOrganizationAttributes(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization attributes", err.Error())
		return
	}
	for _, a := range attrs {
		if a.Key == state.Key.ValueString() {
			state.Value = types.StringValue(a.Value)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *organizationAttributeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationAttributeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attr, err := r.client.UpsertOrganizationAttribute(ctx, plan.Realm.ValueString(), plan.OrganizationID.ValueString(), plan.Key.ValueString(), plan.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization attribute", err.Error())
		return
	}
	plan.Value = types.StringValue(attr.Value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationAttributeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationAttributeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteOrganizationAttribute(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.Key.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting organization attribute", err.Error())
	}
}

func (r *organizationAttributeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, "realm", "organization_id", "key")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
