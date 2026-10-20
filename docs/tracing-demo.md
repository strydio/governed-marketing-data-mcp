# Tracing demo with Grafana Tempo

## Purpose

This demo shows one trace across the local agent runner, MCP tool execution, MCP server handler, authorisation check, and SQLite repository query.

The trace path is intended to look like:

```text
invoke_agent campaign_investigation
├── model.next
├── execute_tool get_campaign_performance
│   └── MCP tools/call get_campaign_performance
│       ├── authorisation.check
│       └── db.query campaign_performance
```

## Components

Docker Compose starts:

```text
otel-collector  receives OTLP traces from apps
tempo           stores traces
grafana         displays traces
```

Config files:

```text
docker-compose.yml
config/otel-collector.yaml
config/tempo.yaml
config/grafana/provisioning/datasources/tempo.yaml
```

## Environment

Use gRPC OTLP endpoint:

```dotenv
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
```

Recommended service names:

```bash
OTEL_SERVICE_NAME=governed-marketing-data-mcp-server
OTEL_SERVICE_NAME=governed-marketing-data-mcp-agent-runner
```

## Run the tracing stack

```bash
docker compose up -d
```

Grafana will be available at:

```text
http://localhost:3000
```

Anonymous admin access is enabled for this local PoC.

## Run the MCP server

Terminal 1:

```bash
set -a
source .env
set +a

make db-setup
OTEL_SERVICE_NAME=governed-marketing-data-mcp-server make run-server
```

## Run the agent

Terminal 2:

```bash
set -a
source .env
set +a

OTEL_SERVICE_NAME=governed-marketing-data-mcp-agent-runner \
go run ./cmd/agent-runner \
  --provider openai_compatible \
  --output artifacts/runs/agent-openai-run.json
```

You can also use the fake provider:

```bash
OTEL_SERVICE_NAME=governed-marketing-data-mcp-agent-runner \
go run ./cmd/agent-runner \
  --output artifacts/runs/agent-fake-run.json
```

## View traces in Grafana

1. Open `http://localhost:3000`.
2. Go to **Explore**.
3. Select the **Tempo** data source.
4. Search recent traces.
5. Look for spans such as:
   - `invoke_agent campaign_investigation`
   - `execute_tool get_campaign_performance`
   - `MCP tools/call get_campaign_performance`
   - `authorisation.check`
   - `db.query campaign_performance`

## Trace propagation

The agent injects W3C trace context into MCP request `_meta`:

```text
traceparent
tracestate
```

The MCP server extracts these fields before starting the server-side tool span.

Only trace context is propagated this way. Bearer tokens, prompts, raw SQL, full tool arguments, full tool results, and model responses must not be propagated as telemetry metadata.

## Safe telemetry policy

See:

```text
docs/telemetry-data-dictionary.md
```

for the list of allowed and forbidden telemetry fields.
