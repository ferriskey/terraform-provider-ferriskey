package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &organizationGroupAttributeResource{}
	_ resource.ResourceWithConfigure   = &organizationGroupAttributeResource{}
	_ resource.ResourceWithImportState = &organizationGroupAttributeResource{}
)

// NewOrganizationGroupAttributeResource is the resource factory.
func NewOrganizationGroupAttributeResource() resource.Resource {
	return &organizationGroupAttributeResource{}
}

type organizationGroupAttributeResource struct {
	client *client.Client
}

type organizationGroupAttributeResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Realm          types.String `tfsdk:"realm"`
	OrganizationID types.String `tfsdk:"organization_id"`
	GroupID        types.String `tfsdk:"group_id"`
	Key            types.String `tfsdk:"key"`
	Value          types.String `tfsdk:"value"`
}

func (r *organizationGroupAttributeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_group_attribute"
}

func (r *organizationGroupAttributeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single key/value attribute on an organization group. Setting the same key again overwrites the value; destroying the resource deletes the attribute.",
		Attributes: map[string]schema.Attribute{
			"id":              orgExtIDAttribute("{realm}/{organization_id}/{group_id}/{key}"),
			"realm":           orgExtReplaceAttribute("Realm the organization belongs to."),
			"organization_id": orgExtReplaceAttribute("UUID of the organization."),
			"group_id":        orgExtReplaceAttribute("UUID of the organization group (`group_uuid` of `ferriskey_organization_group`)."),
			"key":             orgExtReplaceAttribute("Attribute key."),
			"value": schema.StringAttribute{
				MarkdownDescription: "Attribute value.",
				Required:            true,
			},
		},
	}
}

func (r *organizationGroupAttributeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *organizationGroupAttributeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationGroupAttributeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	realm, orgID, groupID, key := plan.Realm.ValueString(), plan.OrganizationID.ValueString(), plan.GroupID.ValueString(), plan.Key.ValueString()
	attr, err := r.client.UpsertOrganizationGroupAttribute(ctx, realm, orgID, groupID, key, plan.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error setting organization group attribute", err.Error())
		return
	}
	plan.ID = types.StringValue(realm + "/" + orgID + "/" + groupID + "/" + key)
	plan.Value = types.StringValue(attr.Value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationGroupAttributeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationGroupAttributeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attrs, err := r.client.ListOrganizationGroupAttributes(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.GroupID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization group attributes", err.Error())
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

func (r *organizationGroupAttributeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationGroupAttributeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attr, err := r.client.UpsertOrganizationGroupAttribute(ctx, plan.Realm.ValueString(), plan.OrganizationID.ValueString(), plan.GroupID.ValueString(), plan.Key.ValueString(), plan.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization group attribute", err.Error())
		return
	}
	plan.Value = types.StringValue(attr.Value)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationGroupAttributeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationGroupAttributeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteOrganizationGroupAttribute(ctx, state.Realm.ValueString(), state.OrganizationID.ValueString(), state.GroupID.ValueString(), state.Key.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("Error deleting organization group attribute", err.Error())
	}
}

func (r *organizationGroupAttributeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := parseCompositeID(req.ID, "realm", "organization_id", "group_id", "key")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[3])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
