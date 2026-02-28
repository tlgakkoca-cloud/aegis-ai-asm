GO ?= go
BINARY ?= aegisctl

.PHONY: deps build test lint run fmt

deps:
	$(GO) mod tidy

build:
	$(GO) build ./...

test:
	$(GO) test ./...

lint:
	$(GO) fmt ./...
	$(GO) vet ./...

run:
	$(GO) run ./cmd/$(BINARY)

fmt:
	gofmt -w $(shell $(GO) list -f '{{.Dir}}' ./...)
