SHELL           := /bin/bash
ALL_SRC         := $(shell find . -name "*.go" | grep -v -e vendor)
GIT_REMOTE_NAME ?= origin
MASTER_BRANCH   ?= master

.PHONY: fmt
fmt:
	@gofmt -e -s -l -w $(ALL_SRC)

.PHONY: lint-go
lint-go:
	@GO111MODULE=on golangci-lint run --timeout=10m --skip-files="input_bin/exit_code_1.go"

.PHONY: lint
lint: lint-go 

.PHONY: coverage
coverage:
      ifdef CI
	@# Run unit tests with coverage.
	@go test .  -coverpkg=./...  -coverprofile=coverage.txt
      else
	@go test .
      endif

.PHONY: test
test: lint coverage
