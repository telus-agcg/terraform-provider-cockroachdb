package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/jackc/pgx/v5"
)

func TestAccGrantResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		PreCheck:                 func() { createTractorTable(t) },
		CheckDestroy:             destroyTractorTable,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: prefixProvider(`
resource "cockroachdb_role" "test_role" {
  name = "test_role"
}

resource "cockroachdb_grant" "test_schema_grant" {
  role        = cockroachdb_role.test_role.name
  database    = "defaultdb"
  object_type = "database"
  privileges  = ["ALL"]
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "role", "test_role"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "database", "defaultdb"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "object_type", "database"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "privileges.0", "ALL"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "cockroachdb_grant.test_schema_grant",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"id"},
			},
			// Update and Read testing
			{
				Config: prefixProvider(`
resource "cockroachdb_role" "test_role" {
  name = "test_role"
}

resource "cockroachdb_grant" "test_schema_grant" {
  role        = cockroachdb_role.test_role.name
  database    = "defaultdb"
  schema      = "public"
  object_type = "schema"
  privileges  = ["USAGE"]
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "role", "test_role"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "database", "defaultdb"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "schema", "public"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "object_type", "schema"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "privileges.0", "USAGE"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "cockroachdb_grant.test_schema_grant",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"id"},
			},
			// Update and Read testing
			{
				Config: prefixProvider(`
resource "cockroachdb_role" "test_role" {
	name = "test_role"
}

resource "cockroachdb_grant" "test_schema_grant" {
	role        = cockroachdb_role.test_role.name
	database    = "defaultdb"
	schema      = "public"
	object_type = "table"
	objects     = ["tractor"]
	privileges  = ["ALL"]
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify everything was updated
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "role", "test_role"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "database", "defaultdb"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "schema", "public"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "object_type", "table"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "objects.0", "tractor"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "privileges.0", "ALL"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "cockroachdb_grant.test_schema_grant",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"id"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func getDbConn() (*pgx.Conn, error) {
	pv := getProviderVals()

	provider := new(cockroachdbProvider)
	provider.config.Host = types.StringValue(pv.host)
	provider.config.Port = types.Int64Value(int64(pv.port))
	provider.config.User = types.StringValue(pv.user)
	sslConfig, _ := types.ObjectValue(map[string]attr.Type{
		"mode":     types.StringType,
		"rootcert": types.StringType,
		"cert":     types.StringType,
		"key":      types.StringType,
	}, map[string]attr.Value{
		"mode":     types.StringValue(pv.sslconfig.mode),
		"rootcert": types.StringValue(pv.sslconfig.rootcert),
		"cert":     types.StringValue(pv.sslconfig.cert),
		"key":      types.StringValue(pv.sslconfig.key),
	})
	provider.config.SslConfig = sslConfig

	// Connect to db
	conn, err := provider.Conn(context.Background(), "defaultdb")
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func createTractorTable(t *testing.T) {
	conn, err := getDbConn()
	if err != nil {
		t.Error(
			"Cockroach database connection error",
			err.Error(),
		)
	}

	// Create the tractor table
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE tractor (
			tractor_id    INTEGER UNIQUE,
			tractor_name  VARCHAR(50)
		);
	`)
	if err != nil {
		t.Error(
			"Cockroach error creating test table",
			err.Error(),
		)
	}
}

func destroyTractorTable(s *terraform.State) error {
	conn, err := getDbConn()
	if err != nil {
		return err
	}

	// Delete the tractor table
	_, err = conn.Exec(context.Background(), `DROP TABLE tractor;`)
	if err != nil {
		return err
	}

	return nil
}
