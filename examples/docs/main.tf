resource "cockroachdb_database" "test_db" {
  name = "test_db"
}

resource "cockroachdb_role" "test_role" {
  name = "test_role"
}

resource "cockroachdb_role" "test_user" {
  name     = "test_user"
  login    = true
  password = "password123"
}

resource "cockroachdb_grant_role" "test_grant_role" {
  user = cockroachdb_role.test_user.name
  role = cockroachdb_role.test_role.name
}

resource "cockroachdb_grant" "test_schema_grant" {
  role        = cockroachdb_role.test_role.name
  database    = "test_db"
  schema      = "public"
  object_type = "schema"
  privileges  = ["USAGE"]
}
