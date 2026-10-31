package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ferriskey/terraform-provider-ferriskey/internal/client"
)

var (
	_ resource.Resource                = &clientSamlAttributeMapperResource{}
	_ resource.ResourceWithConfigure   = &clientSamlAttributeMapperResource{}
	_ resource.ResourceWithImportState = &clientSamlAttributeMapperResource{}
)

const samlmNameFormatBasic = "urn:oasis:names:tc:SAML:2.0:attrname-format:basic"

// NewClientSamlAttributeMapperResource is the resource factory.
func NewClientSamlAttributeMapperResource() resource.Resource {
	return &clientSamlAttributeMapperResource{}
}

type clientSamlAttributeMapperResource struct {
	client *client.Client
}

type clientSamlAttributeMapperResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Realm      types.String `tfsdk:"realm"`
	ClientUUID types.String `tfsdk:"client_uuid"`
	MapperID   types.String `tfsdk:"mapper_id"`
	Name       types.String `tfsdk:"name"`
	Source     types.String `tfsdk:"source"`
	NameFormat types.String `tfsdk:"name_format"`
}

func (r *clientSamlAttributeMapperResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_saml_attribute_mapper"
}

func (r *clientSamlAttributeMapperResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	rr := samlmUUIDRealmPlan()
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SAML attribute mapper of a FerrisKey client, i.e. one user attribute released " +
			"in the SAML assertion. The API has no update endpoint, so every attribute change forces a new resource. " +
			"Attribute names must be unique per client.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier in the form `{realm}/{client_uuid}/{mapper_id}`.",
				Computed:            true,
				PlanModifiers:       samlmIDPlan(),
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "Name of the realm owning the client. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       rr,
			},
			"client_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the SAML client. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       rr,
			},
			"mapper_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the attribute mapper.",
				Computed:            true,
				PlanModifiers:       samlmIDPlan(),
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the attribute in the SAML assertion. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       rr,
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Where the value comes from: `user:id`, `user:username`, `user:email`, " +
					"`user:first_name`, `user:last_name` or `attribute:<key>` for a custom user attribute. " +
					"Changing this forces a new resource.",
				Required:      true,
				PlanModifiers: rr,
			},
			"name_format": schema.StringAttribute{
				MarkdownDescription: "SAML attribute name format URN. One of `urn:oasis:names:tc:SAML:2.0:attrname-format:basic` " +
					"(default), `urn:oasis:names:tc:SAML:2.0:attrname-format:uri` or " +
					"`urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified`. Changing this forces a new resource.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(samlmNameFormatBasic),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{stringvalidator.OneOf(
					samlmNameFormatBasic,
					"urn:oasis:names:tc:SAML:2.0:attrname-format:uri",
					"urn:oasis:names:tc:SAML:2.0:attrname-format:unspecified",
				)},
			},
		},
	}
}

func (r *clientSamlAttributeMapperResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, errMsg := providerClient(req.ProviderData)
	if errMsg != "" {
		resp.Diagnostics.AddError("Unexpected provider data type", errMsg)
		return
	}
	r.client = c
}

func (r *clientSamlAttributeMapperResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientSamlAttributeMapperResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := r.client.CreateSamlAttributeMapper(ctx, plan.Realm.ValueString(), plan.ClientUUID.ValueString(), client.CreateSamlAttributeMapperRequest{
		Name:       plan.Name.ValueString(),
		NameFormat: plan.NameFormat.ValueString(),
		Source:     plan.Source.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating SAML attribute mapper", err.Error())
		return
	}
	r.flatten(plan.Realm.ValueString(), plan.ClientUUID.ValueString(), m, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientSamlAttributeMapperResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientSamlAttributeMapperResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mappers, err := r.client.ListSamlAttributeMappers(ctx, state.Realm.ValueString(), state.ClientUUID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading SAML attribute mappers", err.Error())
		return
	}
	for i := range mappers {
		if mappers[i].ID == state.MapperID.ValueString() {
			r.flatten(state.Realm.ValueString(), state.ClientUUID.ValueString(), &mappers[i], &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *clientSamlAttributeMapperResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clientSamlAttributeMapperResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clientSamlAttributeMapperResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientSamlAttributeMapperResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteSamlAttributeMapper(ctx, state.Realm.ValueString(), state.ClientUUID.ValueString(), state.MapperID.ValueString())
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting SAML attribute mapper", err.Error())
	}
}

func (r *clientSamlAttributeMapperResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := samlmSplitID(req.ID, "realm", "client_uuid", "mapper_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("realm"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_uuid"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mapper_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *clientSamlAttributeMapperResource) flatten(realm, clientUUID string, mp *client.SamlAttributeMapper, m *clientSamlAttributeMapperResourceModel) {
	m.Realm = types.StringValue(realm)
	m.ClientUUID = types.StringValue(clientUUID)
	m.MapperID = types.StringValue(mp.ID)
	m.ID = types.StringValue(realm + "/" + clientUUID + "/" + mp.ID)
	m.Name = types.StringValue(mp.Name)
	m.Source = types.StringValue(mp.Source)
	m.NameFormat = types.StringValue(mp.NameFormat)
}
