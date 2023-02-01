package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: prefixProvider(`
resource "cockroachdb_role" "simple_role" {
  name = "simple"
}

resource "cockroachdb_role" "complex_role" {
  name = "complex"
  password = "test_password_abcdefg"
  create_database = true
  create_role = true
  login = true
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cockroachdb_role.simple_role", "name", "simple"),
					resource.TestCheckResourceAttr("cockroachdb_role.complex_role", "name", "complex"),
					resource.TestCheckResourceAttr("cockroachdb_role.complex_role", "password", "test_password_abcdefg"),
					resource.TestCheckResourceAttr("cockroachdb_role.complex_role", "create_database", "true"),
					resource.TestCheckResourceAttr("cockroachdb_role.complex_role", "create_role", "true"),
					resource.TestCheckResourceAttr("cockroachdb_role.complex_role", "login", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "cockroachdb_role.simple_role",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"id", "password"},
			},
			{
				ResourceName:            "cockroachdb_role.complex_role",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"id", "password"},
			},
			// Update and Read testing
			{
				Config: prefixProvider(`
resource "cockroachdb_role" "complex_role" {
	name = "complex"
}
			`),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify everything was updated
					resource.TestCheckResourceAttr("cockroachdb_role.complex_role", "name", "complex"),
					resource.TestCheckNoResourceAttr("cockroachdb_role.complex_role", "password"),
					resource.TestCheckNoResourceAttr("cockroachdb_role.complex_role", "create_database"),
					resource.TestCheckNoResourceAttr("cockroachdb_role.complex_role", "create_role"),
					resource.TestCheckNoResourceAttr("cockroachdb_role.complex_role", "login"),
				),
			},
			{
				ResourceName:            "cockroachdb_role.complex_role",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"id", "password"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
