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

start-test-crdb:
	mkdir -p certs my-safe-directory cockroach-data
	docker run -it -v $(CURDIR)/certs:/certs -v $(CURDIR)/my-safe-directory:/my-safe-directory cockroachdb/cockroach:v22.2.4 cert create-ca --certs-dir=/certs --ca-key=/my-safe-directory/ca.key --allow-ca-key-reuse --overwrite
	docker run -it -v $(CURDIR)/certs:/certs -v $(CURDIR)/my-safe-directory:/my-safe-directory cockroachdb/cockroach:v22.2.4 cert create-node 127.0.0.1 localhost --certs-dir=/certs --ca-key=/my-safe-directory/ca.key --overwrite
	docker run -it -v $(CURDIR)/certs:/certs -v $(CURDIR)/my-safe-directory:/my-safe-directory cockroachdb/cockroach:v22.2.4 cert create-client root --certs-dir=/certs --ca-key=/my-safe-directory/ca.key --overwrite
	docker run -it -p 8080:8080 -p 26257:26257 -v $(CURDIR)/cockroach-data:/cockroach-data -v $(CURDIR)/certs:/certs -v $(CURDIR)/my-safe-directory:/my-safe-directory cockroachdb/cockroach:v22.2.4 start-single-node --certs-dir=/certs