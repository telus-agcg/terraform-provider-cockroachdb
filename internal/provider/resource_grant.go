package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"telusag/terraform-provider-cockroachdb/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &resourceGrant{}
	_ resource.ResourceWithConfigure   = &resourceGrant{}
	_ resource.ResourceWithImportState = &resourceGrant{}
)

func NewGrantResource() resource.Resource {
	return &resourceGrant{}
}

type resourceGrant struct {
	p *cockroachdbProvider
}

func (r *resourceGrant) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "cockroachdb_grant"
}

func (r *resourceGrant) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a grant.",
		Attributes: map[string]schema.Attribute{
			"role": schema.StringAttribute{
				Description: "Target role name",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"database": schema.StringAttribute{
				Description: "Target database name",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"schema": schema.StringAttribute{
				Description: "Target schema name",
				Optional:    true,
			},
			"object_type": schema.StringAttribute{
				Description: "Object type. Must be one of the following:  database, schema, or table",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(regexp.MustCompile(`^(database|schema|table)$`), "Value must match RegExp: ^(database|schema|table)$"),
				},
			},
			"objects": schema.ListAttribute{
				Description: "Objects to grant privileges on",
				Optional:    true,
				ElementType: types.StringType,
			},
			"privileges": schema.ListAttribute{
				Description: "Privileges to grant",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Description: "ID of the grant",
				Computed:    true,
			},
		},
	}
}

func readRolePrivileges(ctx context.Context, conn *pgx.Conn, grant *Grant) error {
	dbName := grant.Database.ValueString()
	role := grant.Role.ValueString()
	objectType := grant.ObjectType.ValueString()
	objects := []string{}
	privileges := []string{}

	var (
		err  error
		rows pgx.Rows
	)

	rows, err = conn.Query(ctx, fmt.Sprintf(`SHOW GRANTS FOR %s;`, role))
	if err != nil {
		return err
	}

	// Loop through each returned row
	for rows.Next() {
		var (
			database_name  pgtype.Text
			schema_name    pgtype.Text
			relation_name  pgtype.Text
			grantee        pgtype.Text
			privilege_type pgtype.Text
			is_grantable   pgtype.Bool
		)

		// Map row vals
		if err := rows.Scan(&database_name, &schema_name, &relation_name, &grantee, &privilege_type, &is_grantable); err != nil {
			tflog.Error(ctx, err.Error())
			continue
		}

		// See if this row pertains to the state
		switch strings.ToUpper(objectType) {
		case "DATABASE":
			// If schema_name is not NULL or relation_name is not NULL it's not a database privilege
			if !utils.IsNilOrEmpty(&schema_name.String) || !utils.IsNilOrEmpty(&relation_name.String) || database_name.String != dbName {
				continue
			}
		case "SCHEMA":
			// If schema_name is NULL or relation_name is not NULL it's not a schema privilege
			if utils.IsNilOrEmpty(&schema_name.String) || !utils.IsNilOrEmpty(&relation_name.String) {
				continue
			}

			// Set schema
			grant.Schema = types.StringValue(schema_name.String)
		case "TABLE":
			// If schema_name is NULL or relation_name is NULL it's not a table privilege
			if utils.IsNilOrEmpty(&schema_name.String) || utils.IsNilOrEmpty(&relation_name.String) {
				continue
			}

			// Set schema
			grant.Schema = types.StringValue(schema_name.String)
		}

		// Set privileges and objects
		if privilege_type.String != "" {
			privileges = append(privileges, privilege_type.String)
		}
		if relation_name.String != "" {
			objects = append(objects, relation_name.String)
		}
	}

	// Set privileges on grant
	var privilegesVals []attr.Value
	for _, privilege := range privileges {
		privilegesVals = append(privilegesVals, types.StringValue(privilege))
	}
	grant.Privileges, _ = types.ListValue(types.StringType, privilegesVals)

	// Set objects on grant only if they exist
	if len(objects) >= 1 {
		var objectsVals []attr.Value
		for _, object := range objects {
			objectsVals = append(objectsVals, types.StringValue(object))
		}
		grant.Objects, _ = types.ListValue(types.StringType, objectsVals)
	}

	return nil
}

func grantRolePrivileges(ctx context.Context, conn *pgx.Conn, grant *Grant) error {
	var err error

	query := getGrantQuery(ctx, grant)

	tflog.Info(ctx, query)

	_, err = conn.Exec(ctx, query)
	if err != nil {
		return err
	}

	// Generate "ID" from grant
	grant.ID = types.StringValue(grant.Role.ValueString() + "|" + grant.Database.ValueString() + "|" + grant.ObjectType.ValueString())

	return nil
}

func getGrantQuery(ctx context.Context, grant *Grant) string {
	var query string

	objects := []string{}
	grant.Objects.ElementsAs(ctx, &objects, false)

	privileges := []string{}
	grant.Privileges.ElementsAs(ctx, &privileges, false)

	switch strings.ToUpper(grant.ObjectType.ValueString()) {
	case "DATABASE":
		query = fmt.Sprintf(
			"GRANT %s ON DATABASE %s TO %s",
			strings.Join(privileges, ","),
			pq.QuoteIdentifier(grant.Database.ValueString()),
			pq.QuoteIdentifier(grant.Role.ValueString()),
		)
	case "SCHEMA":
		query = fmt.Sprintf(
			"GRANT %s ON SCHEMA %s TO %s",
			strings.Join(privileges, ","),
			pq.QuoteIdentifier(grant.Schema.ValueString()),
			pq.QuoteIdentifier(grant.Role.ValueString()),
		)
	case "TABLE":
		if len(objects) > 0 {
			query = fmt.Sprintf(
				"GRANT %s ON %s %s TO %s",
				strings.Join(privileges, ","),
				strings.ToUpper(strings.ToUpper(grant.ObjectType.ValueString())),
				strings.Join(objects, " "),
				pq.QuoteIdentifier(grant.Role.ValueString()),
			)
		} else {
			query = fmt.Sprintf(
				"GRANT %s ON ALL %sS IN SCHEMA %s TO %s",
				strings.Join(privileges, ","),
				strings.ToUpper(strings.ToUpper(grant.ObjectType.ValueString())),
				pq.QuoteIdentifier(grant.Schema.ValueString()),
				pq.QuoteIdentifier(grant.Role.ValueString()),
			)
		}
	}

	return query
}

func getRevokeQuery(ctx context.Context, grant Grant) string {
	var query string

	objects := []string{}
	grant.Objects.ElementsAs(ctx, &objects, false)

	switch strings.ToUpper(grant.ObjectType.ValueString()) {
	case "DATABASE":
		query = fmt.Sprintf(
			"REVOKE ALL PRIVILEGES ON DATABASE %s FROM %s",
			pq.QuoteIdentifier(grant.Database.ValueString()),
			pq.QuoteIdentifier(grant.Role.ValueString()),
		)
	case "SCHEMA":
		query = fmt.Sprintf(
			"REVOKE ALL PRIVILEGES ON SCHEMA %s FROM %s",
			pq.QuoteIdentifier(grant.Schema.ValueString()),
			pq.QuoteIdentifier(grant.Role.ValueString()),
		)
	case "TABLE":
		if len(objects) > 0 {
			query = fmt.Sprintf(
				"REVOKE ALL PRIVILEGES ON %s %s FROM %s",
				strings.ToUpper(strings.ToUpper(grant.ObjectType.ValueString())),
				strings.Join(objects, " "),
				pq.QuoteIdentifier(grant.Role.ValueString()),
			)
		} else {
			query = fmt.Sprintf(
				"REVOKE ALL PRIVILEGES ON ALL %sS IN SCHEMA %s FROM %s",
				strings.ToUpper(strings.ToUpper(grant.ObjectType.ValueString())),
				pq.QuoteIdentifier(grant.Schema.ValueString()),
				pq.QuoteIdentifier(grant.Role.ValueString()),
			)
		}
	}

	return query
}

func revokeRolePrivileges(ctx context.Context, conn *pgx.Conn, grant *Grant) error {
	var err error

	// Grab revoke query
	query := getRevokeQuery(ctx, *grant)

	tflog.Info(ctx, query)

	// Execute SQL
	_, err = conn.Exec(ctx, query)
	if err != nil {
		return err
	}

	// Delete "ID" from grant
	grant.ID = types.StringValue("")

	return nil
}

// Read resource information
func (r resourceGrant) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Grant

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In cases where we are importing state from a single ID, parse the ID into the proper pieces
	idPieces := strings.Split(state.ID.ValueString(), "|")
	state.Role = types.StringValue(idPieces[0])
	state.Database = types.StringValue(idPieces[1])
	state.ObjectType = types.StringValue(idPieces[2])

	tflog.Info(ctx, fmt.Sprintf("Connecting to database '%s'", state.Database.ValueString()))

	// Connect to db
	conn, err := r.p.Conn(ctx, state.Database.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	err = readRolePrivileges(ctx, conn, &state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceGrant) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.p = req.ProviderData.(*cockroachdbProvider)
}

// Create a new resource
func (r resourceGrant) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Grant

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate params
	objects := []string{}
	plan.Objects.ElementsAs(ctx, &objects, false)
	objectType := plan.ObjectType.ValueString()
	if len(objects) > 0 && (objectType == "database" || objectType == "schema") {
		resp.Diagnostics.AddError(
			"Plan validation error",
			"Cannot specify `objects` when `object_type` is `database` or `schema`",
		)
		return
	}

	if objectType == "database" && plan.Schema.ValueString() != "" {
		resp.Diagnostics.AddError(
			"Plan validation error",
			"Cannot specify `schema` when `object_type` is `database`",
		)
		return
	}

	privileges := []string{}
	plan.Privileges.ElementsAs(ctx, &privileges, false)
	if err := utils.ValidatePrivileges(ctx, plan.ObjectType.ValueString(), privileges); err != nil {
		resp.Diagnostics.AddError(
			"Plan validation error",
			err.Error(),
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Connecting to database '%s'", plan.Database.ValueString()))

	// Connect to db
	conn, err := r.p.Conn(ctx, plan.Database.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	// Revoke any existing privileges because it will result in an error if they already exist
	err = revokeRolePrivileges(ctx, conn, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	// Create the Grant
	err = grantRolePrivileges(ctx, conn, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	// Read back what was just set in DB
	err = readRolePrivileges(ctx, conn, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update resource
func (r resourceGrant) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Remove any grants that are stored in state (may or may not be in there)
	var state Grant
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Connecting to database '%s'", state.Database.ValueString()))

	// Connect to db
	conn, err := r.p.Conn(ctx, state.Database.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	// Delete Grant
	err = revokeRolePrivileges(ctx, conn, &state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	// Add any plans
	var plan Grant
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Connecting to database '%s'", plan.Database.ValueString()))

	// Connect to db
	conn, err = r.p.Conn(ctx, plan.Database.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	// Create the Grant
	err = grantRolePrivileges(ctx, conn, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	// Read back what was set in DB
	err = readRolePrivileges(ctx, conn, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	// Update state with what was actually stored
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r resourceGrant) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Grant

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Connecting to database '%s'", state.Database.ValueString()))

	// Connect to db
	conn, err := r.p.Conn(ctx, state.Database.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	// Delete Grant
	err = revokeRolePrivileges(ctx, conn, &state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceGrant) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type Grant struct {
	ID         types.String `tfsdk:"id"`
	Database   types.String `tfsdk:"database"`
	Role       types.String `tfsdk:"role"`
	Schema     types.String `tfsdk:"schema"`
	ObjectType types.String `tfsdk:"object_type"`
	Objects    types.List   `tfsdk:"objects"`
	Privileges types.List   `tfsdk:"privileges"`
}
