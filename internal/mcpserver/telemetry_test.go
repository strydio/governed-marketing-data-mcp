package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/strydio/governed-marketing-data-mcp/internal/telemetry"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func withSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		_ = provider.Shutdown(context.Background())
	})
	return recorder
}

func TestTelemetryAllowedHTTPCallProducesExpectedSpans(t *testing.T) {
	recorder := withSpanRecorder(t)
	server := testHTTPServer(t)
	defer server.Close()
	session := testClientSession(t, server.URL, "demo-token")

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: ToolCampaignPerformance,
		Arguments: map[string]any{
			"campaign":   "autumn-sale",
			"market":     "UK",
			"start_date": "2026-09-14",
			"end_date":   "2026-09-20",
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %#v", result.Content)
	}

	spans := recorder.Ended()
	assertSpanExists(t, spans, "MCP tools/call get_campaign_performance")
	assertSpanExists(t, spans, "authorisation.check")
	assertSpanExists(t, spans, "db.query campaign_performance")

	authzSpan := findSpan(t, spans, "authorisation.check")
	assertAttr(t, authzSpan, telemetry.AttrAuthorisationDecision, "allow")
	assertAttr(t, authzSpan, telemetry.AttrAuthorisationReason, "policy_match")

	toolSpan := findSpan(t, spans, "MCP tools/call get_campaign_performance")
	assertAttr(t, toolSpan, telemetry.AttrMCPMethodName, telemetry.MCPMethodToolsCall)
	assertAttr(t, toolSpan, telemetry.AttrGenAIToolName, ToolCampaignPerformance)
	assertAttr(t, toolSpan, telemetry.AttrCampaignMarket, "UK")
	assertAttr(t, toolSpan, telemetry.AttrEvidenceCount, int64(1))
}

func TestTelemetryDeniedHTTPCallHasNoDBSpan(t *testing.T) {
	recorder := withSpanRecorder(t)
	server := testHTTPServer(t)
	defer server.Close()
	session := testClientSession(t, server.URL, "demo-token")

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: ToolCampaignPerformance,
		Arguments: map[string]any{
			"campaign":   "summer-sale",
			"market":     "US",
			"start_date": "2026-09-14",
			"end_date":   "2026-09-20",
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error")
	}

	spans := recorder.Ended()
	assertSpanExists(t, spans, "MCP tools/call get_campaign_performance")
	assertSpanExists(t, spans, "authorisation.check")
	assertSpanAbsent(t, spans, "db.query campaign_performance")

	authzSpan := findSpan(t, spans, "authorisation.check")
	assertAttr(t, authzSpan, telemetry.AttrAuthorisationDecision, "deny")
	assertAttr(t, authzSpan, telemetry.AttrAuthorisationReason, "market_not_allowed")
}

func TestTelemetryDoesNotCaptureForbiddenContent(t *testing.T) {
	recorder := withSpanRecorder(t)
	server := testHTTPServer(t)
	defer server.Close()
	session := testClientSession(t, server.URL, "demo-token")

	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: ToolCampaignPerformance,
		Arguments: map[string]any{
			"campaign":   "autumn-sale",
			"market":     "UK",
			"start_date": "2026-09-14",
			"end_date":   "2026-09-20",
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}

	for _, span := range recorder.Ended() {
		text := span.Name()
		for _, attr := range span.Attributes() {
			text += string(attr.Key) + attr.Value.Emit()
		}
		for _, forbidden := range []string{"demo-token", "autumn-sale", "SELECT ", "campaign.db", "Purchase conversion events"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("span %q contains forbidden content %q", span.Name(), forbidden)
			}
		}
	}
}

func assertSpanExists(t *testing.T, spans []sdktrace.ReadOnlySpan, name string) {
	t.Helper()
	for _, span := range spans {
		if span.Name() == name {
			return
		}
	}
	t.Fatalf("expected span %q", name)
}

func assertSpanAbsent(t *testing.T, spans []sdktrace.ReadOnlySpan, name string) {
	t.Helper()
	for _, span := range spans {
		if span.Name() == name {
			t.Fatalf("did not expect span %q", name)
		}
	}
}

func findSpan(t *testing.T, spans []sdktrace.ReadOnlySpan, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	for _, span := range spans {
		if span.Name() == name {
			return span
		}
	}
	t.Fatalf("span %q not found", name)
	return nil
}

func assertAttr(t *testing.T, span sdktrace.ReadOnlySpan, key string, want any) {
	t.Helper()
	for _, attr := range span.Attributes() {
		if string(attr.Key) != key {
			continue
		}
		switch v := want.(type) {
		case string:
			if attr.Value.AsString() != v {
				t.Fatalf("attribute %s = %q, want %q", key, attr.Value.AsString(), v)
			}
		case int64:
			if attr.Value.AsInt64() != v {
				t.Fatalf("attribute %s = %d, want %d", key, attr.Value.AsInt64(), v)
			}
		}
		return
	}
	t.Fatalf("attribute %s not found on span %s", key, span.Name())
}
