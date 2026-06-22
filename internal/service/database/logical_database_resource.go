package database

import (
	"context"
	"fmt"
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
	_ resource.Resource                = &logicalDatabaseResource{}
	_ resource.ResourceWithConfigure   = &logicalDatabaseResource{}
	_ resource.ResourceWithImportState = &logicalDatabaseResource{}
)

func NewLogicalDatabaseResource() resource.Resource {
	return &logicalDatabaseResource{}
}

type logicalDatabaseResource struct {
	client *service.Service
}

type logicalDatabaseModel struct {
	ID           types.String `tfsdk:"id"`
	Service      types.String `tfsdk:"service"`
	Name         types.String `tfsdk:"name"`
	CharacterSet types.String `tfsdk:"character_set"`
	Collation    types.String `tfsdk:"collation"`
}

func (r *logicalDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_database_logical_database"
}

func (r *logicalDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

func (r *logicalDatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource represents a logical database in managed database",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the logical database in {service UUID}/{name} format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				Description: "Service's UUID for which this database belongs to",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the logical database",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"character_set": schema.StringAttribute{
				Description: "Default character set for the database (LC_CTYPE)",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z]{2}_[A-Z]{2}\.[A-z0-9-]+$`),
						"invalid locale; must be in form en_US.UTF8 (language_TERRITORY.CODEPOINT)",
					),
				},
			},
			"collation": schema.StringAttribute{
				Description: "Default collation for the database (LC_COLLATE)",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z]{2}_[A-Z]{2}\.[A-z0-9-]+$`),
						"invalid locale; must be in form en_US.UTF8 (language_TERRITORY.CODEPOINT)",
					),
				},
			},
		},
	}
}

func (r *logicalDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data logicalDatabaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := data.Service.ValueString()
	serviceDetails, err := r.client.GetManagedDatabase(ctx, &request.GetManagedDatabaseRequest{UUID: serviceID})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read managed database",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}
	if !serviceDetails.Powered {
		resp.Diagnostics.AddError(
			"Unable to create logical database",
			fmt.Sprintf("cannot create a logical database while managed database %v (%v) is powered off", serviceDetails.Name, serviceID),
		)
		return
	}

	characterSet := data.CharacterSet.ValueString()
	collation := data.Collation.ValueString()
	if (characterSet != "" || collation != "") && serviceDetails.Type != upcloud.ManagedDatabaseServiceTypePostgreSQL {
		resp.Diagnostics.AddError(
			"Unable to create logical database",
			"setting character_set or collation is only possible for PostgreSQL service",
		)
		return
	}

	_, err = r.client.WaitForManagedDatabaseState(ctx, &request.WaitForManagedDatabaseStateRequest{
		UUID:         serviceID,
		DesiredState: upcloud.ManagedDatabaseStateRunning,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create logical database",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	_, err = r.client.CreateManagedDatabaseLogicalDatabase(ctx, &request.CreateManagedDatabaseLogicalDatabaseRequest{
		ServiceUUID: serviceID,
		Name:        data.Name.ValueString(),
		LCCType:     characterSet,
		LCCollate:   collation,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create logical database",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(utils.MarshalID(serviceID, data.Name.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *logicalDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data logicalDatabaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var serviceID, name string
	if err := utils.UnmarshalID(data.ID.ValueString(), &serviceID, &name); err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse logical database ID",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	_, err := r.client.GetManagedDatabase(ctx, &request.GetManagedDatabaseRequest{UUID: serviceID})
	if err != nil {
		if utils.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to read managed database",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	ldbs, err := r.client.GetManagedDatabaseLogicalDatabases(ctx, &request.GetManagedDatabaseLogicalDatabasesRequest{
		ServiceUUID: serviceID,
	})
	if err != nil {
		if utils.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to list logical databases",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	var details *upcloud.ManagedDatabaseLogicalDatabase
	for i, ldb := range ldbs {
		if ldb.Name == name {
			details = &ldbs[i]
			break
		}
	}

	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Service = types.StringValue(serviceID)
	data.Name = types.StringValue(details.Name)
	data.CharacterSet = types.StringValue(details.LCCType)
	data.Collation = types.StringValue(details.LCCollate)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *logicalDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data logicalDatabaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var serviceID, name string
	if err := utils.UnmarshalID(data.ID.ValueString(), &serviceID, &name); err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse logical database ID",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	serviceDetails, err := r.client.GetManagedDatabase(ctx, &request.GetManagedDatabaseRequest{UUID: serviceID})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read managed database",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}
	if !serviceDetails.Powered {
		resp.Diagnostics.AddError(
			"Unable to delete logical database",
			fmt.Sprintf("cannot delete a logical database while managed database %v (%v) is powered off", serviceDetails.Name, serviceID),
		)
		return
	}

	_, err = r.client.WaitForManagedDatabaseState(ctx, &request.WaitForManagedDatabaseStateRequest{
		UUID:         serviceID,
		DesiredState: upcloud.ManagedDatabaseStateRunning,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete logical database",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	err = r.client.DeleteManagedDatabaseLogicalDatabase(ctx, &request.DeleteManagedDatabaseLogicalDatabaseRequest{
		ServiceUUID: serviceID,
		Name:        name,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete logical database",
			utils.ErrorDiagnosticDetail(err),
		)
	}
}

func (r *logicalDatabaseResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}

func (r *logicalDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
