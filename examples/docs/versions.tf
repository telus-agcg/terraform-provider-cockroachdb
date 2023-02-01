terraform {
  required_version = ">= 1.1.9"

  required_providers {
    cockroachdb = {
      source  = "telusag/cockroachdb"
      version = "1.0.0"
    }
  }
}
