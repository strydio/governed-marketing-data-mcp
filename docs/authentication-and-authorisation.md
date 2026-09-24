# Authentication and authorisation

## Scope

This project currently uses a deliberately local proof-of-concept authentication and authorisation model.

The aim is to prove that the MCP server behaves as a governed data-access layer before introducing production identity infrastructure.

Production OAuth, OIDC, central policy engines, warehouse row-level security, and compliant audit logging are out of scope for the first implementation.

## Authentication model

Authentication uses a fixed local bearer token.

A caller must send:

```http
Authorization: Bearer <token>
```

The server compares the token with the configured demo token:

```dotenv
DEMO_BEARER_TOKEN=replace-me
```

The implementation uses the official MCP Go SDK auth middleware:

```go
auth.RequireBearerToken(...)
```

If the token is valid, the verifier stores the local principal reference in MCP SDK token info:

```go
TokenInfo{UserID: "uk_marketing_demo"}
```

Conceptually:

```text
HTTP Authorization header -> TokenInfo.UserID -> principal_ref
```

For the initial demo there is one configured local identity:

```text
uk_marketing_demo
```

### Authentication rules

- The server must fail closed if `DEMO_BEARER_TOKEN` is missing.
- The server must not log bearer tokens.
- The server must not include token values in errors, traces, or audit events.
- Missing, malformed, or invalid credentials must return safe authentication failures.
- This is a local PoC mechanism only, not a production identity design.

## Authorisation model

Authorisation is enforced by an in-process policy layer backed by YAML.

The demo policy file is:

```text
config/policy.demo.yaml
```

Expected initial shape:

```yaml
principals:
  uk_marketing_demo:
    role: marketing_analyst
    allowed_markets: [UK]
    allowed_campaigns: [autumn-sale]
    allowed_tools:
      - get_campaign_performance
      - get_conversion_funnel
      - get_tracking_health
      - get_data_freshness
    max_date_range_days: 31
    customer_level_data: false
```

Each MCP tool call is authorised independently. Tool discovery is not a security boundary.

## Per-call decision sequence

For every tool call, the server must perform this sequence:

```text
1. Authenticate bearer token at the HTTP boundary and resolve principal.
2. Validate tool input shape and common fields.
3. Validate date range.
4. Resolve whether the requested campaign and market are valid without leaking hidden resources.
5. Check the principal is allowed to call the requested tool.
6. Check the principal is allowed to access the requested market.
7. Check the principal is allowed to access the requested campaign.
8. Check the requested date range is within the principal's maximum range.
9. Reject customer-level access; v1 supports aggregate data only.
10. Query the repository only after an allow decision.
```

## Input validation

Common campaign tool inputs are expected to include:

```json
{
  "campaign": "autumn-sale",
  "market": "UK",
  "start_date": "2026-09-14",
  "end_date": "2026-09-20"
}
```

Validation rules:

- `campaign` must be a slug, not free text.
- `market` must be a supported enum value.
- Dates must be ISO `YYYY-MM-DD`.
- `start_date` must be on or before `end_date`.
- Date ranges must be non-empty.
- Date ranges must not exceed the policy maximum, initially 31 days.
- SQL-looking campaign strings must be rejected before repository access.

## Denial behaviour

Denied requests must be safe and non-revealing.

Errors must not expose:

- bearer tokens;
- SQL;
- database file paths;
- environment variables;
- hidden campaign names;
- policy file internals;
- customer-level data.

A UK-only user requesting a US campaign should receive a generic authorisation failure rather than a detailed resource-discovery response.

## Structured authorisation decisions

Each authorisation decision should produce a structured event that can be correlated with traces.

Example:

```json
{
  "event": "authorisation_decision",
  "trace_id": "...",
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

These events are local structured logs in v1. They are not claimed to be production-compliant audit logs.

## Current implementation plan

Packages:

```text
internal/config    - environment and file-path configuration
internal/authn     - bearer-token verification and principal extraction from request context
internal/authz     - YAML policy loading and per-call authorisation
internal/domain    - shared request validation helpers and domain types
```

The MCP server will later call these packages in this order:

```text
authn -> input validation -> authz -> repository -> MCP response
```

The repository must not be called if authentication, validation, or authorisation fails.

## Future production direction

A production version could replace or extend this design with:

- OIDC/JWT validation;
- identity-provider group/team claims;
- central policy service or OPA;
- warehouse row-level security as defence in depth;
- signed and retained audit logs;
- key rotation;
- multiple principals and service accounts.

Those are intentionally deferred until the local governed MCP flow is working end to end.
