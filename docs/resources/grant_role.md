---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cockroachdb_grant_role Resource - terraform-provider-cockroachdb"
subcategory: ""
description: |-
---

# cockroachdb_grant_role (Resource)

## Example Usage

```terraform
resource "cockroachdb_role" "test_role" {
  name = "test_role"
}

resource "cockroachdb_role" "test_user" {
  name     = "test_user"
  login    = true
  password = "test_password_a1s2d3f4"
}

resource "cockroachdb_grant_role" "test_user_grant_test_role" {
  role       = cockroachdb_role.test_user.name
  grant_role = cockroachdb_role.test_role.name
}
```

<!-- schema generated by tfplugindocs -->

## Schema

### Required

- `user` (String) User or role to grant to

### Optional

- `role` (String) Role to grant

### Read-Only

- `id` (String) ID of the grant_role