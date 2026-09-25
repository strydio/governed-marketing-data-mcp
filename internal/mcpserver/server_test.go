package mcpserver

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/strydio/governed-marketing-data-mcp/internal/authn"
	"github.com/strydio/governed-marketing-data-mcp/internal/authz"
	"github.com/strydio/governed-marketing-data-mcp/internal/repository"
	"golang.org/x/oauth2"
	_ "modernc.org/sqlite"
)

type staticOAuthHandler struct{ token string }

func (h staticOAuthHandler) TokenSource(context.Context) (oauth2.TokenSource, error) {
	if h.token == "" {
		return nil, nil
	}
	return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: h.token}), nil
}

func (h staticOAuthHandler) Authorize(context.Context, *http.Request, *http.Response) error {
	return nil
}

func testHTTPServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := repository.OpenSQLiteDB(db)
	if err := store.ApplyMigrations(ctx, "../../migrations"); err != nil {
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
	server := New(store, authenticator, policy)
	protected := mcpauth.RequireBearerToken(authenticator.VerifyToken, &mcpauth.RequireBearerTokenOptions{AllowMissingExpiration: true})(server.Handler())
	mux := http.NewServeMux()
	mux.Handle("/mcp", protected)
	return httptest.NewServer(mux)
}

func testClientSession(t *testing.T, serverURL, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:             serverURL + "/mcp",
		OAuthHandler:         staticOAuthHandler{token: token},
		DisableStandaloneSSE: true,
	}
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestHTTPMissingBearerTokenDenied(t *testing.T) {
	server := testHTTPServer(t)
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/mcp", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestHTTPInvalidBearerTokenDenied(t *testing.T) {
	server := testHTTPServer(t)
	defer server.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp", OAuthHandler: staticOAuthHandler{token: "wrong-token"}, DisableStandaloneSSE: true}
	if _, err := client.Connect(context.Background(), transport, nil); err == nil {
		t.Fatal("expected MCP connection to fail with invalid token")
	}
}

func TestHTTPToolsListShowsExactlyBusinessTools(t *testing.T) {
	server := testHTTPServer(t)
	defer server.Close()
	session := testClientSession(t, server.URL, "demo-token")

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(result.Tools) != 4 {
		t.Fatalf("expected four tools, got %d", len(result.Tools))
	}
	seen := map[string]bool{}
	for _, tool := range result.Tools {
		seen[tool.Name] = true
	}
	for _, name := range []string{ToolCampaignPerformance, ToolConversionFunnel, ToolTrackingHealth, ToolDataFreshness} {
		if !seen[name] {
			t.Fatalf("missing tool %s", name)
		}
	}
	if seen["run_sql"] {
		t.Fatal("must not expose generic SQL tool")
	}
}

func TestHTTPAuthorisedCampaignPerformance(t *testing.T) {
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
	content, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content map, got %T", result.StructuredContent)
	}
	if content["evidence_id"] != "ev-campaign-uk-autumn-sale-20260914-20260920" {
		t.Fatalf("unexpected evidence id: %#v", content["evidence_id"])
	}
}

func TestHTTPUnauthorisedMarketDeniedSafely(t *testing.T) {
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
		t.Fatal("expected tool error for unauthorised request")
	}
}
