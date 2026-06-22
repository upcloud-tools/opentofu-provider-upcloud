package tag

import (
	"context"
	"regexp"

	"github.com/UpCloudLtd/terraform-provider-upcloud/internal/utils"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/request"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &tagResource{}
	_ resource.ResourceWithConfigure   = &tagResource{}
	_ resource.ResourceWithImportState = &tagResource{}
)

func NewTagResource() resource.Resource {
	return &tagResource{}
}

type tagResource struct {
	client *service.Service
}

type tagModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Servers     types.Set    `tfsdk:"servers"`
}

func (r *tagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *tagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

func (r *tagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `~> Consider using labels instead of tags. Tags are an access control feature and only available for a limited set of resources. Use labels to describe and filter your resources.

This resource is deprecated, use tags schema in server resource`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the tag used as an identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The value representing the tag",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 32),
					stringvalidator.RegexMatches(regexp.MustCompile("[a-zA-Z0-9_]"), ""),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free form text representing the meaning of the tag",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 255),
				},
			},
			"servers": schema.SetAttribute{
				MarkdownDescription: "A collection of servers that have been assigned the tag",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *tagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data tagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &request.CreateTagRequest{
		Tag: upcloud.Tag{
			Name: data.Name.ValueString(),
		},
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		createReq.Description = data.Description.ValueString()
	}
	if !data.Servers.IsNull() && !data.Servers.IsUnknown() {
		var servers []string
		resp.Diagnostics.Append(data.Servers.ElementsAs(ctx, &servers, false)...)
		createReq.Servers = servers
	}

	tag, err := r.client.CreateTag(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create tag",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(tag.Name)
	data.Name = types.StringValue(tag.Name)
	data.Description = types.StringValue(tag.Description)

	servers, diags := types.SetValueFrom(ctx, types.StringType, tag.Servers)
	resp.Diagnostics.Append(diags...)
	data.Servers = servers

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, err := r.client.GetTags(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read tags",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	tagID := data.ID.ValueString()
	var found *upcloud.Tag
	for _, t := range tags.Tags {
		if t.Name == tagID {
			found = &t
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(found.Name)
	data.Description = types.StringValue(found.Description)

	servers, diags := types.SetValueFrom(ctx, types.StringType, found.Servers)
	resp.Diagnostics.Append(diags...)
	data.Servers = servers

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data tagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	modifyReq := &request.ModifyTagRequest{
		Name: data.ID.ValueString(),
	}
	modifyReq.Tag.Name = data.ID.ValueString()

	if !data.Description.IsUnknown() {
		modifyReq.Description = data.Description.ValueString()
	}
	if !data.Servers.IsNull() && !data.Servers.IsUnknown() {
		var servers []string
		resp.Diagnostics.Append(data.Servers.ElementsAs(ctx, &servers, false)...)
		modifyReq.Servers = servers
	}

	_, err := r.client.ModifyTag(ctx, modifyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to modify tag",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTag(ctx, &request.DeleteTagRequest{
		Name: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete tag",
			utils.ErrorDiagnosticDetail(err),
		)
	}
}

func (r *tagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
