BIN := bin
export GOBIN=$(PWD)/$(BIN)

.PHONY:
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.5

.PHONY:
lint:
	$(GOBIN)/golangci-lint run ./...

.PHONY:
test-backends:
	go test -timeout=5m ./backends/...