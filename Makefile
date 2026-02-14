PROJECT_NAME := vyos
PROVIDER     := pulumi-resource-$(PROJECT_NAME)
VERSION      ?= 0.0.1-dev
GOMODULE     := github.com/jaydoubleu/pulumi-vyos
PULUMI       := $(HOME)/.pulumi/bin/pulumi

XML_DIR      := codegen/vyos-1x/interface-definitions
OUTPUT_DIR   := provider

LDFLAGS := -X $(GOMODULE)/provider.Version=$(VERSION)

.PHONY: provider build clean lint test test_provider test_codegen test_integration schema codegen install generate

provider: bin/$(PROVIDER)

bin/$(PROVIDER): $(shell find provider -name '*.go')
	go build -o $@ -ldflags "$(LDFLAGS)" $(GOMODULE)/provider/cmd/$(PROVIDER)

schema: schema.json

schema.json: bin/$(PROVIDER)
	$(PULUMI) package get-schema ./bin/$(PROVIDER) | jq 'del(.version)' > $@

codegen: schema.json
	@for lang in go nodejs python; do \
		echo "Generating $$lang SDK..."; \
		$(PULUMI) package gen-sdk --language $$lang schema.json --version "$(VERSION)"; \
	done

build: generate provider codegen

generate: codegen/vyos-1x
	rm -f $(OUTPUT_DIR)/resource_gen_*.go
	go run ./codegen/cmd/generate \
		--xml-dir $(XML_DIR) \
		--output-dir $(OUTPUT_DIR)/

codegen/vyos-1x:
	git submodule update --init codegen/vyos-1x

test_provider:
	go test -v -count=1 -race ./provider/...

test_codegen:
	go test -v -count=1 -race ./codegen/...

test_integration:
	go test -v -count=1 -tags=integration ./test/integration/...

test: test_provider test_codegen

lint:
	golangci-lint run ./provider/... ./codegen/...

install: bin/$(PROVIDER)
	cp bin/$(PROVIDER) $(GOPATH)/bin/

clean:
	rm -rf bin/ sdk/ schema.json
	rm -f provider/resource_gen_*.go
