# Deterministic evaluator

## Purpose

The evaluator checks a completed run record against known ground truth for the seeded scenario.

It does not call the MCP server and does not use an LLM.

```text
RunRecord + GroundTruth -> EvaluationResult
```

## Inputs

### Run record

A run record is produced by the deterministic MCP client:

```bash
DEMO_BEARER_TOKEN=replace-me go run ./cmd/client --output artifacts/runs/client-run.json
```

It describes what happened during a run:

- tools called;
- evidence IDs returned;
- final diagnosis.

### Ground truth

Ground truth is the known correct answer for the seeded scenario:

```text
testdata/ground_truth.json
```

It defines:

- expected cause;
- accepted cause aliases;
- required evidence types.

## Command

Evaluate a saved run:

```bash
go run ./cmd/evaluator --input artifacts/runs/client-run.json
```

Or with Make:

```bash
make run-evaluator INPUT=artifacts/runs/client-run.json
```

Optional custom ground truth:

```bash
go run ./cmd/evaluator \
  --input artifacts/runs/client-run.json \
  --ground-truth ./testdata/ground_truth.json
```

## Evaluation checks

The evaluator checks:

1. The run has a valid diagnosis shape.
2. The reported cause matches the expected cause or an accepted alias.
3. Every cited evidence ID was actually returned during the run.
4. The cited evidence covers all required evidence types.

Required evidence types for the seeded scenario:

```text
campaign_performance
conversion_funnel
tracking_health
```

Evidence type mapping:

```text
ev-campaign-*  -> campaign_performance
ev-funnel-*    -> conversion_funnel
ev-tracking-*  -> tracking_health
ev-freshness-* -> data_freshness
```

## Example result

```json
{
  "run_id": "client-20260921T041500Z",
  "valid_schema": true,
  "cause_correct": true,
  "evidence_valid": true,
  "required_evidence_complete": true,
  "missing_evidence_types": [],
  "reported_cause": "conversion_tracking_under_reporting",
  "expected_cause": "conversion_tracking_under_reporting"
}
```
