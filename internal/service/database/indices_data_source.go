package database

import (
	"context"
	"time"

	"github.com/UpCloudLtd/terraform-provider-upcloud/internal/utils"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/request"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &opensearchIndicesDataSource{}
	_ datasource.DataSourceWithConfigure = &opensearchIndicesDataSource{}
)

func NewOpenSearchIndicesDataSource() datasource.DataSource {
	return &opensearchIndicesDataSource{}
}

type opensearchIndicesDataSource struct {
	client *service.Service
}

type opensearchIndicesModel struct {
	ID      types.String `tfsdk:"id"`
	Service types.String `tfsdk:"service"`
	Indices types.Set    `tfsdk:"indices"`
}

type opensearchIndexModel struct {
	CreateTime          types.String `tfsdk:"create_time"`
	Docs                types.Int64  `tfsdk:"docs"`
	Health              types.String `tfsdk:"health"`
	IndexName           types.String `tfsdk:"index_name"`
	NumberOfReplicas    types.Int64  `tfsdk:"number_of_replicas"`
	NumberOfShards      types.Int64  `tfsdk:"number_of_shards"`
	ReadOnlyAllowDelete types.Bool   `tfsdk:"read_only_allow_delete"`
	Size                types.Int64  `tfsdk:"size"`
	Status              types.String `tfsdk:"status"`
}

func (d *opensearchIndicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_database_opensearch_indices"
}

func (d *opensearchIndicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

func (d *opensearchIndicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "OpenSearch indices",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"service": schema.StringAttribute{
				Description: "Service's UUID for which these indices belongs to",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"indices": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"create_time": schema.StringAttribute{
							Computed:    true,
							Description: "Timestamp indicating the creation time of the index.",
						},
						"docs": schema.Int64Attribute{
							Computed:    true,
							Description: "Number of documents stored in the index.",
						},
						"health": schema.StringAttribute{
							Computed:    true,
							Description: "Health status of the index e.g. `green`, `yellow`, or `red`.",
						},
						"index_name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the index.",
						},
						"number_of_replicas": schema.Int64Attribute{
							Computed:    true,
							Description: "Number of replicas configured for the index.",
						},
						"number_of_shards": schema.Int64Attribute{
							Computed:    true,
							Description: "Number of shards configured & used by the index.",
						},
						"read_only_allow_delete": schema.BoolAttribute{
							Computed:    true,
							Description: "Indicates whether the index is in a read-only state that permits deletion of the entire index. This attribute can be automatically set to true in certain scenarios where the node disk space exceeds the flood stage.",
						},
						"size": schema.Int64Attribute{
							Computed:    true,
							Description: "Size of the index in bytes.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "Status of the index e.g. `open` or `closed`.",
						},
					},
				},
			},
		},
	}
}

func (d *opensearchIndicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data opensearchIndicesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := data.Service.ValueString()

	indices, err := d.client.GetManagedDatabaseIndices(ctx, &request.GetManagedDatabaseIndicesRequest{
		ServiceUUID: serviceID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read OpenSearch indices",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(serviceID)
	data.Service = types.StringValue(serviceID)

	idxModels := make([]opensearchIndexModel, len(indices))
	for i, idx := range indices {
		idxModels[i].CreateTime = types.StringValue(idx.CreateTime.UTC().Format(time.RFC3339Nano))
		idxModels[i].Docs = types.Int64Value(int64(idx.Docs))
		idxModels[i].Health = types.StringValue(idx.Health)
		idxModels[i].IndexName = types.StringValue(idx.IndexName)
		idxModels[i].NumberOfReplicas = types.Int64Value(int64(idx.NumberOfReplicas))
		idxModels[i].NumberOfShards = types.Int64Value(int64(idx.NumberOfShards))
		idxModels[i].ReadOnlyAllowDelete = types.BoolValue(idx.ReadOnlyAllowDelete)
		idxModels[i].Size = types.Int64Value(int64(idx.Size))
		idxModels[i].Status = types.StringValue(idx.Status)
	}

	var diags diag.Diagnostics
	data.Indices, diags = types.SetValueFrom(ctx, data.Indices.ElementType(ctx), idxModels)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
