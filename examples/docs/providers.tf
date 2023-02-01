provider "cockroachdb" {
  host = "localhost"
  user = "root"
  port = 26257
  sslconfig = {
    mode     = "verify-full",
    rootcert = "/Users/bblazer/git/terraform-provider-cockroachdb/certs/ca.crt",
    cert     = "/Users/bblazer/git/terraform-provider-cockroachdb/certs/client.root.crt",
    key      = "/Users/bblazer/git/terraform-provider-cockroachdb/certs/client.root.key",
  }
}
