package server

import (
	"context"
	"strings"

	"github.com/UpCloudLtd/terraform-provider-upcloud/internal/utils"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/request"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewServerDataSource() datasource.DataSource {
	return &serverDataSource{}
}

var (
	_ datasource.DataSource              = &serverDataSource{}
	_ datasource.DataSourceWithConfigure = &serverDataSource{}
)

type serverDataSource struct {
	client *service.Service
}

func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

type serverDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Hostname          types.String `tfsdk:"hostname"`
	Title             types.String `tfsdk:"title"`
	Zone              types.String `tfsdk:"zone"`
	State             types.String `tfsdk:"state"`
	CPU               types.Int64  `tfsdk:"cpu"`
	Mem               types.Int64  `tfsdk:"mem"`
	Plan              types.String `tfsdk:"plan"`
	Host              types.Int64  `tfsdk:"host"`
	BootOrder         types.String `tfsdk:"boot_order"`
	Timezone          types.String `tfsdk:"timezone"`
	VideoModel        types.String `tfsdk:"video_model"`
	NICModel          types.String `tfsdk:"nic_model"`
	Firewall          types.Bool   `tfsdk:"firewall"`
	Metadata          types.Bool   `tfsdk:"metadata"`
	ServerGroup       types.String `tfsdk:"server_group"`
	Tags              types.Set    `tfsdk:"tags"`
	Labels            types.Map    `tfsdk:"labels"`
	SimpleBackup      types.List   `tfsdk:"simple_backup"`
	StorageDevices    types.Set    `tfsdk:"storage_devices"`
	NetworkInterfaces types.List   `tfsdk:"network_interface"`
}

func storageDeviceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"address":          types.StringType,
		"address_position": types.StringType,
		"uuid":             types.StringType,
		"type":             types.StringType,
		"title":            types.StringType,
		"tier":             types.StringType,
		"size":             types.Int64Type,
		"boot_disk":        types.BoolType,
		"encrypted":        types.BoolType,
	}
}

func networkInterfaceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"index":               types.Int64Type,
		"type":                types.StringType,
		"ip_address":          types.StringType,
		"ip_address_family":   types.StringType,
		"mac":                 types.StringType,
		"network":             types.StringType,
		"source_ip_filtering": types.BoolType,
		"bootable":            types.BoolType,
	}
}

func simpleBackupAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"plan": types.StringType,
		"time": types.StringType,
	}
}

func (d *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Provides details of an UpCloud cloud server. Use this data source to get the details of a server by its UUID.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the server.",
				Required:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname of the server.",
				Computed:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "A short, informational description of the server.",
				Computed:            true,
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "The zone the server is in, e.g. `de-fra1`.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The current state of the server.",
				Computed:            true,
			},
			"cpu": schema.Int64Attribute{
				MarkdownDescription: "The number of CPU cores assigned to the server.",
				Computed:            true,
			},
			"mem": schema.Int64Attribute{
				MarkdownDescription: "The amount of memory assigned to the server in MB.",
				Computed:            true,
			},
			"plan": schema.StringAttribute{
				MarkdownDescription: "The pricing plan used by the server.",
				Computed:            true,
			},
			"host": schema.Int64Attribute{
				MarkdownDescription: "The host ID where the server is running.",
				Computed:            true,
			},
			"boot_order": schema.StringAttribute{
				MarkdownDescription: "The boot order of the server.",
				Computed:            true,
			},
			"timezone": schema.StringAttribute{
				MarkdownDescription: "The timezone of the server.",
				Computed:            true,
			},
			"video_model": schema.StringAttribute{
				MarkdownDescription: "The video model of the server.",
				Computed:            true,
			},
			"nic_model": schema.StringAttribute{
				MarkdownDescription: "The model of the network interface.",
				Computed:            true,
			},
			"firewall": schema.BoolAttribute{
				MarkdownDescription: "Whether firewall is enabled on the server.",
				Computed:            true,
			},
			"metadata": schema.BoolAttribute{
				MarkdownDescription: "Whether metadata service is enabled on the server.",
				Computed:            true,
			},
			"server_group": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server group the server belongs to.",
				Computed:            true,
			},
			"tags": schema.SetAttribute{
				MarkdownDescription: "The tags assigned to the server.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"labels": utils.ReadOnlyLabelsAttribute("server"),
			"simple_backup": schema.ListAttribute{
				MarkdownDescription: "The simple backup schedule configuration.",
				Computed:            true,
				ElementType: types.ObjectType{
					AttrTypes: simpleBackupAttrTypes(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"storage_devices": schema.SetNestedBlock{
				MarkdownDescription: "The storage devices attached to the server.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							MarkdownDescription: "The address of the storage device.",
							Computed:            true,
						},
						"address_position": schema.StringAttribute{
							MarkdownDescription: "The address position of the storage device.",
							Computed:            true,
						},
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The UUID of the storage.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of the storage device.",
							Computed:            true,
						},
						"title": schema.StringAttribute{
							MarkdownDescription: "The title of the storage.",
							Computed:            true,
						},
						"tier": schema.StringAttribute{
							MarkdownDescription: "The storage tier.",
							Computed:            true,
						},
						"size": schema.Int64Attribute{
							MarkdownDescription: "The size of the storage in GB.",
							Computed:            true,
						},
						"boot_disk": schema.BoolAttribute{
							MarkdownDescription: "Whether this storage is the boot disk.",
							Computed:            true,
						},
						"encrypted": schema.BoolAttribute{
							MarkdownDescription: "Whether the storage is encrypted.",
							Computed:            true,
						},
					},
				},
			},
			"network_interface": schema.ListNestedBlock{
				MarkdownDescription: "The network interfaces of the server.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"index": schema.Int64Attribute{
							MarkdownDescription: "The index of the network interface.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of the network interface.",
							Computed:            true,
						},
						"ip_address": schema.StringAttribute{
							MarkdownDescription: "The primary IP address of the interface.",
							Computed:            true,
						},
						"ip_address_family": schema.StringAttribute{
							MarkdownDescription: "The IP address family (IPv4 or IPv6).",
							Computed:            true,
						},
						"mac": schema.StringAttribute{
							MarkdownDescription: "The MAC address of the interface.",
							Computed:            true,
						},
						"network": schema.StringAttribute{
							MarkdownDescription: "The UUID of the network the interface is connected to.",
							Computed:            true,
						},
						"source_ip_filtering": schema.BoolAttribute{
							MarkdownDescription: "Whether source IP filtering is enabled.",
							Computed:            true,
						},
						"bootable": schema.BoolAttribute{
							MarkdownDescription: "Whether the interface is bootable.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := d.client.GetServerDetails(ctx, &request.GetServerDetailsRequest{
		UUID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read server details",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	resp.Diagnostics.Append(setDataSourceValues(ctx, &data, server)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setDataSourceValues(ctx context.Context, data *serverDataSourceModel, server *upcloud.ServerDetails) diag.Diagnostics {
	var respDiagnostics, diags diag.Diagnostics

	data.ID = types.StringValue(server.UUID)
	data.Hostname = types.StringValue(server.Hostname)
	data.Title = types.StringValue(server.Title)
	data.Zone = types.StringValue(server.Zone)
	data.State = types.StringValue(server.State)
	data.CPU = types.Int64Value(int64(server.CoreNumber))
	data.Mem = types.Int64Value(int64(server.MemoryAmount))
	data.Plan = types.StringValue(server.Plan)
	data.Host = types.Int64Value(server.HostID)
	data.BootOrder = types.StringValue(server.BootOrder)
	data.Timezone = types.StringValue(server.Timezone)
	data.VideoModel = types.StringValue(server.VideoModel)
	data.NICModel = types.StringValue(server.NICModel)

	if server.Firewall == "on" {
		data.Firewall = types.BoolValue(true)
	} else {
		data.Firewall = types.BoolValue(false)
	}

	data.Metadata = types.BoolValue(server.Metadata.Bool())

	if server.ServerGroup != "" {
		data.ServerGroup = types.StringValue(server.ServerGroup)
	} else {
		data.ServerGroup = types.StringNull()
	}

	// Tags
	if len(server.Tags) > 0 {
		data.Tags, diags = types.SetValueFrom(ctx, types.StringType, server.Tags)
		respDiagnostics.Append(diags...)
	} else {
		data.Tags = types.SetNull(types.StringType)
	}

	// Labels
	data.Labels, diags = types.MapValueFrom(ctx, types.StringType, utils.LabelsSliceToMap(server.Labels))
	respDiagnostics.Append(diags...)

	// Simple backup
	if server.SimpleBackup != "" && server.SimpleBackup != "no" {
		parts := strings.Split(server.SimpleBackup, ",")
		if n := len(parts); n == 2 {
			sbVal, diags := types.ObjectValue(simpleBackupAttrTypes(), map[string]attr.Value{
				"plan": types.StringValue(parts[1]),
				"time": types.StringValue(parts[0]),
			})
			respDiagnostics.Append(diags...)
			sbList, diags := types.ListValue(types.ObjectType{AttrTypes: simpleBackupAttrTypes()}, []attr.Value{sbVal})
			respDiagnostics.Append(diags...)
			data.SimpleBackup = sbList
		} else {
			data.SimpleBackup = types.ListNull(types.ObjectType{AttrTypes: simpleBackupAttrTypes()})
		}
	} else {
		data.SimpleBackup = types.ListNull(types.ObjectType{AttrTypes: simpleBackupAttrTypes()})
	}

	// Storage devices
	storageDevices := make([]attr.Value, 0, len(server.StorageDevices))
	for _, dev := range server.StorageDevices {
		storageObj, diags := types.ObjectValue(storageDeviceAttrTypes(), map[string]attr.Value{
			"address":          types.StringValue(utils.StorageAddressFormat(dev.Address)),
			"address_position": types.StringValue(utils.StorageAddressPositionFormat(dev.Address)),
			"uuid":             types.StringValue(dev.UUID),
			"type":             types.StringValue(dev.Type),
			"title":            types.StringValue(dev.Title),
			"tier":             types.StringValue(dev.Tier),
			"size":             types.Int64Value(int64(dev.Size)),
			"boot_disk":        types.BoolValue(dev.BootDisk == 1),
			"encrypted":        types.BoolValue(dev.Encrypted.Bool()),
		})
		respDiagnostics.Append(diags...)
		storageDevices = append(storageDevices, storageObj)
	}
	data.StorageDevices, diags = types.SetValue(types.ObjectType{AttrTypes: storageDeviceAttrTypes()}, storageDevices)
	respDiagnostics.Append(diags...)

	// Network interfaces
	networkInterfaces := make([]attr.Value, 0, len(server.Networking.Interfaces))
	for _, iface := range server.Networking.Interfaces {
		primaryIP := ""
		primaryFamily := ""
		for _, ip := range iface.IPAddresses {
			if ip.Family == upcloud.IPAddressFamilyIPv4 {
				primaryIP = ip.Address
				primaryFamily = ip.Family
				break
			}
		}
		if primaryIP == "" && len(iface.IPAddresses) > 0 {
			primaryIP = iface.IPAddresses[0].Address
			primaryFamily = iface.IPAddresses[0].Family
		}

		nicObj, diags := types.ObjectValue(networkInterfaceAttrTypes(), map[string]attr.Value{
			"index":               types.Int64Value(int64(iface.Index)),
			"type":                types.StringValue(iface.Type),
			"ip_address":          types.StringValue(primaryIP),
			"ip_address_family":   types.StringValue(primaryFamily),
			"mac":                 types.StringValue(iface.MAC),
			"network":             types.StringValue(iface.Network),
			"source_ip_filtering": types.BoolValue(iface.SourceIPFiltering.Bool()),
			"bootable":            types.BoolValue(iface.Bootable.Bool()),
		})
		respDiagnostics.Append(diags...)
		networkInterfaces = append(networkInterfaces, nicObj)
	}
	data.NetworkInterfaces, diags = types.ListValue(types.ObjectType{AttrTypes: networkInterfaceAttrTypes()}, networkInterfaces)
	respDiagnostics.Append(diags...)

	return respDiagnostics
}
