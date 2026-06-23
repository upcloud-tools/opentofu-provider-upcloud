package gateway

import (
	"context"
	"fmt"
	"regexp"

	"github.com/UpCloudLtd/terraform-provider-upcloud/internal/utils"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/request"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var namePattern = regexp.MustCompile("^[a-zA-Z0-9_-]+$")
var uuidPattern = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

var (
	_ resource.Resource                = &connectionResource{}
	_ resource.ResourceWithConfigure   = &connectionResource{}
	_ resource.ResourceWithImportState = &connectionResource{}
)

func NewConnectionResource() resource.Resource {
	return &connectionResource{}
}

type connectionResource struct {
	client *service.Service
}

type connectionModel struct {
	ID           types.String `tfsdk:"id"`
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	Gateway      types.String `tfsdk:"gateway"`
	Type         types.String `tfsdk:"type"`
	LocalRoute   types.Set    `tfsdk:"local_route"`
	RemoteRoute  types.Set    `tfsdk:"remote_route"`
	Tunnels      types.List   `tfsdk:"tunnels"`
}

type routeModel struct {
	Type          types.String `tfsdk:"type"`
	StaticNetwork types.String `tfsdk:"static_network"`
	Name          types.String `tfsdk:"name"`
}

func (m routeModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":           types.StringType,
		"static_network": types.StringType,
		"name":           types.StringType,
	}
}

func (r *connectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway_connection"
}

func (r *connectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

func (r *connectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gateway connection represents a connection between a gateway and a remote network.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the connection in {gateway UUID}/{connection UUID} format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The UUID of the connection",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:      "The name of the connection, should be unique within the gateway.",
				Required:         true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(namePattern, "must contain only alphanumeric characters, hyphens, and underscores"),
				},
			},
			"gateway": schema.StringAttribute{
				Description: "The ID of the upcloud_gateway resource to which the connection belongs.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of the connection; currently the only supported type is 'ipsec'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("ipsec"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("ipsec"),
				},
			},
			"tunnels": schema.ListAttribute{
				Description: "List of connection's tunnels names.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"local_route": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Type of route; currently the only supported type is 'static'",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("static"),
							Validators: []validator.String{
								stringvalidator.OneOf("static"),
							},
						},
						"static_network": schema.StringAttribute{
							Description: "Destination prefix of the route; needs to be a valid IPv4 prefix",
							Required:    true,
						},
						"name": schema.StringAttribute{
							Description:      "Name of the route",
							Required:         true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 64),
								stringvalidator.RegexMatches(namePattern, "must contain only alphanumeric characters, hyphens, and underscores"),
							},
						},
					},
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"remote_route": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Type of route; currently the only supported type is 'static'",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("static"),
							Validators: []validator.String{
								stringvalidator.OneOf("static"),
							},
						},
						"static_network": schema.StringAttribute{
							Description: "Destination prefix of the route; needs to be a valid IPv4 prefix",
							Required:    true,
						},
						"name": schema.StringAttribute{
							Description:      "Name of the route",
							Required:         true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 64),
								stringvalidator.RegexMatches(namePattern, "must contain only alphanumeric characters, hyphens, and underscores"),
							},
						},
					},
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *connectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data connectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := data.Gateway.ValueString()

	localRoutes, diags := expandRoutesFramework(ctx, data.LocalRoute)
	resp.Diagnostics.Append(diags...)
	remoteRoutes, diags := expandRoutesFramework(ctx, data.RemoteRoute)
	resp.Diagnostics.Append(diags...)

	conn, err := r.client.CreateGatewayConnection(ctx, &request.CreateGatewayConnectionRequest{
		ServiceUUID: serviceID,
		Connection: request.GatewayConnection{
			Name:         data.Name.ValueString(),
			Type:         upcloud.GatewayConnectionType(data.Type.ValueString()),
			LocalRoutes:  localRoutes,
			RemoteRoutes: remoteRoutes,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create gateway connection",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(utils.MarshalID(serviceID, conn.UUID))
	resp.Diagnostics.Append(setConnectionState(ctx, &data, conn)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data connectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceUUID, connUUID, err := parseConnectionID(ctx, r.client, data.ID.ValueString(), data.Gateway.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse connection ID",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(utils.MarshalID(serviceUUID, connUUID))

	conn, err := r.client.GetGatewayConnection(ctx, &request.GetGatewayConnectionRequest{
		ServiceUUID: serviceUUID,
		UUID:        connUUID,
	})
	if err != nil {
		if utils.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to read gateway connection",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(utils.MarshalID(serviceUUID, conn.UUID))
	data.Gateway = types.StringValue(serviceUUID)
	resp.Diagnostics.Append(setConnectionState(ctx, &data, conn)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data connectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceUUID, connUUID, err := parseConnectionID(ctx, r.client, data.ID.ValueString(), data.Gateway.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse connection ID",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	localRoutes, diags := expandRoutesFramework(ctx, data.LocalRoute)
	resp.Diagnostics.Append(diags...)
	remoteRoutes, diags := expandRoutesFramework(ctx, data.RemoteRoute)
	resp.Diagnostics.Append(diags...)

	conn, err := r.client.ModifyGatewayConnection(ctx, &request.ModifyGatewayConnectionRequest{
		ServiceUUID: serviceUUID,
		UUID:        connUUID,
		Connection: request.ModifyGatewayConnection{
			LocalRoutes:  localRoutes,
			RemoteRoutes: remoteRoutes,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to modify gateway connection",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(utils.MarshalID(serviceUUID, conn.UUID))
	resp.Diagnostics.Append(setConnectionState(ctx, &data, conn)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *connectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data connectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceUUID, connUUID, err := parseConnectionID(ctx, r.client, data.ID.ValueString(), data.Gateway.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse connection ID",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	if err := r.client.DeleteGatewayConnection(ctx, &request.DeleteGatewayConnectionRequest{
		ServiceUUID: serviceUUID,
		UUID:        connUUID,
	}); err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete gateway connection",
			utils.ErrorDiagnosticDetail(err),
		)
	}
}

func (r *connectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func parseConnectionID(ctx context.Context, svc *service.Service, id string, gateway string) (serviceUUID, connUUID string, err error) {
	if err := utils.UnmarshalID(id, &serviceUUID, &connUUID); err != nil || connUUID == "" {
		connUUID = id
		serviceUUID = gateway
	}

	if uuidPattern.MatchString(connUUID) {
		return serviceUUID, connUUID, nil
	}

	if serviceUUID == "" {
		return "", "", fmt.Errorf("invalid connection ID %q and no gateway in state", id)
	}

	conns, err := svc.GetGatewayConnections(ctx, &request.GetGatewayConnectionsRequest{ServiceUUID: serviceUUID})
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve connection name %q: %w", connUUID, err)
	}

	for _, conn := range conns {
		if conn.Name == connUUID {
			return serviceUUID, conn.UUID, nil
		}
	}

	return "", "", fmt.Errorf("connection by name %q not found in service %s", connUUID, serviceUUID)
}

func setConnectionState(ctx context.Context, data *connectionModel, conn *upcloud.GatewayConnection) diag.Diagnostics {
	var respDiags diag.Diagnostics

	data.UUID = types.StringValue(conn.UUID)
	data.Name = types.StringValue(conn.Name)
	data.Type = types.StringValue(string(conn.Type))

	localRoutes := make([]routeModel, len(conn.LocalRoutes))
	for i, r := range conn.LocalRoutes {
		localRoutes[i] = routeModel{
			Type:          types.StringValue(string(r.Type)),
			StaticNetwork: types.StringValue(r.StaticNetwork),
			Name:          types.StringValue(r.Name),
		}
	}
	var diags diag.Diagnostics
	data.LocalRoute, diags = types.SetValueFrom(ctx, types.ObjectType{AttrTypes: routeModel{}.AttributeTypes()}, localRoutes)
	respDiags.Append(diags...)

	remoteRoutes := make([]routeModel, len(conn.RemoteRoutes))
	for i, r := range conn.RemoteRoutes {
		remoteRoutes[i] = routeModel{
			Type:          types.StringValue(string(r.Type)),
			StaticNetwork: types.StringValue(r.StaticNetwork),
			Name:          types.StringValue(r.Name),
		}
	}
	data.RemoteRoute, diags = types.SetValueFrom(ctx, types.ObjectType{AttrTypes: routeModel{}.AttributeTypes()}, remoteRoutes)
	respDiags.Append(diags...)

	tunnels := make([]string, len(conn.Tunnels))
	for i, t := range conn.Tunnels {
		tunnels[i] = t.Name
	}
	data.Tunnels, diags = types.ListValueFrom(ctx, types.StringType, tunnels)
	respDiags.Append(diags...)

	return respDiags
}

func expandRoutesFramework(ctx context.Context, routes types.Set) ([]upcloud.GatewayRoute, diag.Diagnostics) {
	if routes.IsNull() || routes.IsUnknown() {
		return nil, nil
	}
	var models []routeModel
	diags := routes.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}
	result := make([]upcloud.GatewayRoute, len(models))
	for i, m := range models {
		result[i] = upcloud.GatewayRoute{
			Type:          upcloud.GatewayRouteType(m.Type.ValueString()),
			StaticNetwork: m.StaticNetwork.ValueString(),
			Name:          m.Name.ValueString(),
		}
	}
	return result, nil
}
