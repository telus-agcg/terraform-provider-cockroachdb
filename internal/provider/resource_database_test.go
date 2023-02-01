package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatabaseResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: prefixProvider(`
resource "cockroachdb_database" "test_database" {
	name = "test_database"
	owner = "root"
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify name and owner of database
					resource.TestCheckResourceAttr("cockroachdb_database.test_database", "name", "test_database"),
					resource.TestCheckResourceAttr("cockroachdb_database.test_database", "owner", "root"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "cockroachdb_database.test_database",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"id"},
			},
			// Update and Read testing
			{
				Config: prefixProvider(`
			resource "cockroachdb_database" "test_database" {
				name = "test_database_two"
				owner = "root"
			}
			`),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify database name was updated
					resource.TestCheckResourceAttr("cockroachdb_database.test_database", "name", "test_database_two"),
					resource.TestCheckResourceAttr("cockroachdb_database.test_database", "owner", "root"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
