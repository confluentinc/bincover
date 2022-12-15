.golangci-bin:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $@ v1.50.1

.PHONY: deps
deps: .golangci-bin

.PHONY: lint
lint: deps
	.golangci-bin/golangci-lint run --timeout=10m

.PHONY: test
test:
ifdef CI
	go test ./... -v -coverpkg=github.com/confluentinc/bincover -coverprofile=coverage.out
else
	go test ./... -v
endif
