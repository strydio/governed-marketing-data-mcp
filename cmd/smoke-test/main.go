package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpserver "github.com/strydio/governed-marketing-data-mcp/internal/mcpserver"
	"golang.org/x/oauth2"
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

func main() {
	ctx := context.Background()
	endpoint := envOrDefault("MCP_ENDPOINT", "http://localhost:8080/mcp")
	token := os.Getenv("DEMO_BEARER_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "DEMO_BEARER_TOKEN is required")
		os.Exit(1)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "governed-marketing-data-mcp-smoke-test", Version: "v0.1.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:             endpoint,
		OAuthHandler:         staticOAuthHandler{token: token},
		DisableStandaloneSSE: true,
	}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		fail("connect to MCP server", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		fail("list tools", err)
	}
	expectedTools := []string{mcpserver.ToolCampaignPerformance, mcpserver.ToolConversionFunnel, mcpserver.ToolTrackingHealth, mcpserver.ToolDataFreshness}
	if len(tools.Tools) != len(expectedTools) {
		failf("expected %d tools, got %d", len(expectedTools), len(tools.Tools))
	}
	seen := map[string]bool{}
	for _, tool := range tools.Tools {
		seen[tool.Name] = true
	}
	for _, name := range expectedTools {
		if !seen[name] {
			failf("missing tool %s", name)
		}
	}
	if seen["run_sql"] {
		failf("MCP server advertised an unapproved tool: run_sql")
	}
	fmt.Println("✓ MCP server advertises only the approved governed tools")

	allowedResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: mcpserver.ToolCampaignPerformance,
		Arguments: map[string]any{
			"campaign":   "autumn-sale",
			"market":     "UK",
			"start_date": "2026-09-14",
			"end_date":   "2026-09-20",
		},
	})
	if err != nil {
		fail("call authorised UK campaign performance tool", err)
	}
	if allowedResult.IsError {
		failf("authorised UK call returned tool error: %#v", allowedResult.Content)
	}
	content, ok := allowedResult.StructuredContent.(map[string]any)
	if !ok {
		failf("authorised UK call did not return structured content")
	}
	if content["evidence_id"] != "ev-campaign-uk-autumn-sale-20260914-20260920" {
		failf("unexpected evidence_id: %#v", content["evidence_id"])
	}
	fmt.Println("✓ authorised UK campaign call returned expected evidence")

	deniedResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: mcpserver.ToolCampaignPerformance,
		Arguments: map[string]any{
			"campaign":   "summer-sale",
			"market":     "US",
			"start_date": "2026-09-14",
			"end_date":   "2026-09-20",
		},
	})
	if err != nil {
		fail("call unauthorised US campaign performance tool", err)
	}
	if !deniedResult.IsError {
		failf("expected unauthorised US call to be denied")
	}
	fmt.Println("✓ unauthorised US campaign call was denied safely")

	fmt.Println("smoke test passed")
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func fail(step string, err error) {
	fmt.Fprintf(os.Stderr, "smoke test failed during %s: %v\n", step, err)
	os.Exit(1)
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "smoke test failed: "+format+"\n", args...)
	os.Exit(1)
}
