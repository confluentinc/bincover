.PHONY: deps
deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1

.PHONY: lint
lint: deps
	golangci-lint run --timeout=10m

.PHONY: test
test:
ifdef CI
	go test ./... -v -coverpkg=github.com/confluentinc/bincover -coverprofile=coverage.out
else
	go test ./... -v
endif
