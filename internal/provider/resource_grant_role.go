package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &resourceGrantRole{}
	_ resource.ResourceWithConfigure   = &resourceGrantRole{}
	_ resource.ResourceWithImportState = &resourceGrantRole{}
)

func NewGrantRoleResource() resource.Resource {
	return &resourceGrantRole{}
}

type resourceGrantRole struct {
	p *cockroachdbProvider
}

func (r *resourceGrantRole) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "cockroachdb_grant_role"
}

func (r *resourceGrantRole) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a grant role.",
		Attributes: map[string]schema.Attribute{
			"user": schema.StringAttribute{
				Description: "User or role to grant to",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Required: true,
			},
			"role": schema.StringAttribute{
				Description: "Role to grant",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Description: "ID of the grant_role",
				Computed:    true,
			},
		},
	}
}

func (r *resourceGrantRole) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.p = req.ProviderData.(*cockroachdbProvider)
}

func CreateGrantRole(ctx context.Context, conn *pgx.Conn, grantRole *GrantRole) error {
	var err error

	// Execute SQL
	_, err = conn.Exec(ctx, fmt.Sprintf(
		"GRANT %s TO %s",
		pq.QuoteIdentifier(grantRole.Role.ValueString()),
		pq.QuoteIdentifier(grantRole.User.ValueString()),
	))
	if err != nil {
		return err
	}

	// Set the ID of the grant role
	grantRole.ID = types.StringValue(grantRole.Role.ValueString() + "|" + grantRole.User.ValueString())

	return nil
}

// Create a new resource
func (r *resourceGrantRole) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GrantRole

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

	// Create the Grant Role
	err = CreateGrantRole(ctx, conn, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r *resourceGrantRole) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GrantRole

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Decode "ID" to user and role
	roleUser := strings.Split(state.ID.ValueString(), "|")
	state.Role = types.StringValue(roleUser[0])
	state.User = types.StringValue(roleUser[1])

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
		role_name string
		member    string
	)
	rows, _ := conn.Query(context.Background(), fmt.Sprintf("SHOW GRANTS FOR %s", pq.QuoteIdentifier(state.User.ValueString())))
	_, err = pgx.ForEachRow(rows, []interface{}{&role_name, &member}, func() error {
		// Update state
		if role_name == state.Role.ValueString() {
			state.Role = types.StringValue(role_name)
			state.User = types.StringValue(member)
			state.ID = types.StringValue(state.Role.ValueString() + "|" + state.User.ValueString())
		}

		return nil
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach execute sql error",
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

// Update resource
func (r *resourceGrantRole) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Connect to db
	conn, err := r.p.Conn(ctx, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach connection error",
			err.Error(),
		)
		return
	}

	// Remove any grants that are stored in state (may or may not be in there)
	var state GrantRole
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete Grant
	err = DeleteGrantRole(ctx, conn, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	// Add any plans
	var plan GrantRole
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the Grant Role
	err = CreateGrantRole(ctx, conn, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cockroach sql error",
			err.Error(),
		)
		return
	}

	// Update state with what was actually stored
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func DeleteGrantRole(ctx context.Context, conn *pgx.Conn, grantRole GrantRole) error {
	var err error

	// Execute SQL
	_, err = conn.Exec(ctx, fmt.Sprintf(
		"REVOKE %s FROM %s",
		pq.QuoteIdentifier(grantRole.Role.ValueString()),
		pq.QuoteIdentifier(grantRole.User.ValueString()),
	))
	if err != nil {
		return err
	}

	// Delete "ID" from grant
	grantRole.ID = types.StringValue("")

	return nil
}

// Delete resource
func (r *resourceGrantRole) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GrantRole

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

	// Delete Grant Role
	err = DeleteGrantRole(ctx, conn, state)
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

func (r *resourceGrantRole) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type GrantRole struct {
	ID   types.String `tfsdk:"id"`
	User types.String `tfsdk:"user"`
	Role types.String `tfsdk:"role"`
}
