BIN := bin
export GOBIN=$(PWD)/$(BIN)

.PHONY:
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.5

lint:
	$(GOBIN)/golangci-lint run ./...