package database

import (
	"context"
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
	_ datasource.DataSource              = &databaseSessionsDataSource{}
	_ datasource.DataSourceWithConfigure = &databaseSessionsDataSource{}
)

func NewMySQLSessionsDataSource() datasource.DataSource {
	return &databaseSessionsDataSource{serviceType: upcloud.ManagedDatabaseServiceTypeMySQL}
}

func NewPostgreSQLSessionsDataSource() datasource.DataSource {
	return &databaseSessionsDataSource{serviceType: upcloud.ManagedDatabaseServiceTypePostgreSQL}
}

func NewValkeySessionsDataSource() datasource.DataSource {
	return &databaseSessionsDataSource{serviceType: upcloud.ManagedDatabaseServiceTypeValkey}
}

type databaseSessionsDataSource struct {
	client      *service.Service
	serviceType upcloud.ManagedDatabaseServiceType
}

type sessionsModel struct {
	ID       types.String `tfsdk:"id"`
	Limit    types.Int64  `tfsdk:"limit"`
	Offset   types.Int64  `tfsdk:"offset"`
	Order    types.String `tfsdk:"order"`
	Service  types.String `tfsdk:"service"`
	Sessions types.Set    `tfsdk:"sessions"`
}

func (d *databaseSessionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	switch d.serviceType {
	case upcloud.ManagedDatabaseServiceTypeMySQL:
		resp.TypeName = req.ProviderTypeName + "_managed_database_mysql_sessions"
	case upcloud.ManagedDatabaseServiceTypePostgreSQL:
		resp.TypeName = req.ProviderTypeName + "_managed_database_postgresql_sessions"
	case upcloud.ManagedDatabaseServiceTypeValkey:
		resp.TypeName = req.ProviderTypeName + "_managed_database_valkey_sessions"
	}
}

func (d *databaseSessionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client, resp.Diagnostics = utils.GetClientFromProviderData(req.ProviderData)
}

func (d *databaseSessionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Current sessions of a managed database",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"limit": schema.Int64Attribute{
				Description: "Number of entries to receive at most.",
				Optional:    true,
				Computed:    true,
			},
			"offset": schema.Int64Attribute{
				Description: "Offset for retrieved results based on sort order.",
				Optional:    true,
				Computed:    true,
			},
			"order": schema.StringAttribute{
				Description: "Order by session field and sort retrieved results. Limited variables can be used for ordering.",
				Optional:    true,
				Computed:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service's UUID for which these sessions belongs to",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"sessions": d.sessionsBlock(),
		},
	}
}

func (d *databaseSessionsDataSource) sessionsBlock() schema.SetNestedBlock {
	switch d.serviceType {
	case upcloud.ManagedDatabaseServiceTypeMySQL:
		return schema.SetNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"application_name": schema.StringAttribute{Computed: true, Description: "Name of the application that is connected to this service."},
					"client_addr":      schema.StringAttribute{Computed: true, Description: "IP address of the client connected to this service."},
					"datname":          schema.StringAttribute{Computed: true, Description: "Name of the database this service is connected to."},
					"id":               schema.StringAttribute{Computed: true, Description: "Process ID of this service."},
					"query":            schema.StringAttribute{Computed: true, Description: "Text of this service's most recent query."},
					"query_duration":   schema.StringAttribute{Computed: true, Description: "The active query current duration."},
					"state":            schema.StringAttribute{Computed: true, Description: "Current overall state of this service."},
					"usename":          schema.StringAttribute{Computed: true, Description: "Name of the user logged into this service."},
				},
			},
		}
	case upcloud.ManagedDatabaseServiceTypePostgreSQL:
		return schema.SetNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"application_name": schema.StringAttribute{Computed: true, Description: "Name of the application that is connected to this service."},
					"backend_start":    schema.StringAttribute{Computed: true, Description: "Time when this process was started."},
					"backend_type":     schema.StringAttribute{Computed: true, Description: "Type of current service."},
					"backend_xid":      schema.Int64Attribute{Computed: true, Description: "Top-level transaction identifier of this service, if any."},
					"backend_xmin":     schema.Int64Attribute{Computed: true, Description: "The current service's xmin horizon."},
					"client_addr":      schema.StringAttribute{Computed: true, Description: "IP address of the client connected to this service."},
					"client_hostname":  schema.StringAttribute{Computed: true, Description: "Host name of the connected client."},
					"client_port":      schema.Int64Attribute{Computed: true, Description: "TCP port number that the client is using."},
					"datid":            schema.Int64Attribute{Computed: true, Description: "OID of the database this service is connected to."},
					"datname":          schema.StringAttribute{Computed: true, Description: "Name of the database this service is connected to."},
					"id":               schema.StringAttribute{Computed: true, Description: "Process ID of this service."},
					"query":            schema.StringAttribute{Computed: true, Description: "Text of this service's most recent query."},
					"query_duration":   schema.StringAttribute{Computed: true, Description: "The active query current duration."},
					"query_start":      schema.StringAttribute{Computed: true, Description: "Time when the currently active query was started."},
					"state":            schema.StringAttribute{Computed: true, Description: "Current overall state of this service."},
					"state_change":     schema.StringAttribute{Computed: true, Description: "Time when the state was last changed."},
					"usename":          schema.StringAttribute{Computed: true, Description: "Name of the user logged into this service."},
					"usesysid":         schema.Int64Attribute{Computed: true, Description: "OID of the user logged into this service."},
					"wait_event":       schema.StringAttribute{Computed: true, Description: "Wait event name if service is currently waiting."},
					"wait_event_type":  schema.StringAttribute{Computed: true, Description: "The type of event for which the service is waiting, if any; otherwise NULL."},
					"xact_start":       schema.StringAttribute{Computed: true, Description: "Time when this process' current transaction was started, or null if no transaction is active."},
				},
			},
		}
	default:
		return schema.SetNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"active_channel_subscriptions":                  schema.Int64Attribute{Computed: true, Description: "Number of active channel subscriptions"},
					"active_database":                               schema.StringAttribute{Computed: true, Description: "Current database ID"},
					"active_pattern_matching_channel_subscriptions": schema.Int64Attribute{Computed: true, Description: "Number of pattern matching subscriptions."},
					"application_name":                              schema.StringAttribute{Computed: true, Description: "Name of the application that is connected to this service."},
					"client_addr":                                   schema.StringAttribute{Computed: true, Description: "Client address."},
					"connection_age":                                schema.Int64Attribute{Computed: true, Description: "Total duration of the connection in nanoseconds."},
					"connection_idle":                               schema.Int64Attribute{Computed: true, Description: "Idle time of the connection in nanoseconds."},
					"flags":                                         schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "A set containing flags' descriptions."},
					"flags_raw":                                     schema.StringAttribute{Computed: true, Description: "Client connection flags in raw string format."},
					"id":                                            schema.StringAttribute{Computed: true, Description: "Process ID of this session."},
					"multi_exec_commands":                           schema.Int64Attribute{Computed: true, Description: "Number of commands in a MULTI/EXEC context."},
					"output_buffer":                                 schema.Int64Attribute{Computed: true, Description: "Output buffer length."},
					"output_buffer_memory":                          schema.Int64Attribute{Computed: true, Description: "Output buffer memory usage."},
					"output_list_length":                            schema.Int64Attribute{Computed: true, Description: "Output list length."},
					"query":                                         schema.StringAttribute{Computed: true, Description: "The last executed command."},
					"query_buffer":                                  schema.Int64Attribute{Computed: true, Description: "Query buffer length (0 means no query pending)."},
					"query_buffer_free":                             schema.Int64Attribute{Computed: true, Description: "Free space of the query buffer."},
				},
			},
		}
	}
}

func (d *databaseSessionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sessionsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := data.Service.ValueString()

	db, err := d.client.GetManagedDatabase(ctx, &request.GetManagedDatabaseRequest{UUID: serviceID})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read managed database",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}
	if db.Type != d.serviceType {
		resp.Diagnostics.AddError(
			"Invalid database type",
			"Getting sessions for Managed Database "+serviceID+" failed: database type "+string(db.Type)+" is not valid for this data source",
		)
		return
	}

	limit := int(data.Limit.ValueInt64())
	if data.Limit.IsNull() {
		limit = 10
	}
	offset := int(data.Offset.ValueInt64())
	if data.Offset.IsNull() {
		offset = 0
	}
	order := data.Order.ValueString()
	if data.Order.IsNull() {
		order = "query_duration:desc"
	}

	sessions, err := d.client.GetManagedDatabaseSessions(ctx, &request.GetManagedDatabaseSessionsRequest{
		UUID:   serviceID,
		Limit:  limit,
		Offset: offset,
		Order:  order,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read database sessions",
			utils.ErrorDiagnosticDetail(err),
		)
		return
	}

	data.ID = types.StringValue(serviceID)
	data.Service = types.StringValue(serviceID)
	data.Limit = types.Int64Value(int64(limit))
	data.Offset = types.Int64Value(int64(offset))
	data.Order = types.StringValue(order)

	var diags diag.Diagnostics
	switch d.serviceType {
	case upcloud.ManagedDatabaseServiceTypeMySQL:
		data.Sessions, diags = types.SetValueFrom(ctx, data.Sessions.ElementType(ctx), buildSessionsFrameworkMySQL(sessions.MySQL))
	case upcloud.ManagedDatabaseServiceTypePostgreSQL:
		data.Sessions, diags = types.SetValueFrom(ctx, data.Sessions.ElementType(ctx), buildSessionsFrameworkPostgreSQL(sessions.PostgreSQL))
	case upcloud.ManagedDatabaseServiceTypeValkey:
		data.Sessions, diags = types.SetValueFrom(ctx, data.Sessions.ElementType(ctx), buildSessionsFrameworkValkey(sessions.Valkey))
	}
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type sessionMySQLModel struct {
	ApplicationName types.String `tfsdk:"application_name"`
	ClientAddr      types.String `tfsdk:"client_addr"`
	Datname         types.String `tfsdk:"datname"`
	ID              types.String `tfsdk:"id"`
	Query           types.String `tfsdk:"query"`
	QueryDuration   types.String `tfsdk:"query_duration"`
	State           types.String `tfsdk:"state"`
	Usename         types.String `tfsdk:"usename"`
}

type sessionPostgreSQLModel struct {
	ApplicationName types.String `tfsdk:"application_name"`
	BackendStart    types.String `tfsdk:"backend_start"`
	BackendType     types.String `tfsdk:"backend_type"`
	BackendXid      types.Int64  `tfsdk:"backend_xid"`
	BackendXmin     types.Int64  `tfsdk:"backend_xmin"`
	ClientAddr      types.String `tfsdk:"client_addr"`
	ClientHostname  types.String `tfsdk:"client_hostname"`
	ClientPort      types.Int64  `tfsdk:"client_port"`
	Datid           types.Int64  `tfsdk:"datid"`
	Datname         types.String `tfsdk:"datname"`
	ID              types.String `tfsdk:"id"`
	Query           types.String `tfsdk:"query"`
	QueryDuration   types.String `tfsdk:"query_duration"`
	QueryStart      types.String `tfsdk:"query_start"`
	State           types.String `tfsdk:"state"`
	StateChange     types.String `tfsdk:"state_change"`
	Usename         types.String `tfsdk:"usename"`
	Usesysid        types.Int64  `tfsdk:"usesysid"`
	WaitEvent       types.String `tfsdk:"wait_event"`
	WaitEventType   types.String `tfsdk:"wait_event_type"`
	XactStart       types.String `tfsdk:"xact_start"`
}

type sessionValkeyModel struct {
	ActiveChannelSubscriptions                types.Int64  `tfsdk:"active_channel_subscriptions"`
	ActiveDatabase                            types.String `tfsdk:"active_database"`
	ActivePatternMatchingChannelSubscriptions types.Int64  `tfsdk:"active_pattern_matching_channel_subscriptions"`
	ApplicationName                           types.String `tfsdk:"application_name"`
	ClientAddr                                types.String `tfsdk:"client_addr"`
	ConnectionAge                             types.Int64  `tfsdk:"connection_age"`
	ConnectionIdle                            types.Int64  `tfsdk:"connection_idle"`
	Flags                                     types.Set    `tfsdk:"flags"`
	FlagsRaw                                  types.String `tfsdk:"flags_raw"`
	ID                                        types.String `tfsdk:"id"`
	MultiExecCommands                         types.Int64  `tfsdk:"multi_exec_commands"`
	OutputBuffer                              types.Int64  `tfsdk:"output_buffer"`
	OutputBufferMemory                        types.Int64  `tfsdk:"output_buffer_memory"`
	OutputListLength                          types.Int64  `tfsdk:"output_list_length"`
	Query                                     types.String `tfsdk:"query"`
	QueryBuffer                               types.Int64  `tfsdk:"query_buffer"`
	QueryBufferFree                           types.Int64  `tfsdk:"query_buffer_free"`
}

func intPointerToInt64(v *int) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

func stringPointerToString(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func timePointerToString(v *time.Time) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(v.UTC().Format(time.RFC3339Nano))
}

func buildSessionsFrameworkMySQL(sessions []upcloud.ManagedDatabaseSessionMySQL) []sessionMySQLModel {
	res := make([]sessionMySQLModel, 0, len(sessions))
	for _, s := range sessions {
		res = append(res, sessionMySQLModel{
			ApplicationName: types.StringValue(s.ApplicationName),
			ClientAddr:      types.StringValue(s.ClientAddr),
			Datname:         types.StringValue(s.Datname),
			ID:              types.StringValue(s.Id),
			Query:           types.StringValue(s.Query),
			QueryDuration:   types.StringValue(s.QueryDuration.String()),
			State:           types.StringValue(s.State),
			Usename:         types.StringValue(s.Usename),
		})
	}
	return res
}

func buildSessionsFrameworkPostgreSQL(sessions []upcloud.ManagedDatabaseSessionPostgreSQL) []sessionPostgreSQLModel {
	res := make([]sessionPostgreSQLModel, 0, len(sessions))
	for _, s := range sessions {
		res = append(res, sessionPostgreSQLModel{
			ApplicationName: types.StringValue(s.ApplicationName),
			BackendStart:    types.StringValue(s.BackendStart.UTC().Format(time.RFC3339Nano)),
			BackendType:     types.StringValue(s.BackendType),
			BackendXid:      intPointerToInt64(s.BackendXid),
			BackendXmin:     intPointerToInt64(s.BackendXmin),
			ClientAddr:      types.StringValue(s.ClientAddr),
			ClientHostname:  stringPointerToString(s.ClientHostname),
			ClientPort:      types.Int64Value(int64(s.ClientPort)),
			Datid:           types.Int64Value(int64(s.Datid)),
			Datname:         types.StringValue(s.Datname),
			ID:              types.StringValue(s.Id),
			Query:           types.StringValue(s.Query),
			QueryDuration:   types.StringValue(s.QueryDuration.String()),
			QueryStart:      types.StringValue(s.QueryStart.UTC().Format(time.RFC3339Nano)),
			State:           types.StringValue(s.State),
			StateChange:     types.StringValue(s.StateChange.UTC().Format(time.RFC3339Nano)),
			Usename:         types.StringValue(s.Usename),
			Usesysid:        types.Int64Value(int64(s.Usesysid)),
			WaitEvent:       types.StringValue(s.WaitEvent),
			WaitEventType:   types.StringValue(s.WaitEventType),
			XactStart:       timePointerToString(s.XactStart),
		})
	}
	return res
}

func buildSessionsFrameworkValkey(sessions []upcloud.ManagedDatabaseSessionValkey) []sessionValkeyModel {
	res := make([]sessionValkeyModel, 0, len(sessions))
	for _, s := range sessions {
		res = append(res, sessionValkeyModel{
			ActiveChannelSubscriptions:                types.Int64Value(int64(s.ActiveChannelSubscriptions)),
			ActiveDatabase:                            types.StringValue(s.ActiveDatabase),
			ActivePatternMatchingChannelSubscriptions: types.Int64Value(int64(s.ActivePatternMatchingChannelSubscriptions)),
			ApplicationName:                           types.StringValue(s.ApplicationName),
			ClientAddr:                                types.StringValue(s.ClientAddr),
			ConnectionAge:                             types.Int64Value(s.ConnectionAge.Nanoseconds()),
			ConnectionIdle:                            types.Int64Value(s.ConnectionIdle.Nanoseconds()),
			FlagsRaw:                                  types.StringValue(s.FlagsRaw),
			ID:                                        types.StringValue(s.Id),
			MultiExecCommands:                         types.Int64Value(int64(s.MultiExecCommands)),
			OutputBuffer:                              types.Int64Value(int64(s.OutputBuffer)),
			OutputBufferMemory:                        types.Int64Value(int64(s.OutputBufferMemory)),
			OutputListLength:                          types.Int64Value(int64(s.OutputListLength)),
			Query:                                     types.StringValue(s.Query),
			QueryBuffer:                               types.Int64Value(int64(s.QueryBuffer)),
			QueryBufferFree:                           types.Int64Value(int64(s.QueryBufferFree)),
		})

		flags, diags := types.SetValueFrom(context.Background(), types.StringType, s.Flags)
		if diags.HasError() {
			return res
		}
		res[len(res)-1].Flags = flags
	}
	return res
}
