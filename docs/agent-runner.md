# Agent runner

## Purpose

`cmd/agent-runner` runs the LLM-agent orchestration path.

It can use either the deterministic fake model provider or an OpenAI-compatible chat-completions provider, while still calling the real MCP server through the MCP protocol.

The flow is:

```text
model provider -> agent loop -> MCP tool executor -> MCP server -> run record
```

This proves the agent loop can execute model-requested MCP tool calls while enforcing the governed tool boundary.

## Provider support

Supported providers:

```text
fake
openai_compatible
```

The fake provider deterministically requests:

```text
get_campaign_performance
get_conversion_funnel
get_tracking_health
get_data_freshness
```

Then it returns a fixed final diagnosis.

## Command

With the MCP server running, run with the default fake provider:

```bash
DEMO_BEARER_TOKEN=replace-me go run ./cmd/agent-runner
```

Run with an OpenAI-compatible provider:

```bash
MODEL_BASE_URL=https://api.openai.com/v1 \
MODEL_API_KEY=your-key \
MODEL_NAME=gpt-4o-mini \
DEMO_BEARER_TOKEN=replace-me \
go run ./cmd/agent-runner --provider openai_compatible
```

Write output to a file:

```bash
DEMO_BEARER_TOKEN=replace-me go run ./cmd/agent-runner \
  --output artifacts/runs/agent-run.json
```

Evaluate the run:

```bash
make run-evaluator INPUT=artifacts/runs/agent-run.json
```

## Flags

```text
--endpoint      MCP Streamable HTTP endpoint
--token         bearer token for MCP server
--output        optional run-record output path
--campaign      campaign slug
--market        market code
--start-date    start date in YYYY-MM-DD format
--end-date      end date in YYYY-MM-DD format
--provider      model provider: fake or openai_compatible; defaults to MODEL_PROVIDER or fake
```

## Difference from deterministic client

`cmd/client` is a deterministic MCP client. It directly calls the four tools in a fixed order.

`cmd/agent-runner` runs through the agent abstraction. The model provider requests tool calls, and the agent loop validates and executes those requests.

With `--provider fake`, behaviour is deterministic. With `--provider openai_compatible`, tool selection and final diagnosis are model-driven through a `/chat/completions` compatible API.

## OpenAI-compatible configuration

The OpenAI-compatible provider uses:

```dotenv
MODEL_BASE_URL=https://api.openai.com/v1
MODEL_API_KEY=your-key
MODEL_NAME=gpt-4o-mini
```

The API key is sent only as an HTTP `Authorization` header to the model provider. It must not be logged, traced, or committed.

## Safety controls currently implemented

The agent loop rejects:

- missing model provider;
- missing tool executor;
- model-requested unknown tools such as `run_sql`;
- malformed final diagnosis JSON;
- incomplete final diagnosis;
- too many tool calls;
- too many model turns.

Defaults:

```text
MaxTurns = 12
MaxToolCalls = 8
```
