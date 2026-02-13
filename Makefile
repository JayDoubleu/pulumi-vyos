PROJECT_NAME := vyos
PROVIDER     := pulumi-resource-$(PROJECT_NAME)
VERSION      ?= 0.0.1-dev
GOMODULE     := github.com/jaydoubleu/pulumi-vyos
PULUMI       := $(HOME)/.pulumi/bin/pulumi

LDFLAGS := -X $(GOMODULE)/provider.Version=$(VERSION)

.PHONY: provider build clean lint test test_provider schema codegen install generate

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

build: provider codegen

test_provider:
	go test -v -count=1 -race ./provider/...

test: test_provider

lint:
	golangci-lint run ./provider/...

install: bin/$(PROVIDER)
	cp bin/$(PROVIDER) $(GOPATH)/bin/

generate:
	@echo "Code generation from VyOS XML definitions is not yet implemented."
	@echo "See codegen/ directory and DESIGN.md Phase 2 for details."

clean:
	rm -rf bin/ sdk/ schema.json
