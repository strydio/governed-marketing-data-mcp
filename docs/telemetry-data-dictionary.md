# Telemetry data dictionary

## Scope

This document describes the safe telemetry policy for backend MCP traces.

The goal is to make backend execution observable without turning traces into an ungoverned copy of business data, credentials, prompts, SQL, or tool responses.

## Default mode

Telemetry content capture is off by default:

```dotenv
TELEMETRY_CAPTURE_CONTENT=false
```

Safe mode must not record raw prompts, bearer tokens, raw SQL, full tool arguments, or full tool results.

## Trace spans planned for the backend

Backend MCP requests should produce spans similar to:

```text
MCP tools/call get_campaign_performance
├── authorisation.check
└── db.query campaign_performance
```

Denied requests should produce no database span:

```text
MCP tools/call get_campaign_performance
└── authorisation.check
```

## Allowed trace attributes

The following attributes are allowed because they are operational, low-cardinality, or sanitised.

| Attribute | Example | Notes |
|---|---:|---|
| `mcp.method.name` | `tools/call` | MCP method category. |
| `gen_ai.tool.name` | `get_campaign_performance` | Tool name exposed by the MCP server. |
| `campaign.market` | `UK` | Low-cardinality market code. |
| `campaign.id_hash` | `a1b2c3d4e5f6` | Optional stable hash if campaign correlation is needed. |
| `data.source` | `ad_reporting` | Logical data source, not physical DB path. |
| `data.complete_through` | `2026-09-21T04:00:00Z` | Freshness timestamp. |
| `result.record_count` | `1` | Repository/database result count only, not records. |
| `authorisation.decision` | `allow` / `deny` | Project-owned British-English spelling. |
| `authorisation.reason` | `policy_match` | Safe reason code. |
| `evidence.count` | `1` | Number of evidence objects returned by an MCP tool, not findings, rows, IDs, or payload. |

## Result and evidence counts

MCP tool spans may set:

```text
evidence.count = 1
```

This means the tool returned one structured evidence object. It must not contain evidence IDs or the evidence payload.

Repository spans may set:

```text
result.record_count = 3
```

This represents the number of rows/findings/source records involved in the repository result. It must not contain the records themselves.

For example, `get_tracking_health` returns one evidence object that may contain one or more sanitised findings. The MCP span should report `evidence.count = 1`; the repository span can report the finding row count using `result.record_count`.

## Values not captured by default

The following must not be captured in traces in safe mode:

- bearer tokens;
- API keys;
- environment variables;
- raw user prompts;
- system prompts;
- complete model responses;
- full tool arguments;
- full tool results;
- raw SQL;
- generated SQL;
- database file paths;
- customer identifiers;
- customer-level data;
- campaign slug as a clear trace attribute.

## Campaign identifiers

`campaign.market` may be captured because values such as `UK` or `US` are low-cardinality and useful for operations.

Clear campaign slugs should not be captured as trace attributes by default. Campaign names can reveal commercial information, for example product launches, competitor strategy, audience targeting, or internal naming conventions.

If correlation by campaign is needed, use a stable hash instead:

```text
campaign.id_hash = sha256(campaign_slug + ":" + market), truncated
```

The clear campaign slug may appear in local structured authorisation events for the PoC, but should not be used as a broad trace attribute.

## Structured authorisation events

Authorisation decisions may be logged as structured events on stdout in v1:

```json
{
  "event": "authorisation_decision",
  "principal_ref": "uk_marketing_demo",
  "tool": "get_campaign_performance",
  "target": {
    "campaign": "autumn-sale",
    "market": "UK"
  },
  "decision": "allow",
  "reason": "policy_match"
}
```

These events are local PoC audit-style logs. They are not production-compliant audit logs and have different data-handling rules from traces.

## Metrics guidance

Future metrics should use only low-cardinality labels such as:

- tool name;
- outcome;
- authorisation decision;
- error type.

Metrics must not be labelled by:

- principal;
- campaign slug;
- evidence ID;
- prompt;
- run ID;
- date range.
