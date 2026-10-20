# Architecture

## Current status

Initial scaffold for a governed marketing data MCP proof of concept.

## Repository shape

The repository contains two binaries:

- `cmd/mcp-server`: the governed MCP server over marketing data.
- `cmd/agent-runner`: the later demo/experiment client that calls the MCP server.

The MCP server must not depend on the agent runner or model-provider packages. The reusable backend lives in `internal/mcpserver`, `internal/repository`, `internal/authn`, `internal/authz`, `internal/domain`, `internal/config`, and `internal/telemetry`.

## Dependency direction

```text
mcp-server -> mcpserver/repository/authn/authz/domain/config/telemetry
agent-runner -> agent/model/evaluator/reporting/domain/telemetry
agent-runner -> MCP over HTTP -> mcp-server
```

## Authentication and authorisation

The initial implementation uses local fixed bearer-token authentication, enforced at the HTTP boundary with the official MCP Go SDK auth middleware, and YAML-backed in-process authorisation policy.

Every MCP tool call must be checked independently:

```text
authenticate -> validate input -> authorise -> query repository -> return evidence
```

The repository must not be called if authentication, validation, or authorisation fails.

See [authentication-and-authorisation.md](authentication-and-authorisation.md) for the detailed design.

## Telemetry safety

Backend traces should capture operational metadata such as tool name, market, authorisation decision, logical data source, freshness, and record counts.

Traces must not capture bearer tokens, raw SQL, full tool arguments, full tool results, prompts, database paths, or clear campaign slugs by default.

Use `campaign.id_hash` rather than clear campaign slug if campaign-level trace correlation is needed.

See [telemetry-data-dictionary.md](telemetry-data-dictionary.md) for the trace attribute policy.
