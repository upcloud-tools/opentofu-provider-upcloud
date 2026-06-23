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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &tunnelResource{}
	_ resource.ResourceWithConfigure   = &tunnelResource{}
	_ resource.ResourceWithImportState = &tunnelResource{}
)

func NewTunnelResource() resource.Resource {
	return &tunnelResource{}
}

type tunnelResource struct {
	client *service.Service
}

type tunnelModel struct {
	ID               types.String `tfsdk:"id"`
	UUID             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	ConnectionID     types.String `tfsdk:"connection_id"`
	LocalAddressName types.String `tfsdk:"local_address_name"`
	RemoteAddress    types.String `tfsdk:"remote_address"`
	OperationalState types.String `tfsdk:"operational_state"`
	IPSecAuthPSK     types.Object `tfsdk:"ipsec_auth_psk"`
	IPSecProperties  types.Object `tfsdk:"ipsec_properties"`
}

type ipsecAuthPSKModel struct {
	PSK types.String `tfsdk:"psk"`
}

func (m ipsecAuthPSKModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"psk": types.StringType,
	}
}

type ipsecPropertiesModel struct {
	ChildRekeyTime            types.Int64  `tfsdk:"child_rekey_time"`
	DPDDelay                  types.Int64  `tfsdk:"dpd_delay"`
	DPDTimeout                types.Int64  `tfsdk:"dpd_timeout"`
	IKELifetime               types.Int64  `tfsdk:"ike_lifetime"`
	RekeyTime                 types.Int64  `tfsdk:"rekey_time"`
	Phase1Algorithms          types.Set    `tfsdk:"phase1_algorithms"`
	Phase1DHGroupNumbers      types.Set    `tfsdk:"phase1_dh_group_numbers"`
	Phase1IntegrityAlgorithms types.Set    `tfsdk:"phase1_integrity_algorithms"`
	Phase2Algorithms          types.Set    `tfsdk:"phase2_algorithms"`
	Phase2DHGroupNumbers      types.Set    `tfsdk:"phase2_dh_group_numbers"`
	Phase2IntegrityAlgorithms types.Set    `tfsdk:"phase2_integrity_algorithms"`
}

func (m ipsecPropertiesModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"child_rekey_time":             types.Int64Type,
		"dpd_delay":                    types.Int64Type,
		"dpd_timeout":                  types.Int64Type,
		"ike_lifetime":                 types.Int64Type,
		"rekey_time":                   types.Int64Type,
		"phase1_algorithms":           types.SetType{ElemType: types.StringType},
		"phase1_dh_group_numbers":     types.SetType{ElemType: types.Int64Type},
		"phase1_integrity_algorithms": types.SetType{ElemType: types.StringType},
		"phase2_algorithms":           types.SetType{ElemType: types.StringType},
		"phase2_dh_group_numbers":     types.SetType{ElemType: types.Int64Type},
		"phase2_integrity_algorithms": types.SetType{ElemType: types.StringType},
	}
}

func (r *tunnelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway_connection_tunnel"
}

func (r *tunnelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

func (r *tunnelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gateway connection tunnel represents a tunnel within a gateway connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the tunnel in {service UUID}/{connection UUID}/{tunnel UUID} format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The UUID of the tunnel",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the tunnel, should be unique within the connection",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(namePattern, "must contain only alphanumeric characters, hyphens, and underscores"),
				},
			},
			"connection_id": schema.StringAttribute{
				Description: "ID of the upcloud_gateway_connection resource to which the tunnel belongs",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"local_address_name": schema.StringAttribute{
				Description: "Public (UpCloud) endpoint address of this tunnel",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(namePattern, "must contain only alphanumeric characters, hyphens, and underscores"),
				},
			},
			"remote_address": schema.StringAttribute{
				Description: "Remote public IP address of the tunnel",
				Required:    true,
			},
			"operational_state": schema.StringAttribute{
				Description: "Tunnel's current operational, effective state",
				Computed:    true,
			},
			"ipsec_auth_psk": schema.SingleNestedAttribute{
				Description: "Configuration for authenticating with pre-shared key",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"psk": schema.StringAttribute{
						Description: "The pre-shared key. This value is only used during resource creation and is not returned in the state.",
						Required:    true,
						Sensitive:   true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(8, 64),
							stringvalidator.RegexMatches(
								regexp.MustCompile("^[a-zA-Z1-9_.][a-zA-Z0-9_.]+$"),
								"must contain only alphanumeric characters, underscores, and dots",
							),
						},
					},
				},
			},
			"ipsec_properties": schema.SingleNestedAttribute{
				Description: "IPsec configuration for the tunnel",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"child_rekey_time": schema.Int64Attribute{
						Description: "IKE child SA rekey time in seconds.",
						Optional:    true,
						Computed:    true,
					},
					"dpd_delay": schema.Int64Attribute{
						Description: "Delay before sending Dead Peer Detection packets if no traffic is detected, in seconds.",
						Optional:    true,
						Computed:    true,
					},
					"dpd_timeout": schema.Int64Attribute{
						Description: "Timeout period for DPD reply before considering the peer to be dead, in seconds.",
						Optional:    true,
						Computed:    true,
					},
					"ike_lifetime": schema.Int64Attribute{
						Description: "Maximum IKE SA lifetime in seconds.",
						Optional:    true,
						Computed:    true,
					},
					"rekey_time": schema.Int64Attribute{
						Description: "IKE SA rekey time in seconds.",
						Optional:    true,
						Computed:    true,
					},
					"phase1_algorithms": schema.SetAttribute{
						Description: "List of Phase 1: Proposal algorithms.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
					"phase1_dh_group_numbers": schema.SetAttribute{
						Description: "List of Phase 1 Diffie-Hellman group numbers.",
						ElementType: types.Int64Type,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
					"phase1_integrity_algorithms": schema.SetAttribute{
						Description: "List of Phase 1 integrity algorithms.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
					"phase2_algorithms": schema.SetAttribute{
						Description: "List of Phase 2: Security Association algorithms.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
					"phase2_dh_group_numbers": schema.SetAttribute{
						Description: "List of Phase 2 Diffie-Hellman group numbers.",
						ElementType: types.Int64Type,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
					"phase2_integrity_algorithms": schema.SetAttribute{
						Description: "List of Phase 2 integrity algorithms.",
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func (r *tunnelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data tunnelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceUUID, connectionUUID, err := parseTunnelConnectionID(data.ConnectionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse connection ID",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	ipsec := buildIPSecFromPlan(ctx, data.IPSecProperties)
	auth, diags := buildIPSecAuthFromPlan(ctx, data.IPSecAuthPSK)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ipsec.Authentication = auth

	tunnel, err := r.client.CreateGatewayConnectionTunnel(ctx, &request.CreateGatewayConnectionTunnelRequest{
		ServiceUUID:    serviceUUID,
		ConnectionUUID: connectionUUID,
		Tunnel: request.GatewayTunnel{
			Name: data.Name.ValueString(),
			LocalAddress: upcloud.GatewayTunnelLocalAddress{
				Name: data.LocalAddressName.ValueString(),
			},
			RemoteAddress: upcloud.GatewayTunnelRemoteAddress{
				Address: data.RemoteAddress.ValueString(),
			},
			IPSec: ipsec,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create gateway tunnel",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(utils.MarshalID(serviceUUID, connectionUUID, tunnel.UUID))
	resp.Diagnostics.Append(setTunnelState(ctx, &data, tunnel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tunnelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tunnelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceUUID, connectionUUID, tunnelUUID, err := parseTunnelID(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse tunnel ID",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(utils.MarshalID(serviceUUID, connectionUUID, tunnelUUID))

	tunnel, err := r.client.GetGatewayConnectionTunnel(ctx, &request.GetGatewayConnectionTunnelRequest{
		ServiceUUID:    serviceUUID,
		ConnectionUUID: connectionUUID,
		UUID:           tunnelUUID,
	})
	if err != nil {
		if utils.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to read gateway tunnel",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(utils.MarshalID(serviceUUID, connectionUUID, tunnel.UUID))
	data.ConnectionID = types.StringValue(utils.MarshalID(serviceUUID, connectionUUID))
	resp.Diagnostics.Append(setTunnelState(ctx, &data, tunnel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tunnelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data tunnelModel
	var state tunnelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceUUID, connectionUUID, tunnelUUID, err := parseTunnelID(ctx, r.client, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse tunnel ID",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	modifyReq := request.ModifyGatewayConnectionTunnelRequest{
		ServiceUUID:    serviceUUID,
		ConnectionUUID: connectionUUID,
		UUID:           tunnelUUID,
		Tunnel: request.ModifyGatewayTunnel{
			Name: data.Name.ValueString(),
		},
	}

	if !data.IPSecProperties.IsNull() && !data.IPSecProperties.IsUnknown() {
		ipsec := buildIPSecFromPlan(ctx, data.IPSecProperties)
		modifyReq.Tunnel.IPSec = &ipsec
	}
	if !data.LocalAddressName.IsUnknown() {
		modifyReq.Tunnel.LocalAddress = &upcloud.GatewayTunnelLocalAddress{
			Name: data.LocalAddressName.ValueString(),
		}
	}
	if !data.RemoteAddress.IsUnknown() {
		modifyReq.Tunnel.RemoteAddress = &upcloud.GatewayTunnelRemoteAddress{
			Address: data.RemoteAddress.ValueString(),
		}
	}

	tunnel, err := r.client.ModifyGatewayConnectionTunnel(ctx, &modifyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to modify gateway tunnel",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(utils.MarshalID(serviceUUID, connectionUUID, tunnel.UUID))
	data.ConnectionID = types.StringValue(utils.MarshalID(serviceUUID, connectionUUID))
	resp.Diagnostics.Append(setTunnelState(ctx, &data, tunnel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tunnelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tunnelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceUUID, connectionUUID, tunnelUUID, err := parseTunnelID(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse tunnel ID",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	if err := r.client.DeleteGatewayConnectionTunnel(ctx, &request.DeleteGatewayConnectionTunnelRequest{
		ServiceUUID:    serviceUUID,
		ConnectionUUID: connectionUUID,
		UUID:           tunnelUUID,
	}); err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete gateway tunnel",
			utils.ErrorDiagnosticDetail(err),
		)
	}
}

func (r *tunnelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func parseTunnelConnectionID(id string) (serviceUUID, connectionUUID string, err error) {
	if err := utils.UnmarshalID(id, &serviceUUID, &connectionUUID); err != nil {
		return "", "", fmt.Errorf("invalid tunnel connection ID %q", id)
	}
	return serviceUUID, connectionUUID, nil
}

type tunnelLookup interface {
	GetGatewayConnections(ctx context.Context, r *request.GetGatewayConnectionsRequest) ([]upcloud.GatewayConnection, error)
	GetGatewayConnectionTunnels(ctx context.Context, r *request.GetGatewayConnectionTunnelsRequest) ([]upcloud.GatewayTunnel, error)
}

func parseTunnelID(ctx context.Context, svc tunnelLookup, id string) (serviceUUID, connectionUUID, tunnelUUID string, err error) {
	if err := utils.UnmarshalID(id, &serviceUUID, &connectionUUID, &tunnelUUID); err != nil {
		return "", "", "", fmt.Errorf("invalid tunnel ID %q", id)
	}

	if uuidPattern.MatchString(connectionUUID) && uuidPattern.MatchString(tunnelUUID) {
		return serviceUUID, connectionUUID, tunnelUUID, nil
	}

	return migrateTunnelID(ctx, svc, serviceUUID, connectionUUID, tunnelUUID)
}

func migrateTunnelID(ctx context.Context, svc tunnelLookup, svcUUID, connName, tunName string) (serviceUUID, connectionUUID, tunnelUUID string, err error) {
	conns, err := svc.GetGatewayConnections(ctx, &request.GetGatewayConnectionsRequest{ServiceUUID: svcUUID})
	if err != nil {
		return "", "", "", fmt.Errorf("failed to resolve connection name %q: %w", connName, err)
	}

	var connUUID string
	for _, conn := range conns {
		if conn.Name == connName {
			connUUID = conn.UUID
			break
		}
	}
	if connUUID == "" {
		return "", "", "", fmt.Errorf("connection by name %q not found in service %s", connName, svcUUID)
	}

	tuns, err := svc.GetGatewayConnectionTunnels(ctx, &request.GetGatewayConnectionTunnelsRequest{
		ServiceUUID:    svcUUID,
		ConnectionUUID: connUUID,
	})
	if err != nil {
		return "", "", "", fmt.Errorf("failed to resolve tunnel name %q: %w", tunName, err)
	}

	var tunUUID string
	for _, tun := range tuns {
		if tun.Name == tunName {
			tunUUID = tun.UUID
			break
		}
	}
	if tunUUID == "" {
		return "", "", "", fmt.Errorf("tunnel by name %q not found in connection %s", tunName, connUUID)
	}

	return svcUUID, connUUID, tunUUID, nil
}

func setTunnelState(ctx context.Context, data *tunnelModel, tunnel *upcloud.GatewayTunnel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	data.UUID = types.StringValue(tunnel.UUID)
	data.Name = types.StringValue(tunnel.Name)
	data.LocalAddressName = types.StringValue(tunnel.LocalAddress.Name)
	data.RemoteAddress = types.StringValue(tunnel.RemoteAddress.Address)
	data.OperationalState = types.StringValue(string(tunnel.OperationalState))

	auth, diags := types.ObjectValueFrom(ctx, ipsecAuthPSKModel{}.AttributeTypes(), ipsecAuthPSKModel{})
	respDiags.Append(diags...)
	data.IPSecAuthPSK = auth

	ipsec := ipsecPropertiesModel{
		ChildRekeyTime: types.Int64Value(int64(tunnel.IPSec.ChildRekeyTime)),
		DPDDelay:       types.Int64Value(int64(tunnel.IPSec.DPDDelay)),
		DPDTimeout:     types.Int64Value(int64(tunnel.IPSec.DPDTimeout)),
		IKELifetime:    types.Int64Value(int64(tunnel.IPSec.IKELifetime)),
		RekeyTime:      types.Int64Value(int64(tunnel.IPSec.RekeyTime)),
	}

	var diags2 diag.Diagnostics
	ipsec.Phase1Algorithms, diags2 = types.SetValueFrom(ctx, types.StringType, tunnel.IPSec.Phase1Algorithms)
	respDiags.Append(diags2...)
	ipsec.Phase1DHGroupNumbers, diags2 = types.SetValueFrom(ctx, types.Int64Type, tunnel.IPSec.Phase1DHGroupNumbers)
	respDiags.Append(diags2...)
	ipsec.Phase1IntegrityAlgorithms, diags2 = types.SetValueFrom(ctx, types.StringType, tunnel.IPSec.Phase1IntegrityAlgorithms)
	respDiags.Append(diags2...)
	ipsec.Phase2Algorithms, diags2 = types.SetValueFrom(ctx, types.StringType, tunnel.IPSec.Phase2Algorithms)
	respDiags.Append(diags2...)
	ipsec.Phase2DHGroupNumbers, diags2 = types.SetValueFrom(ctx, types.Int64Type, tunnel.IPSec.Phase2DHGroupNumbers)
	respDiags.Append(diags2...)
	ipsec.Phase2IntegrityAlgorithms, diags2 = types.SetValueFrom(ctx, types.StringType, tunnel.IPSec.Phase2IntegrityAlgorithms)
	respDiags.Append(diags2...)

	data.IPSecProperties, diags2 = types.ObjectValueFrom(ctx, ipsecPropertiesModel{}.AttributeTypes(), ipsec)
	respDiags.Append(diags2...)

	return respDiags
}

func buildIPSecFromPlan(ctx context.Context, obj types.Object) upcloud.GatewayTunnelIPSec {
	var props ipsecPropertiesModel
	diags := obj.As(ctx, &props, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return upcloud.GatewayTunnelIPSec{}
	}

	result := upcloud.GatewayTunnelIPSec{
		ChildRekeyTime: int(props.ChildRekeyTime.ValueInt64()),
		DPDDelay:       int(props.DPDDelay.ValueInt64()),
		DPDTimeout:     int(props.DPDTimeout.ValueInt64()),
		IKELifetime:    int(props.IKELifetime.ValueInt64()),
		RekeyTime:      int(props.RekeyTime.ValueInt64()),
	}

	var strAlgos []string
	props.Phase1Algorithms.ElementsAs(ctx, &strAlgos, false)
	for _, a := range strAlgos {
		result.Phase1Algorithms = append(result.Phase1Algorithms, upcloud.GatewayIPSecAlgorithm(a))
	}

	var dhGroups []int64
	props.Phase1DHGroupNumbers.ElementsAs(ctx, &dhGroups, false)
	for _, n := range dhGroups {
		result.Phase1DHGroupNumbers = append(result.Phase1DHGroupNumbers, int(n))
	}

	var intAlgos []string
	props.Phase1IntegrityAlgorithms.ElementsAs(ctx, &intAlgos, false)
	for _, a := range intAlgos {
		result.Phase1IntegrityAlgorithms = append(result.Phase1IntegrityAlgorithms, upcloud.GatewayIPSecIntegrityAlgorithm(a))
	}

	props.Phase2Algorithms.ElementsAs(ctx, &strAlgos, false)
	for _, a := range strAlgos {
		result.Phase2Algorithms = append(result.Phase2Algorithms, upcloud.GatewayIPSecAlgorithm(a))
	}

	props.Phase2DHGroupNumbers.ElementsAs(ctx, &dhGroups, false)
	for _, n := range dhGroups {
		result.Phase2DHGroupNumbers = append(result.Phase2DHGroupNumbers, int(n))
	}

	props.Phase2IntegrityAlgorithms.ElementsAs(ctx, &intAlgos, false)
	for _, a := range intAlgos {
		result.Phase2IntegrityAlgorithms = append(result.Phase2IntegrityAlgorithms, upcloud.GatewayIPSecIntegrityAlgorithm(a))
	}

	return result
}

func buildIPSecAuthFromPlan(ctx context.Context, obj types.Object) (upcloud.GatewayTunnelIPSecAuth, diag.Diagnostics) {
	var auth ipsecAuthPSKModel
	diags := obj.As(ctx, &auth, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return upcloud.GatewayTunnelIPSecAuth{}, diags
	}
	return upcloud.GatewayTunnelIPSecAuth{
		Authentication: upcloud.GatewayTunnelIPSecAuthTypePSK,
		PSK:            auth.PSK.ValueString(),
	}, nil
}
