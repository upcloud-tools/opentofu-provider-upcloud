package tag

import (
	"context"
	"time"

	"github.com/UpCloudLtd/terraform-provider-upcloud/internal/utils"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &tagsDataSource{}
	_ datasource.DataSourceWithConfigure = &tagsDataSource{}
)

func NewTagsDataSource() datasource.DataSource {
	return &tagsDataSource{}
}

type tagsDataSource struct {
	client *service.Service
}

type tagsModel struct {
	ID   types.String `tfsdk:"id"`
	Tags types.Set    `tfsdk:"tags"`
}

type tagDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Servers     types.Set    `tfsdk:"servers"`
}

func (d *tagsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tags"
}

func (d *tagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

func (d *tagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `~> Consider using labels instead of tags. Tags are an access control feature and only available for a limited set of resources. Use labels to describe and filter your resources.

Data-source is deprecated.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
		Blocks: map[string]schema.Block{
			"tags": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The value representing the tag",
						},
						"description": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Free form text representing the meaning of the tag",
						},
						"servers": schema.SetAttribute{
							Computed:            true,
							MarkdownDescription: "A collection of servers that have been assigned the tag",
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *tagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, err := d.client.GetTags(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read tags",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(time.Now().UTC().String())

	tagModels := make([]tagDataSourceModel, len(tags.Tags))
	for i, t := range tags.Tags {
		tagModels[i].Name = types.StringValue(t.Name)
		tagModels[i].Description = types.StringValue(t.Description)

		servers, diags := types.SetValueFrom(ctx, types.StringType, t.Servers)
		resp.Diagnostics.Append(diags...)
		tagModels[i].Servers = servers
	}

	var diags diag.Diagnostics
	data.Tags, diags = types.SetValueFrom(ctx, data.Tags.ElementType(ctx), tagModels)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
