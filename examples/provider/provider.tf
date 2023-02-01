provider "cockroachdb" {
  host = "localhost"
  user = "root"
  port = 26257
  sslconfig = {
    mode     = "disable"
    rootcert = ""
    cert     = ""
    key      = ""
  }
}
