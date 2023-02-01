PKG_NAME=cockroachdb

default: install

format-examples:
	terraform fmt -recursive ./examples/

generate-docs:
	go get github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

release:
	GITHUB_TOKEN=$(GITHUB_TOKEN) \
	GPG_FINGERPRINT=$(GPG_FINGERPRINT) \
		goreleaser release --rm-dist

install:
	go install .

test:
	go test -count=1 -parallel=4 ./...

testacc:
	TF_ACC=1 go test -count=1 -parallel=4 -timeout 10m -v ./...

build-dev:
	go mod init || true
	go fmt
	go mod tidy
	go build -o ~/go/bin/terraform-provider-cockroachdb

create-dev-overrides:
	@echo "Create a file at ~/.terraformrc"
	@echo "Follow these instructions https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers"
	@echo "Set dev_overrides to \"telusag/cockroachdb\" = \"[bin_location]\""

# Taken straight from https://www.cockroachlabs.com/docs/stable/cockroach-start-single-node.html#start-a-single-node-cluster
# Pre-req to install the cockroach CLI
start-crdb:
	mkdir -p certs my-safe-directory
	cockroach cert create-ca --certs-dir=certs --ca-key=my-safe-directory/ca.key --allow-ca-key-reuse --overwrite
	cockroach cert create-node localhost $(hostname) --certs-dir=certs --ca-key=my-safe-directory/ca.key --overwrite
	cockroach cert create-client root --certs-dir=certs --ca-key=my-safe-directory/ca.key --overwrite
	cockroach start-single-node --certs-dir=certs --listen-addr=localhost:26257 --http-addr=localhost:8080