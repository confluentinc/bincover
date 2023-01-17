.PHONY: lint
lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1
	golangci-lint run --timeout=10m

.PHONY: test
test:
ifdef CI
	go install gotest.tools/gotestsum@v1.8.2
	gotestsum --junitfile test-report.xml -- -v ./...
else
	go test -v ./...
endif