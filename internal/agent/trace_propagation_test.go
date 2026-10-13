package agent

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/strydio/governed-marketing-data-mcp/internal/authn"
	"github.com/strydio/governed-marketing-data-mcp/internal/authz"
	"github.com/strydio/governed-marketing-data-mcp/internal/mcpserver"
	"github.com/strydio/governed-marketing-data-mcp/internal/model"
	"github.com/strydio/governed-marketing-data-mcp/internal/repository"
	"github.com/strydio/governed-marketing-data-mcp/internal/telemetry"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	_ "modernc.org/sqlite"
)

func TestAgentMCPTracePropagation(t *testing.T) {
	telemetry.InitTextMapPropagator()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		_ = provider.Shutdown(context.Background())
	})

	server := traceTestMCPServer(t)
	defer server.Close()

	executor, closeExecutor, err := NewMCPToolExecutor(context.Background(), server.URL+"/mcp", "demo-token")
	if err != nil {
		t.Fatalf("new MCP executor: %v", err)
	}
	defer closeExecutor()

	final := []byte(`{"cause":"conversion_tracking_under_reporting","confidence":0.91,"summary":"ok","evidence_ids":["ev-campaign-uk-autumn-sale-20260914-20260920"],"limitations":[]}`)
	runner := Runner{Provider: &model.FakeProvider{Responses: []model.Response{
		{ToolCalls: []model.ToolCall{{ID: "call-1", Name: mcpserver.ToolCampaignPerformance, Arguments: map[string]any{"campaign": "autumn-sale", "market": "UK", "start_date": "2026-09-14", "end_date": "2026-09-20"}}}},
		{FinalJSON: final},
	}}, Tools: executor}

	_, err = runner.Run(context.Background(), Config{Question: "Why did conversions fall?", Campaign: "autumn-sale", Market: "UK", StartDate: "2026-09-14", EndDate: "2026-09-20"})
	if err != nil {
		t.Fatalf("run agent: %v", err)
	}

	spans := recorder.Ended()
	root := findTraceSpan(t, spans, "invoke_agent campaign_investigation")
	execute := findTraceSpan(t, spans, "execute_tool get_campaign_performance")
	serverSpan := findTraceSpan(t, spans, "MCP tools/call get_campaign_performance")
	dbSpan := findTraceSpan(t, spans, "db.query campaign_performance")

	rootTraceID := root.SpanContext().TraceID()
	for _, span := range []sdktrace.ReadOnlySpan{execute, serverSpan, dbSpan} {
		if span.SpanContext().TraceID() != rootTraceID {
			t.Fatalf("span %q trace ID = %s, want %s", span.Name(), span.SpanContext().TraceID(), rootTraceID)
		}
	}
	if serverSpan.Parent().SpanID() != execute.SpanContext().SpanID() {
		t.Fatalf("server span parent = %s, want execute_tool span %s", serverSpan.Parent().SpanID(), execute.SpanContext().SpanID())
	}
}

func traceTestMCPServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := repository.OpenSQLiteDB(db)
	if err := store.ApplyMigrations(context.Background(), "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	authenticator, err := authn.NewAuthenticator("demo-token")
	if err != nil {
		t.Fatalf("new authenticator: %v", err)
	}
	policy, err := authz.LoadPolicy("../../config/policy.demo.yaml")
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	server := mcpserver.New(store, authenticator, policy)
	protected := mcpauth.RequireBearerToken(authenticator.VerifyToken, &mcpauth.RequireBearerTokenOptions{AllowMissingExpiration: true})(server.Handler())
	mux := http.NewServeMux()
	mux.Handle("/mcp", protected)
	return httptest.NewServer(mux)
}

func findTraceSpan(t *testing.T, spans []sdktrace.ReadOnlySpan, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	for _, span := range spans {
		if span.Name() == name {
			return span
		}
	}
	t.Fatalf("span %q not found", name)
	return nil
}
