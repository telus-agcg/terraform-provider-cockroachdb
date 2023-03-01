# CockroachDB Terraform Provider

## Introduction

This is a Terraform Provider for CockroachDb. It allows you to configure various resources including databases, user, roles, and permissions.

## Installation

To install this provider, use the following snippet in your Terraform configuration:

```hcl
terraform {
  required_providers {
    cockroachdb = {
      source  = "telusag/cockroachdb"
      version = ">= 0.0.1"
    }
  }
}

provider "cockroachdb" {
  host     = "host"
  port     = 26257
  username = "username"
  password = "password"
  database = "database"
}
```

## Usage

For documentation on the available resources and their properties, see [docs](docs/index.md)

## Development

To enable provider development, check out the provider and install the binary into your local `~/go/bin` path:

```
$ go install
```

In order for Terraform to use the provider installed locally, rather than downloading a fresh copy from the registry, enable a development override in your `~/terraformrc`:

```hcl
provider_installation {

  dev_overrides {
      "telusag/cockroachdb" = "your ~/go/bin path"
  }

  direct {}

}
```

To start a local cockroach db and mount certs so you can connect to it, run (from root of project) `make start-crdb`
To connect as root using psql run: `psql "postgres://root@localhost:26257/defaultb?sslmode=verify-full&sslrootcert=./certs/ca.crt&sslcert=./certs/client.root.crt&sslkey=./certs/client.root.key"`
To connect via another user, create the user in crdb (`CREATE USER [username]`) or using TF and then run the following commands (need cockroach installed):
- `cockroach cert create-client [username] --certs-dir=certs --ca-key=my-safe-directory/ca.key`

Always run `gofmt -w -s .` before committing to make sure the diffs don't contain minor formatting differences.

### Docs

To generate or update documentation, run `make generate-docs`.

### Tests

In order to run the full suite of Acceptance tests, first run `make start-test-crdb` (in a different terminal) and wait for the DB to become available. Next run `make testacc` to run the acc tests against the db.

## Publishing

1. [Create and upload a GPG key](https://www.terraform.io/cloud-docs/registry/publish-providers#publishing-a-provider-and-creating-a-version) to Terraform Cloud.
1. Set the GPG secret key as an action secret on the repository called GPG_SECRET_KEY, the passphrase should be set as an action secret called PASSPHRASE.
1. Create a tag on the repository called vx.y.z (v1.0.0) and push the tag to Github.
1. Github Actions will build and publish the new release
