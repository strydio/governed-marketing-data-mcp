package client

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/strydio/governed-marketing-data-mcp/internal/authn"
	"github.com/strydio/governed-marketing-data-mcp/internal/authz"
	mcpserver "github.com/strydio/governed-marketing-data-mcp/internal/mcpserver"
	"github.com/strydio/governed-marketing-data-mcp/internal/repository"
	_ "modernc.org/sqlite"
)

func testMCPServer(t *testing.T) *httptest.Server {
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

func TestRunDeterministic(t *testing.T) {
	server := testMCPServer(t)
	defer server.Close()

	record, err := RunDeterministic(context.Background(), Config{
		Endpoint:  server.URL + "/mcp",
		Token:     "demo-token",
		Campaign:  "autumn-sale",
		Market:    "UK",
		StartDate: "2026-09-14",
		EndDate:   "2026-09-20",
	})
	if err != nil {
		t.Fatalf("run deterministic client: %v", err)
	}

	wantTools := []string{mcpserver.ToolCampaignPerformance, mcpserver.ToolConversionFunnel, mcpserver.ToolTrackingHealth, mcpserver.ToolDataFreshness}
	if len(record.ToolsCalled) != len(wantTools) {
		t.Fatalf("expected %d tools called, got %d", len(wantTools), len(record.ToolsCalled))
	}
	for i, want := range wantTools {
		if record.ToolsCalled[i] != want {
			t.Fatalf("tool %d = %s, want %s", i, record.ToolsCalled[i], want)
		}
	}

	wantEvidence := []string{
		"ev-campaign-uk-autumn-sale-20260914-20260920",
		"ev-funnel-uk-autumn-sale-20260914-20260920",
		"ev-tracking-incident-017",
		"ev-freshness-20260921T0415Z",
	}
	if len(record.EvidenceIDs) != len(wantEvidence) {
		t.Fatalf("expected %d evidence IDs, got %d", len(wantEvidence), len(record.EvidenceIDs))
	}
	for i, want := range wantEvidence {
		if record.EvidenceIDs[i] != want {
			t.Fatalf("evidence %d = %s, want %s", i, record.EvidenceIDs[i], want)
		}
	}

	if record.Diagnosis.Cause != "conversion_tracking_under_reporting" {
		t.Fatalf("unexpected diagnosis cause: %s", record.Diagnosis.Cause)
	}
	if record.Diagnosis.Confidence != 0.91 {
		t.Fatalf("unexpected confidence: %f", record.Diagnosis.Confidence)
	}
}

func TestRunDeterministicRequiresToken(t *testing.T) {
	_, err := RunDeterministic(context.Background(), Config{Endpoint: "http://example.invalid/mcp"})
	if err == nil {
		t.Fatal("expected token error")
	}
}

func TestRunDeterministicDeniedTargetFails(t *testing.T) {
	server := testMCPServer(t)
	defer server.Close()

	_, err := RunDeterministic(context.Background(), Config{
		Endpoint:  server.URL + "/mcp",
		Token:     "demo-token",
		Campaign:  "summer-sale",
		Market:    "US",
		StartDate: "2026-09-14",
		EndDate:   "2026-09-20",
	})
	if err == nil {
		t.Fatal("expected denied target error")
	}
}
