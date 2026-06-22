package network

import (
	"context"
	"regexp"
	"time"

	"github.com/UpCloudLtd/terraform-provider-upcloud/internal/utils"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/request"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &networksDataSource{}
	_ datasource.DataSourceWithConfigure = &networksDataSource{}
)

func NewNetworksDataSource() datasource.DataSource {
	return &networksDataSource{}
}

type networksDataSource struct {
	client *service.Service
}

type networksModel struct {
	ID         types.String `tfsdk:"id"`
	Zone       types.String `tfsdk:"zone"`
	FilterName types.String `tfsdk:"filter_name"`
	Networks   types.Set    `tfsdk:"networks"`
}

type networkDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	Zone      types.String `tfsdk:"zone"`
	IPNetwork types.Set    `tfsdk:"ip_network"`
	Servers   types.Set    `tfsdk:"servers"`
}

type dsIPNetworkModel struct {
	Address          types.String `tfsdk:"address"`
	DHCP             types.Bool   `tfsdk:"dhcp"`
	DHCPDefaultRoute types.Bool   `tfsdk:"dhcp_default_route"`
	DHCPDns          types.List   `tfsdk:"dhcp_dns"`
	DHCPRoutes       types.Set    `tfsdk:"dhcp_routes"`
	Family           types.String `tfsdk:"family"`
	Gateway          types.String `tfsdk:"gateway"`
}

type networkServerModel struct {
	ID    types.String `tfsdk:"id"`
	Title types.String `tfsdk:"title"`
}

func (d *networksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networks"
}

func (d *networksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

func (d *networksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to get the available UpCloud networks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"zone": schema.StringAttribute{
				Description: "If specified, this data source will return only networks from this zone",
				Optional:    true,
			},
			"filter_name": schema.StringAttribute{
				Description: "If specified, results will be filtered to match name using a regular expression",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"networks": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The UUID of the network",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "A valid name for the network",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The network type",
						},
						"zone": schema.StringAttribute{
							Computed:    true,
							Description: "The zone the network is in, e.g. `de-fra1`.",
						},
					},
					Blocks: map[string]schema.Block{
						"ip_network": schema.SetNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"address": schema.StringAttribute{
										Computed:    true,
										Description: "The CIDR range of the subnet",
									},
									"dhcp": schema.BoolAttribute{
										Computed:    true,
										Description: "Is DHCP enabled?",
									},
									"dhcp_default_route": schema.BoolAttribute{
										Computed:    true,
										Description: "Is the gateway the DHCP default route?",
									},
									"dhcp_dns": schema.ListAttribute{
										Computed:    true,
										Description: "The DNS servers given by DHCP",
										ElementType: types.StringType,
									},
									"dhcp_routes": schema.SetAttribute{
										Computed:    true,
										Description: "The additional DHCP classless static routes given by DHCP",
										ElementType: types.StringType,
									},
									"family": schema.StringAttribute{
										Computed:    true,
										Description: "IP address family",
									},
									"gateway": schema.StringAttribute{
										Computed:    true,
										Description: "Gateway address given by DHCP",
									},
								},
							},
						},
						"servers": schema.SetNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:    true,
										Description: "The UUID of the server",
									},
									"title": schema.StringAttribute{
										Computed:    true,
										Description: "The short description of the server",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *networksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data networksModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone := data.Zone.ValueString()
	filterName := data.FilterName.ValueString()

	var err error
	var fetchedNetworks *upcloud.Networks
	if zone != "" {
		fetchedNetworks, err = d.client.GetNetworksInZone(ctx, &request.GetNetworksInZoneRequest{
			Zone: zone,
		})
	} else {
		fetchedNetworks, err = d.client.GetNetworks(ctx)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read networks",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	filteredNetworks := fetchedNetworks.Networks
	if filterName != "" {
		filteredNetworks, err = utils.FilterNetworks(fetchedNetworks.Networks, func(n upcloud.Network) (bool, error) {
			return regexp.MatchString(filterName, n.Name)
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to filter networks",
				utils.ErrorDiagnosticDetail(err),
			)
			return
		}
	}

	networkModels := make([]networkDataSourceModel, 0, len(filteredNetworks))
	for _, fn := range filteredNetworks {
		n := networkDataSourceModel{
			ID:   types.StringValue(fn.UUID),
			Name: types.StringValue(fn.Name),
			Type: types.StringValue(fn.Type),
			Zone: types.StringValue(fn.Zone),
		}

		ipnModels := make([]dsIPNetworkModel, 0, len(fn.IPNetworks))
		for _, fipn := range fn.IPNetworks {
			ipn := dsIPNetworkModel{
				Address:          types.StringValue(fipn.Address),
				DHCP:             utils.AsBool(fipn.DHCP),
				DHCPDefaultRoute: utils.AsBool(fipn.DHCPDefaultRoute),
				Family:           types.StringValue(fipn.Family),
				Gateway:          types.StringValue(fipn.Gateway),
			}

			var diags diag.Diagnostics
			ipn.DHCPDns, diags = types.ListValueFrom(ctx, types.StringType, fipn.DHCPDns)
			resp.Diagnostics.Append(diags...)

			ipn.DHCPRoutes, diags = types.SetValueFrom(ctx, types.StringType, fipn.DHCPRoutes)
			resp.Diagnostics.Append(diags...)

			ipnModels = append(ipnModels, ipn)
		}

		var diags diag.Diagnostics
		n.IPNetwork, diags = types.SetValueFrom(ctx, n.IPNetwork.ElementType(ctx), ipnModels)
		resp.Diagnostics.Append(diags...)

		serverModels := make([]networkServerModel, 0, len(fn.Servers))
		for _, s := range fn.Servers {
			serverModels = append(serverModels, networkServerModel{
				ID:    types.StringValue(s.ServerUUID),
				Title: types.StringValue(s.ServerTitle),
			})
		}
		n.Servers, diags = types.SetValueFrom(ctx, n.Servers.ElementType(ctx), serverModels)
		resp.Diagnostics.Append(diags...)

		networkModels = append(networkModels, n)
	}

	var diags diag.Diagnostics
	data.Networks, diags = types.SetValueFrom(ctx, data.Networks.ElementType(ctx), networkModels)
	resp.Diagnostics.Append(diags...)

	data.ID = types.StringValue(time.Now().UTC().Format(time.RFC3339Nano))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
