# Cause taxonomy

## Current PoC schema

The PoC uses a canonical cause label in the diagnosis `cause` field.

The `cause` field is machine-readable and must not contain prose.

Allowed cause labels are defined in code under:

```text
internal/runs/causes.go
```

Current labels:

```text
conversion_tracking_under_reporting
safari_tracking_failure
missing_conversion_events
lower_quality_traffic
budget_or_delivery_change
stale_or_incomplete_data
unknown
```

For the seeded autumn campaign scenario, the expected cause is:

```text
conversion_tracking_under_reporting
```

The human-readable explanation belongs in:

```json
"summary": "Safari conversion events are under-reported while actual purchases remain stable."
```

not in:

```json
"cause": "..."
```

## Why use canonical labels?

Canonical labels make deterministic evaluation possible.

The evaluator can compare:

```text
reported cause == expected cause
```

without using fuzzy matching or an LLM-as-judge.

This is intentional for the PoC because one goal is to show that successful tool execution and correct final diagnosis are separate concerns.

## Production direction

A production exploratory analytics agent may need more nuance than a single fixed cause label.

A more realistic future schema could be:

```json
{
  "cause_category": "tracking_issue",
  "cause_subtype": "conversion_tracking_under_reporting",
  "summary": "Safari conversion events are under-reported while actual purchases remain stable.",
  "confidence": 0.91,
  "evidence_ids": []
}
```

This would keep a controlled top-level taxonomy while allowing more specific subtypes and human-readable explanations.
