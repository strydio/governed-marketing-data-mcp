.PHONY: help test vet build run-server run-agent fmt tidy db-reset db-seed smoke

help:
	@echo "Available targets:"
	@echo "  make test       Run Go tests"
	@echo "  make vet        Run go vet"
	@echo "  make build      Build binaries into ./bin"
	@echo "  make run-server Run the MCP server"
	@echo "  make run-agent  Run the agent runner"
	@echo "  make fmt        Format Go code"
	@echo "  make tidy       Tidy Go module"
	@echo "  make db-reset   Reset local demo database (not implemented yet)"
	@echo "  make db-seed    Seed local demo database (not implemented yet)"
	@echo "  make smoke      Run smoke test (not implemented yet)"

test:
	go test ./...

vet:
	go vet ./...

build:
	mkdir -p bin
	go build -o bin/mcp-server ./cmd/mcp-server
	go build -o bin/agent-runner ./cmd/agent-runner

run-server:
	go run ./cmd/mcp-server

run-agent:
	go run ./cmd/agent-runner

fmt:
	gofmt -w $$(find . -name '*.go')

tidy:
	go mod tidy

db-reset:
	go run ./cmd/mcp-server --db-reset

db-seed:
	go run ./cmd/mcp-server --db-seed

smoke:
	@echo "smoke test is not implemented yet"
