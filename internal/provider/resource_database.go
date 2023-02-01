package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &resourceDatabase{}
	_ resource.ResourceWithConfigure   = &resourceDatabase{}
	_ resource.ResourceWithImportState = &resourceDatabase{}
)

func NewDatabaseResource() resource.Resource {
	return &resourceDatabase{}
}

type resourceDatabase struct {
	p *cockroachdbProvider
}

func (r *resourceDatabase) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "cockroachdb_database"
}

func (r *resourceDatabase) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a database.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the database",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Required: true,
			},
			"owner": schema.StringAttribute{
				Description: "Owner of the database",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Description: "ID of the database",
				Computed:    true,
			},
		},
	}
}

func (r *resourceDatabase) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.p = req.ProviderData.(*cockroachdbProvider)
}

// Create a new resource
func (r *resourceDatabase) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Database

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

	// Execute SQL
	if plan.Owner.ValueString() == "" {
		_, err = conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, plan.Name.ValueString()))
	} else {
		_, err = conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, plan.Name.ValueString(), plan.Owner.ValueString()))
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach execute sql error",
			err.Error(),
		)
		return
	}

	var id int
	err = conn.QueryRow(ctx, `SELECT id FROM crdb_internal.databases WHERE name = $1`, plan.Name.ValueString()).Scan(
		&id,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach execute sql error",
			err.Error(),
		)
		return
	}

	// Set the ID of the database
	plan.ID = types.StringValue(strconv.Itoa(id))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r *resourceDatabase) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Database

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

	var (
		name  string
		owner string
	)
	err = conn.QueryRow(ctx, `SELECT name, owner FROM crdb_internal.databases WHERE id = $1`, state.ID.ValueString()).Scan(
		&name,
		&owner,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach execute sql error",
			err.Error(),
		)
		return
	}

	// Update state with the current name and owner
	state.Name = types.StringValue(name)
	state.Owner = types.StringValue(owner)

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update resource
func (r *resourceDatabase) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var (
		stateDb Database
		planDb  Database
	)

	diags := req.State.Get(ctx, &stateDb)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.Plan.Get(ctx, &planDb)
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

	if stateDb.Name.ValueString() != planDb.Name.ValueString() {
		// Update database
		_, err := conn.Exec(ctx, fmt.Sprintf(`ALTER DATABASE %s RENAME TO %s`, stateDb.Name.ValueString(), planDb.Name.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError(
				"Cockroach connection error",
				err.Error(),
			)
			return
		}

		// Update state
		stateDb.Name = types.StringValue(planDb.Name.ValueString())
	}

	if stateDb.Owner.ValueString() != planDb.Owner.ValueString() {
		// Update database
		_, err := conn.Exec(ctx, fmt.Sprintf(`ALTER DATABASE %s OWNER TO %s`, stateDb.Name.ValueString(), planDb.Owner.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError(
				"Cockroach execute sql error",
				err.Error(),
			)
			return
		}

		// Update state
		stateDb.Owner = types.StringValue(planDb.Owner.ValueString())
	}

	// Set state
	diags = resp.State.Set(ctx, &stateDb)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r *resourceDatabase) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Database

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

	_, err = conn.Exec(ctx, fmt.Sprintf(`DROP DATABASE %s`, state.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	// Set id to ""
	state.ID = types.StringValue("")

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceDatabase) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type Database struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Owner types.String `tfsdk:"owner"`
}
