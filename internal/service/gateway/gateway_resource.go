package gateway

import (
	"context"
	"fmt"
	"regexp"
	"time"

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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	cleanupWaitTimeSeconds = 15
)

var (
	_ resource.Resource                = &gatewayResource{}
	_ resource.ResourceWithConfigure   = &gatewayResource{}
	_ resource.ResourceWithImportState = &gatewayResource{}
)

func NewGatewayResource() resource.Resource {
	return &gatewayResource{}
}

type gatewayResource struct {
	client *service.Service
}

type gatewayModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Zone             types.String `tfsdk:"zone"`
	Features         types.Set    `tfsdk:"features"`
	Router           types.Object `tfsdk:"router"`
	Labels           types.Map    `tfsdk:"labels"`
	ConfiguredStatus types.String `tfsdk:"configured_status"`
	OperationalState types.String `tfsdk:"operational_state"`
	Plan             types.String `tfsdk:"plan"`
	Address          types.Object `tfsdk:"address"`
	Connections      types.List   `tfsdk:"connections"`
	Addresses        types.Set    `tfsdk:"addresses"`
}

type gatewayRouterModel struct {
	ID types.String `tfsdk:"id"`
}

func (m gatewayRouterModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType,
	}
}

type gatewayAddressModel struct {
	Address types.String `tfsdk:"address"`
	Name    types.String `tfsdk:"name"`
}

func (m gatewayAddressModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"address": types.StringType,
		"name":    types.StringType,
	}
}

func (r *gatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway"
}

func (r *gatewayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

func (r *gatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Network gateways connect SDN Private Networks to external IP networks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the gateway",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Gateway name. Needs to be unique within the account.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(
						regexp.MustCompile("^[a-zA-Z0-9_-]+$"),
						"must contain only alphanumeric characters, hyphens, and underscores",
					),
				},
			},
			"zone": schema.StringAttribute{
				Description: "Zone in which the gateway will be hosted, e.g. `de-fra1`.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"features": schema.SetAttribute{
				Description: "Features enabled for the gateway. Valid item values are `nat` and `vpn`.",
				ElementType: types.StringType,
				Required:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"labels": utils.LabelsAttribute("network gateway"),
			"configured_status": schema.StringAttribute{
				Description: "The service configured status indicates the service's current intended status. Managed by the customer.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(string(upcloud.GatewayConfiguredStatusStarted)),
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(upcloud.GatewayConfiguredStatusStarted),
						string(upcloud.GatewayConfiguredStatusStopped),
					),
				},
			},
			"operational_state": schema.StringAttribute{
				Description: "The service operational state indicates the service's current operational, effective state. Managed by the system.",
				Computed:    true,
			},
			"plan": schema.StringAttribute{
				Description: "Gateway pricing plan.",
				Optional:    true,
				Computed:    true,
			},
			"router": schema.SingleNestedAttribute{
				Description: "Attached Router from where traffic is routed towards the network gateway service.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "ID of the router attached to the gateway.",
						Required:    true,
					},
				},
			},
			"address": schema.SingleNestedAttribute{
				Description: "IP addresses assigned to the gateway.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"address": schema.StringAttribute{
						Description: "IP address",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the IP address",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"connections": schema.ListAttribute{
				Description: "Names of connections attached to the gateway.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"addresses": schema.SetNestedAttribute{
				DeprecationMessage: "Use 'address' attribute instead. This attribute will be removed in the next major version of the provider",
				Description:        "IP addresses assigned to the gateway.",
				Computed:           true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Description: "IP address",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the IP address",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *gatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data gatewayModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	features := []upcloud.GatewayFeature{}
	resp.Diagnostics.Append(data.Features.ElementsAs(ctx, &features, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var router gatewayRouterModel
	resp.Diagnostics.Append(data.Router.As(ctx, &router, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	var labels map[string]string
	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
	}

	createReq := &request.CreateGatewayRequest{
		Name: data.Name.ValueString(),
		Zone: data.Zone.ValueString(),
		Plan: data.Plan.ValueString(),
		Routers: []request.GatewayRouter{
			{UUID: router.ID.ValueString()},
		},
		Labels:           utils.LabelsMapToSlice(labels),
		ConfiguredStatus: upcloud.GatewayConfiguredStatus(data.ConfiguredStatus.ValueString()),
	}
	createReq.Features = features

	if !data.Address.IsNull() && !data.Address.IsUnknown() {
		var addr gatewayAddressModel
		resp.Diagnostics.Append(data.Address.As(ctx, &addr, basetypes.ObjectAsOptions{})...)
		if addr.Name.ValueString() != "" {
			createReq.Addresses = []upcloud.GatewayAddress{
				{Name: addr.Name.ValueString()},
			}
		}
	}

	gw, err := r.client.CreateGateway(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create gateway",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(gw.UUID)

	gw, err = waitForGatewayToBeRunning(ctx, r.client, gw.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to wait for gateway to be running",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	resp.Diagnostics.Append(setGatewayState(ctx, &data, gw)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gatewayModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gw, err := r.client.GetGateway(ctx, &request.GetGatewayRequest{UUID: data.ID.ValueString()})
	if err != nil {
		if utils.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to read gateway",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	resp.Diagnostics.Append(setGatewayState(ctx, &data, gw)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data gatewayModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	modifyReq := request.ModifyGatewayRequest{
		UUID: data.ID.ValueString(),
	}

	if !data.Name.IsUnknown() {
		modifyReq.Name = data.Name.ValueString()
	}
	if !data.Plan.IsUnknown() {
		modifyReq.Plan = data.Plan.ValueString()
	}
	if !data.ConfiguredStatus.IsUnknown() {
		modifyReq.ConfiguredStatus = upcloud.GatewayConfiguredStatus(data.ConfiguredStatus.ValueString())
	}
	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		var labels map[string]string
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		modifyReq.Labels = utils.LabelsMapToSlice(labels)
	}

	gw, err := r.client.ModifyGateway(ctx, &modifyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to modify gateway",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	resp.Diagnostics.Append(setGatewayState(ctx, &data, gw)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *gatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gatewayModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteGateway(ctx, &request.DeleteGatewayRequest{UUID: data.ID.ValueString()}); err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete gateway",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	if err := waitForGatewayToBeDeleted(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Unable to wait for gateway to be deleted",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	time.Sleep(time.Second * cleanupWaitTimeSeconds)
}

func (r *gatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func setGatewayState(ctx context.Context, data *gatewayModel, gw *upcloud.Gateway) diag.Diagnostics {
	var respDiags diag.Diagnostics

	data.Name = types.StringValue(gw.Name)
	data.Zone = types.StringValue(gw.Zone)
	data.Plan = types.StringValue(gw.Plan)
	data.OperationalState = types.StringValue(string(gw.OperationalState))
	data.ConfiguredStatus = types.StringValue(string(gw.ConfiguredStatus))

	features, diags := types.SetValueFrom(ctx, types.StringType, gw.Features)
	respDiags.Append(diags...)
	data.Features = features

	router, diags := types.ObjectValueFrom(ctx, gatewayRouterModel{}.AttributeTypes(), gatewayRouterModel{
		ID: types.StringValue(gw.Routers[0].UUID),
	})
	respDiags.Append(diags...)
	data.Router = router

	data.Labels, diags = types.MapValueFrom(ctx, types.StringType, utils.LabelsSliceToMap(gw.Labels))
	respDiags.Append(diags...)

	if len(gw.Addresses) > 0 {
		addr := gw.Addresses[0]
		address, diags := types.ObjectValueFrom(ctx, gatewayAddressModel{}.AttributeTypes(), gatewayAddressModel{
			Address: types.StringValue(addr.Address),
			Name:    types.StringValue(addr.Name),
		})
		respDiags.Append(diags...)
		data.Address = address
	} else {
		data.Address = types.ObjectNull(gatewayAddressModel{}.AttributeTypes())
	}

	var connections []string
	for _, conn := range gw.Connections {
		connections = append(connections, conn.Name)
	}
	data.Connections, diags = types.ListValueFrom(ctx, types.StringType, connections)
	respDiags.Append(diags...)

	addressModels := make([]gatewayAddressModel, len(gw.Addresses))
	for i, addr := range gw.Addresses {
		addressModels[i] = gatewayAddressModel{
			Address: types.StringValue(addr.Address),
			Name:    types.StringValue(addr.Name),
		}
	}
	data.Addresses, diags = types.SetValueFrom(ctx, types.ObjectType{AttrTypes: gatewayAddressModel{}.AttributeTypes()}, addressModels)
	respDiags.Append(diags...)

	return respDiags
}

func waitForGatewayToBeRunning(ctx context.Context, svc *service.Service, id string) (*upcloud.Gateway, error) {
	const maxRetries = 500

	for i := 0; i <= maxRetries; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			gw, err := svc.GetGateway(ctx, &request.GetGatewayRequest{UUID: id})
			if err != nil {
				return nil, err
			}
			if gw.OperationalState == upcloud.GatewayOperationalStateRunning {
				return gw, nil
			}
		}
		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("max retries (%d) reached while waiting for network gateway to be running", maxRetries)
}

func getGatewayDeleted(ctx context.Context, svc *service.Service, id ...string) (map[string]interface{}, error) {
	gw, err := svc.GetGateway(ctx, &request.GetGatewayRequest{UUID: id[0]})
	return map[string]interface{}{"resource": "gateway", "name": gw.Name, "state": gw.OperationalState}, err
}

func waitForGatewayToBeDeleted(ctx context.Context, svc *service.Service, id string) error {
	return utils.WaitForResourceToBeDeleted(ctx, svc, getGatewayDeleted, id)
}
