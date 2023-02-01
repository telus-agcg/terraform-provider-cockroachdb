package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGrantResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
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
	objects     = ["distributors"]
	privileges  = ["ALL"]
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify everything was updated
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "role", "test_role"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "database", "defaultdb"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "schema", "public"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "object_type", "table"),
					resource.TestCheckResourceAttr("cockroachdb_grant.test_schema_grant", "objects.0", "distributors"),
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
