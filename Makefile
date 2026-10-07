.PHONY: help test vet build run-server run-client run-agent run-evaluator fmt tidy db-reset db-seed db-setup smoke

help:
	@echo "Available targets:"
	@echo "  make test       Run Go tests"
	@echo "  make vet        Run go vet"
	@echo "  make build      Build binaries into ./bin"
	@echo "  make run-server Run the MCP server"
	@echo "  make run-client Run the deterministic MCP client"
	@echo "  make run-agent  Run the LLM-backed agent runner; defaults to fake provider"
	@echo "  make run-evaluator Run the evaluator; pass INPUT=path/to/run.json"
	@echo "  make fmt        Format Go code"
	@echo "  make tidy       Tidy Go module"
	@echo "  make db-reset   Reset local demo database (not implemented yet)"
	@echo "  make db-seed    Seed local demo database"
	@echo "  make db-setup   Reset and seed local demo database"
	@echo "  make smoke      Run backend MCP smoke test against a running server"

test:
	go test ./...

vet:
	go vet ./...

build:
	mkdir -p bin
	go build -o bin/mcp-server ./cmd/mcp-server
	go build -o bin/client ./cmd/client
	go build -o bin/agent-runner ./cmd/agent-runner
	go build -o bin/evaluator ./cmd/evaluator
	go build -o bin/smoke-test ./cmd/smoke-test

run-server:
	go run ./cmd/mcp-server

run-client:
	go run ./cmd/client

run-agent:
	go run ./cmd/agent-runner

run-evaluator:
	@if [ -z "$(INPUT)" ]; then echo "INPUT is required, e.g. make run-evaluator INPUT=artifacts/runs/client-run.json"; exit 1; fi
	go run ./cmd/evaluator --input $(INPUT)

fmt:
	gofmt -w $$(find . -name '*.go')

tidy:
	go mod tidy

db-reset:
	go run ./cmd/mcp-server --db-reset

db-seed:
	go run ./cmd/mcp-server --db-seed

db-setup: db-reset db-seed

smoke:
	./scripts/smoke-test.sh
