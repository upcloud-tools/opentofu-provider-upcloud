package account

import (
	"context"

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

func NewAccountDataSource() datasource.DataSource {
	return &accountDataSource{}
}

var (
	_ datasource.DataSource              = &accountDataSource{}
	_ datasource.DataSourceWithConfigure = &accountDataSource{}
)

type accountDataSource struct {
	client *service.Service
}

func (d *accountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *accountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

type accountDataSourceModel struct {
	ID             types.String  `tfsdk:"id"`
	Username       types.String  `tfsdk:"username"`
	Credits        types.Float64 `tfsdk:"credits"`
	ResourceLimits types.Object  `tfsdk:"resource_limits"`
	AccountDetails types.Object  `tfsdk:"account_details"`
}

type resourceLimitsModel struct {
	Cores                 types.Int64 `tfsdk:"cores"`
	DetachedFloatingIPs   types.Int64 `tfsdk:"detached_floating_ips"`
	ManagedObjectStorages types.Int64 `tfsdk:"managed_object_storages"`
	MemoryMB              types.Int64 `tfsdk:"memory_mb"`
	NetworkPeerings       types.Int64 `tfsdk:"network_peerings"`
	Networks              types.Int64 `tfsdk:"networks"`
	PublicIPv4            types.Int64 `tfsdk:"public_ipv4"`
	PublicIPv6            types.Int64 `tfsdk:"public_ipv6"`
	StorageHDD            types.Int64 `tfsdk:"storage_hdd"`
	StorageMaxIOPS        types.Int64 `tfsdk:"storage_maxiops"`
	StorageSSD            types.Int64 `tfsdk:"storage_ssd"`
	LoadBalancers         types.Int64 `tfsdk:"load_balancers"`
	GPUs                  types.Int64 `tfsdk:"gpus"`
}

func resourceLimitsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cores":                   types.Int64Type,
		"detached_floating_ips":   types.Int64Type,
		"managed_object_storages": types.Int64Type,
		"memory_mb":               types.Int64Type,
		"network_peerings":        types.Int64Type,
		"networks":                types.Int64Type,
		"public_ipv4":             types.Int64Type,
		"public_ipv6":             types.Int64Type,
		"storage_hdd":             types.Int64Type,
		"storage_maxiops":         types.Int64Type,
		"storage_ssd":             types.Int64Type,
		"load_balancers":          types.Int64Type,
		"gpus":                    types.Int64Type,
	}
}

type accountDetailsModel struct {
	MainAccount  types.String `tfsdk:"main_account"`
	Type         types.String `tfsdk:"type"`
	FirstName    types.String `tfsdk:"first_name"`
	LastName     types.String `tfsdk:"last_name"`
	Company      types.String `tfsdk:"company"`
	Address      types.String `tfsdk:"address"`
	PostalCode   types.String `tfsdk:"postal_code"`
	City         types.String `tfsdk:"city"`
	Email        types.String `tfsdk:"email"`
	Phone        types.String `tfsdk:"phone"`
	State        types.String `tfsdk:"state"`
	Country      types.String `tfsdk:"country"`
	Currency     types.String `tfsdk:"currency"`
	Language     types.String `tfsdk:"language"`
	VATNumber    types.String `tfsdk:"vat_number"`
	Timezone     types.String `tfsdk:"timezone"`
	AllowAPI     types.Bool   `tfsdk:"allow_api"`
	AllowGUI     types.Bool   `tfsdk:"allow_gui"`
	SimpleBackup types.Bool   `tfsdk:"simple_backup"`
}

func accountDetailsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"main_account":  types.StringType,
		"type":          types.StringType,
		"first_name":    types.StringType,
		"last_name":     types.StringType,
		"company":       types.StringType,
		"address":       types.StringType,
		"postal_code":   types.StringType,
		"city":          types.StringType,
		"email":         types.StringType,
		"phone":         types.StringType,
		"state":         types.StringType,
		"country":       types.StringType,
		"currency":      types.StringType,
		"language":      types.StringType,
		"vat_number":    types.StringType,
		"timezone":      types.StringType,
		"allow_api":     types.BoolType,
		"allow_gui":     types.BoolType,
		"simple_backup": types.BoolType,
	}
}

const accountDataSourceDescription = `Provides details of the UpCloud account associated with the provider credentials.

Use this data source to retrieve account information, including credits, resource limits, and account details.`

func (d *accountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: accountDataSourceDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The username of the account.",
			},
			"credits": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "The available credits on the account.",
			},
			"resource_limits": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The resource limits for the account.",
				Attributes: map[string]schema.Attribute{
					"cores":                   schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum number of CPU cores."},
					"detached_floating_ips":   schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum number of detached floating IPs."},
					"managed_object_storages": schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum number of managed object storages."},
					"memory_mb":               schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum amount of memory in MB."},
					"network_peerings":        schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum number of network peerings."},
					"networks":                schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum number of networks."},
					"public_ipv4":             schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum number of public IPv4 addresses."},
					"public_ipv6":             schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum number of public IPv6 addresses."},
					"storage_hdd":             schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum amount of HDD storage in GB."},
					"storage_maxiops":         schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum amount of MaxIOPS storage in GB."},
					"storage_ssd":             schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum amount of SSD storage in GB."},
					"load_balancers":          schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum number of load balancers."},
					"gpus":                    schema.Int64Attribute{Computed: true, MarkdownDescription: "Maximum number of GPUs."},
				},
			},
			"account_details": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Detailed account information including contact and billing details.",
				Attributes: map[string]schema.Attribute{
					"main_account":  schema.StringAttribute{Computed: true, MarkdownDescription: "The main account username."},
					"type":          schema.StringAttribute{Computed: true, MarkdownDescription: "The account type (main or sub)."},
					"first_name":    schema.StringAttribute{Computed: true, MarkdownDescription: "First name of the account holder."},
					"last_name":     schema.StringAttribute{Computed: true, MarkdownDescription: "Last name of the account holder."},
					"company":       schema.StringAttribute{Computed: true, MarkdownDescription: "Company name."},
					"address":       schema.StringAttribute{Computed: true, MarkdownDescription: "Street address."},
					"postal_code":   schema.StringAttribute{Computed: true, MarkdownDescription: "Postal code."},
					"city":          schema.StringAttribute{Computed: true, MarkdownDescription: "City."},
					"email":         schema.StringAttribute{Computed: true, MarkdownDescription: "Email address."},
					"phone":         schema.StringAttribute{Computed: true, MarkdownDescription: "Phone number."},
					"state":         schema.StringAttribute{Computed: true, MarkdownDescription: "State or province."},
					"country":       schema.StringAttribute{Computed: true, MarkdownDescription: "ISO 3166-1 three character country code."},
					"currency":      schema.StringAttribute{Computed: true, MarkdownDescription: "Account currency."},
					"language":      schema.StringAttribute{Computed: true, MarkdownDescription: "Account language."},
					"vat_number":    schema.StringAttribute{Computed: true, MarkdownDescription: "VAT number."},
					"timezone":      schema.StringAttribute{Computed: true, MarkdownDescription: "Account timezone."},
					"allow_api":     schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether API access is allowed."},
					"allow_gui":     schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether GUI access is allowed."},
					"simple_backup": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether simple backups are enabled."},
				},
			},
		},
	}
}

func (d *accountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data accountDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	acc, err := d.client.GetAccount(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read account",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	accDetails, err := d.client.GetAccountDetails(ctx, &request.GetAccountDetailsRequest{
		Username: acc.UserName,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read account details",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	setAccountValues(ctx, &data, acc, accDetails, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setAccountValues(ctx context.Context, data *accountDataSourceModel, acc *upcloud.Account, details *upcloud.AccountDetails, diags *diag.Diagnostics) {
	data.ID = types.StringValue(acc.UserName)
	data.Username = types.StringValue(acc.UserName)
	data.Credits = types.Float64Value(acc.Credits)

	rl := resourceLimitsModel{
		Cores:                 types.Int64Value(int64(acc.ResourceLimits.Cores)),
		DetachedFloatingIPs:   types.Int64Value(int64(acc.ResourceLimits.DetachedFloatingIps)),
		ManagedObjectStorages: types.Int64Value(int64(acc.ResourceLimits.ManagedObjectStorages)),
		MemoryMB:              types.Int64Value(int64(acc.ResourceLimits.Memory)),
		NetworkPeerings:       types.Int64Value(int64(acc.ResourceLimits.NetworkPeerings)),
		Networks:              types.Int64Value(int64(acc.ResourceLimits.Networks)),
		PublicIPv4:            types.Int64Value(int64(acc.ResourceLimits.PublicIPv4)),
		PublicIPv6:            types.Int64Value(int64(acc.ResourceLimits.PublicIPv6)),
		StorageHDD:            types.Int64Value(int64(acc.ResourceLimits.StorageHDD)),
		StorageMaxIOPS:        types.Int64Value(int64(acc.ResourceLimits.StorageMaxIOPS)),
		StorageSSD:            types.Int64Value(int64(acc.ResourceLimits.StorageSSD)),
		LoadBalancers:         types.Int64Value(int64(acc.ResourceLimits.LoadBalancers)),
		GPUs:                  types.Int64Value(int64(acc.ResourceLimits.GPUs)),
	}

	rlValue, d := types.ObjectValueFrom(ctx, resourceLimitsAttrTypes(), rl)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	data.ResourceLimits = rlValue

	ad := accountDetailsModel{
		MainAccount:  types.StringValue(details.MainAccount),
		Type:         types.StringValue(string(details.Type)),
		FirstName:    types.StringValue(details.FirstName),
		LastName:     types.StringValue(details.LastName),
		Company:      types.StringValue(details.Company),
		Address:      types.StringValue(details.Address),
		PostalCode:   types.StringValue(details.PostalCode),
		City:         types.StringValue(details.City),
		Email:        types.StringValue(details.Email),
		Phone:        types.StringValue(details.Phone),
		State:        types.StringValue(details.State),
		Country:      types.StringValue(details.Country),
		Currency:     types.StringValue(details.Currency),
		Language:     types.StringValue(details.Language),
		VATNumber:    types.StringValue(details.VATNnumber),
		Timezone:     types.StringValue(details.Timezone),
		AllowAPI:     types.BoolValue(details.AllowAPI.Bool()),
		AllowGUI:     types.BoolValue(details.AllowGUI.Bool()),
		SimpleBackup: types.BoolValue(details.SimpleBackup.Bool()),
	}

	adValue, d := types.ObjectValueFrom(ctx, accountDetailsAttrTypes(), ad)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	data.AccountDetails = adValue
}
