package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGrantRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: prefixProvider(`
resource "cockroachdb_role" "test_role" {
	name = "test_role"
}

resource "cockroachdb_role" "test_user" {
	name = "test_user"
	login = true
	password = "test_role"
}

resource "cockroachdb_grant_role" "test_grant_role" {
	user = cockroachdb_role.test_user.name
	role = cockroachdb_role.test_role.name
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cockroachdb_grant_role.test_grant_role", "user", "test_user"),
					resource.TestCheckResourceAttr("cockroachdb_grant_role.test_grant_role", "role", "test_role"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "cockroachdb_grant_role.test_grant_role",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"id"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
