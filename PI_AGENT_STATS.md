# PiAgent Development Statistics

This document records PiAgent usage statistics during the
development of the AI Observability PoC.

## Snapshot

**Recorded:** 21 October 2026  
**Agent:** PiAgent  
**Authentication:** ChatGPT subscription

### Session Statistics

| Metric | Value |
|---|---:|
| Input tokens | 3.7M |
| Output tokens | 150K |
| Cache-read tokens | 46M |
| Latest cache hit rate | 99.9% |
| Estimated model usage cost | $43.758 |
| Context utilisation | 84.0% |
| Context window | 272K tokens |
| Context compaction | Automatic |

### Raw PiAgent Output

    ↑3.7M ↓150k R46M CH99.9% $43.758 (sub) 84.0%/272k (auto)

## Notes

- Token usage figures are cumulative for the recorded session.
- Cache reads represent previously processed input tokens reused
  across model requests.
- The cache hit rate represents the latest request, not the
  entire session.
- The estimated cost is calculated by PiAgent and does not
  necessarily represent actual subscription charges.
- Context utilisation reflects the active context window,
  rather than cumulative token consumption.
- Automatic context compaction is enabled.

## Purpose

These statistics provide a reference for the AI-assisted
development effort involved in building the PoC.

They are not measurements of the PoC's runtime performance
or its OpenTelemetry instrumentation.