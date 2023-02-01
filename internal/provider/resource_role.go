package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jackc/pgx/v5"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &resourceRole{}
	_ resource.ResourceWithConfigure   = &resourceRole{}
	_ resource.ResourceWithImportState = &resourceRole{}
)

func NewRoleResource() resource.Resource {
	return &resourceRole{}
}

type resourceRole struct {
	p *cockroachdbProvider
}

func (r *resourceRole) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "cockroachdb_role"
}

func (r *resourceRole) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a role.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the role. Must be unique on the CockroachDb server instance where it is configured.",
				Required:    true,
			},
			"password": schema.StringAttribute{
				Description: "Sets the role's password. Setting a password creates a user that's able to log in",
				Optional:    true,
			},
			"create_database": schema.BoolAttribute{
				Description: "Defines a role's ability to execute CREATE DATABASE. Default value is false.",
				Optional:    true,
				// PlanModifiers: []planmodifier.Bool{
				// 	modifiers.BoolDefault(false),
				// },
			},
			"create_role": schema.BoolAttribute{
				Description: "Defines a role's ability to execute CREATE ROLE. A role with this privilege can also alter and drop other roles. Default value is false.",
				Optional:    true,
				// PlanModifiers: []planmodifier.Bool{
				// 	modifiers.BoolDefault(false),
				// },
			},
			"login": schema.BoolAttribute{
				Description: "Defines whether role is allowed to log in. Roles without this attribute are useful for managing database privileges, but are not users in the usual sense of the word. Default value is false.",
				Optional:    true,
				// PlanModifiers: []planmodifier.Bool{
				// 	modifiers.BoolDefault(false),
				// },
			},
			"id": schema.StringAttribute{
				Description: "ID of the role (it's really just the name because they have to be unique)",
				Computed:    true,
			},
		},
	}
}

func (r *resourceRole) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.p = req.ProviderData.(*cockroachdbProvider)
}

// Create a new resource
func (r *resourceRole) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Role

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to db
	conn, err := r.p.Conn(ctx, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	// Build query
	// Why Go, why don't you have ternaries :'(
	createRoleQuery := fmt.Sprintf(`CREATE ROLE "%s" WITH`, plan.Name.ValueString())
	if !plan.CreateDatabase.ValueBool() || plan.CreateDatabase.IsNull() { // Default false
		createRoleQuery += " NOCREATEDB"
	} else {
		createRoleQuery += " CREATEDB"
	}
	if !plan.CreateRole.ValueBool() || plan.CreateRole.IsNull() { // Default true
		createRoleQuery += " NOCREATEROLE"
	} else {
		createRoleQuery += " CREATEROLE"
	}
	if !plan.Login.ValueBool() || plan.Login.IsNull() { // Default false
		createRoleQuery += " NOLOGIN"
	} else {
		createRoleQuery += " LOGIN"
	}
	if plan.Password.ValueString() != "" {
		createRoleQuery = fmt.Sprintf(`%s PASSWORD '%s'`, createRoleQuery, plan.Password.ValueString())
	}

	tflog.Info(ctx, createRoleQuery)

	// Run query
	_, err = conn.Exec(ctx, createRoleQuery)

	// Error handling
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach execute sql error",
			err.Error(),
		)
		return
	}

	// Grab the ID that was created for the role
	newRole, err := GetRoleByKeyValue(conn, ctx, "rolname", plan.Name.ValueString())

	// Error handling
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach execute sql error",
			err.Error(),
		)
		return
	}

	// Set the ID of the role
	plan.ID = types.StringValue(newRole["name"].(string))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func GetRoleByKeyValue(conn *pgx.Conn, ctx context.Context, searchKey string, searchValue string) (map[string]interface{}, error) {
	var (
		rolname       string
		rolcreaterole bool
		rolcreatedb   bool
		rolcanlogin   bool
	)
	selectQuery := fmt.Sprintf(`SELECT rolname, rolcreaterole, rolcreatedb, rolcanlogin FROM pg_roles WHERE %s = '%s'`, searchKey, searchValue)
	tflog.Info(ctx, selectQuery)
	err := conn.QueryRow(ctx, selectQuery).Scan(
		&rolname,
		&rolcreaterole,
		&rolcreatedb,
		&rolcanlogin,
	)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name":            rolname,
		"create_role":     rolcreaterole,
		"create_database": rolcreatedb,
		"login":           rolcanlogin,
	}, nil
}

// Read resource information
func (r *resourceRole) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Role

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to db
	conn, err := r.p.Conn(ctx, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	roleRow, err := GetRoleByKeyValue(conn, ctx, "rolname", state.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach execute sql error",
			err.Error(),
		)
		return
	}

	// Set the state object
	state.ID = types.StringValue(roleRow["name"].(string))
	state.Name = types.StringValue(roleRow["name"].(string))
	if roleRow["create_database"].(bool) {
		state.CreateDatabase = types.BoolValue(true)
	}
	if roleRow["create_role"].(bool) {
		state.CreateRole = types.BoolValue(true)
	}
	if roleRow["login"].(bool) {
		state.Login = types.BoolValue(true)
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update resource
func (r *resourceRole) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var (
		state Role
		plan  Role
	)

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to db
	conn, err := r.p.Conn(ctx, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	if state.Name.ValueString() != plan.Name.ValueString() {
		resp.Diagnostics.AddError(
			"Terraform config error",
			"Role name cannot be updated in cockroachdb, create new roles",
		)
		return
	}

	// Build query
	// Why Go, why don't you have ternaries :'(
	alterRoleQuery := fmt.Sprintf(`ALTER ROLE "%s" WITH`, plan.Name.ValueString())
	if !plan.CreateDatabase.ValueBool() || plan.CreateDatabase.IsNull() { // Default false
		alterRoleQuery += " NOCREATEDB"
	} else {
		alterRoleQuery += " CREATEDB"
	}
	if !plan.CreateRole.ValueBool() || plan.CreateRole.IsNull() { // Default true
		alterRoleQuery += " NOCREATEROLE"
	} else {
		alterRoleQuery += " CREATEROLE"
	}
	if !plan.Login.ValueBool() || plan.Login.IsNull() { // Default false
		alterRoleQuery += " NOLOGIN"
	} else {
		alterRoleQuery += " LOGIN"
	}
	if plan.Password.ValueString() == "" || plan.Password.IsNull() { // Default false
		alterRoleQuery = fmt.Sprintf(`%s PASSWORD %s`, alterRoleQuery, "null")
	} else {
		alterRoleQuery = fmt.Sprintf(`%s PASSWORD '%s'`, alterRoleQuery, plan.Password.ValueString())
	}

	tflog.Info(ctx, alterRoleQuery)

	// Run query
	_, err = conn.Exec(ctx, alterRoleQuery)

	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach execute sql error",
			err.Error(),
		)
		return
	}

	// Set the state object
	state.ID = plan.Name
	state.Name = plan.Name
	state.CreateDatabase = plan.CreateDatabase
	state.CreateRole = plan.CreateRole
	state.Login = plan.Login
	state.Password = plan.Password

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r *resourceRole) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Role

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to db
	conn, err := r.p.Conn(ctx, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	_, err = conn.Exec(ctx, fmt.Sprintf(`DROP ROLE "%s"`, state.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach execute sql error",
			err.Error(),
		)
		return
	}

	// Set id to -1
	state.ID = types.StringValue("")

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceRole) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type Role struct {
	Name           types.String `tfsdk:"name"`
	Password       types.String `tfsdk:"password"`
	CreateDatabase types.Bool   `tfsdk:"create_database"`
	CreateRole     types.Bool   `tfsdk:"create_role"`
	Login          types.Bool   `tfsdk:"login"`
	ID             types.String `tfsdk:"id"`
}
