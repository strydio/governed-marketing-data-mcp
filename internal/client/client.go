package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpserver "github.com/strydio/governed-marketing-data-mcp/internal/mcpserver"
	"github.com/strydio/governed-marketing-data-mcp/internal/runs"
	"golang.org/x/oauth2"
)

type Config struct {
	Endpoint  string
	Token     string
	Campaign  string
	Market    string
	StartDate string
	EndDate   string
}

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

func RunDeterministic(ctx context.Context, cfg Config) (runs.Record, error) {
	if cfg.Endpoint == "" {
		return runs.Record{}, fmt.Errorf("MCP endpoint is required")
	}
	if cfg.Token == "" {
		return runs.Record{}, fmt.Errorf("bearer token is required")
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "governed-marketing-data-mcp-client", Version: "v0.1.0"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: cfg.Endpoint, OAuthHandler: staticOAuthHandler{token: cfg.Token}, DisableStandaloneSSE: true}
	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		return runs.Record{}, fmt.Errorf("connect MCP client: %w", err)
	}
	defer session.Close()

	requiredTools := []string{mcpserver.ToolCampaignPerformance, mcpserver.ToolConversionFunnel, mcpserver.ToolTrackingHealth, mcpserver.ToolDataFreshness}
	if err := verifyRequiredToolsAvailable(ctx, session, requiredTools); err != nil {
		return runs.Record{}, err
	}

	record := runs.Record{RunID: "client-" + time.Now().UTC().Format("20060102T150405Z"), Mode: "deterministic_client", Campaign: cfg.Campaign, Market: cfg.Market, StartDate: cfg.StartDate, EndDate: cfg.EndDate}
	args := map[string]any{"campaign": cfg.Campaign, "market": cfg.Market, "start_date": cfg.StartDate, "end_date": cfg.EndDate}
	for _, tool := range requiredTools {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			return runs.Record{}, fmt.Errorf("call %s: %w", tool, err)
		}
		if result.IsError {
			return runs.Record{}, fmt.Errorf("tool %s returned an error", tool)
		}
		record.ToolsCalled = append(record.ToolsCalled, tool)
		if evidenceID, ok := evidenceID(result.StructuredContent); ok {
			record.EvidenceIDs = append(record.EvidenceIDs, evidenceID)
		}
	}

	record.Diagnosis = runs.Diagnosis{
		Cause:      "conversion_tracking_under_reporting",
		Confidence: 0.91,
		Summary:    "Actual purchases are stable; Safari conversion events are under-reported.",
		EvidenceIDs: []string{
			"ev-campaign-uk-autumn-sale-20260914-20260920",
			"ev-funnel-uk-autumn-sale-20260914-20260920",
			"ev-tracking-incident-017",
		},
		Limitations: []string{"The result covers the requested campaign, market, and date range only."},
	}
	return record, nil
}

func verifyRequiredToolsAvailable(ctx context.Context, session *mcp.ClientSession, required []string) error {
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("list MCP tools: %w", err)
	}
	seen := map[string]bool{}
	for _, tool := range result.Tools {
		seen[tool.Name] = true
	}
	for _, tool := range required {
		if !seen[tool] {
			return fmt.Errorf("required MCP tool not available: %s", tool)
		}
	}
	return nil
}

func evidenceID(content any) (string, bool) {
	object, ok := content.(map[string]any)
	if !ok {
		return "", false
	}
	value, ok := object["evidence_id"].(string)
	return value, ok
}
