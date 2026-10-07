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
	"github.com/strydio/governed-marketing-data-mcp/internal/repository"
	_ "modernc.org/sqlite"
)

func testAgentMCPServer(t *testing.T) *httptest.Server {
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

func TestMCPToolExecutor(t *testing.T) {
	server := testAgentMCPServer(t)
	defer server.Close()

	executor, closeExecutor, err := NewMCPToolExecutor(context.Background(), server.URL+"/mcp", "demo-token")
	if err != nil {
		t.Fatalf("new MCP tool executor: %v", err)
	}
	defer closeExecutor()

	tools, err := executor.ListTools(context.Background())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) != 4 {
		t.Fatalf("expected four tools, got %d", len(tools))
	}

	result, err := executor.CallTool(context.Background(), mcpserver.ToolCampaignPerformance, map[string]any{
		"campaign":   "autumn-sale",
		"market":     "UK",
		"start_date": "2026-09-14",
		"end_date":   "2026-09-20",
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.EvidenceID != "ev-campaign-uk-autumn-sale-20260914-20260920" {
		t.Fatalf("unexpected evidence id: %s", result.EvidenceID)
	}
}

func TestMCPToolExecutorDeniedToolCall(t *testing.T) {
	server := testAgentMCPServer(t)
	defer server.Close()

	executor, closeExecutor, err := NewMCPToolExecutor(context.Background(), server.URL+"/mcp", "demo-token")
	if err != nil {
		t.Fatalf("new MCP tool executor: %v", err)
	}
	defer closeExecutor()

	_, err = executor.CallTool(context.Background(), mcpserver.ToolCampaignPerformance, map[string]any{
		"campaign":   "summer-sale",
		"market":     "US",
		"start_date": "2026-09-14",
		"end_date":   "2026-09-20",
	})
	if err == nil {
		t.Fatal("expected denied tool call error")
	}
}
