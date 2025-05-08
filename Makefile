BIN := bin
export GOBIN=$(PWD)/$(BIN)

.PHONY:
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.5
	go install github.com/vektra/mockery/v3@v3.2.5

.PHONY:
lint:
	$(GOBIN)/golangci-lint run ./...

.PHONY:
fmt:
	gofumpt -l -w .

.PHONY:
mock:
	$(GOBIN)/mockery

.PHONY:
test.unit:
	go test .

.PHONY:
test.integration:
	go test -timeout=1m ./backends/...

.PHONY:
test.e2e:
	go test -timeout=1m ./e2e

.PHONY:
test: test.unit test.integration test.e2e