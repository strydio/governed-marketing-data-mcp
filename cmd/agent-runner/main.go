package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/strydio/governed-marketing-data-mcp/internal/agent"
	"github.com/strydio/governed-marketing-data-mcp/internal/mcpserver"
	"github.com/strydio/governed-marketing-data-mcp/internal/model"
)

func main() {
	var endpoint string
	var token string
	var outputPath string
	var campaign string
	var market string
	var startDate string
	var endDate string
	var provider string

	flag.StringVar(&endpoint, "endpoint", envOrDefault("MCP_ENDPOINT", "http://localhost:8080/mcp"), "MCP Streamable HTTP endpoint")
	flag.StringVar(&token, "token", os.Getenv("DEMO_BEARER_TOKEN"), "bearer token for the MCP server")
	flag.StringVar(&outputPath, "output", "", "optional path to write the run record JSON")
	flag.StringVar(&campaign, "campaign", "autumn-sale", "campaign slug")
	flag.StringVar(&market, "market", "UK", "market code")
	flag.StringVar(&startDate, "start-date", "2026-09-14", "start date in YYYY-MM-DD format")
	flag.StringVar(&endDate, "end-date", "2026-09-20", "end date in YYYY-MM-DD format")
	flag.StringVar(&provider, "provider", envOrDefault("MODEL_PROVIDER", "fake"), "model provider: fake or openai_compatible")
	flag.Parse()

	ctx := context.Background()
	tools, closeTools, err := agent.NewMCPToolExecutor(ctx, endpoint, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create MCP tool executor: %v\n", err)
		os.Exit(1)
	}
	defer closeTools()

	modelProvider, err := buildModelProvider(provider, campaign, market, startDate, endDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create model provider: %v\n", err)
		os.Exit(1)
	}

	runner := agent.Runner{Provider: modelProvider, Tools: tools}
	record, err := runner.Run(ctx, agent.Config{
		Question:  "Why did conversions from the autumn campaign fall this week?",
		Campaign:  campaign,
		Market:    market,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "run agent: %v\n", err)
		os.Exit(1)
	}

	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode run record: %v\n", err)
		os.Exit(1)
	}
	if outputPath != "" {
		if err := os.WriteFile(outputPath, append(encoded, '\n'), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote run record to %s\n", outputPath)
		return
	}
	fmt.Println(string(encoded))
}

func buildModelProvider(provider, campaign, market, startDate, endDate string) (model.Provider, error) {
	switch provider {
	case "fake":
		return fakeScenarioProvider(campaign, market, startDate, endDate), nil
	case "openai_compatible":
		return model.NewOpenAICompatibleProvider(model.OpenAICompatibleConfig{
			BaseURL: os.Getenv("MODEL_BASE_URL"),
			APIKey:  os.Getenv("MODEL_API_KEY"),
			Model:   os.Getenv("MODEL_NAME"),
		}, nil)
	default:
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
}

func fakeScenarioProvider(campaign, market, startDate, endDate string) model.Provider {
	args := map[string]any{"campaign": campaign, "market": market, "start_date": startDate, "end_date": endDate}
	final := []byte(`{
  "cause": "conversion_tracking_under_reporting",
  "confidence": 0.91,
  "summary": "Actual purchases are stable; Safari conversion events are under-reported.",
  "evidence_ids": [
    "ev-campaign-uk-autumn-sale-20260914-20260920",
    "ev-funnel-uk-autumn-sale-20260914-20260920",
    "ev-tracking-incident-017"
  ],
  "limitations": ["The result covers the requested campaign, market, and date range only."]
}`)
	return &model.FakeProvider{Responses: []model.Response{
		{ToolCalls: []model.ToolCall{{ID: "call-1", Name: mcpserver.ToolCampaignPerformance, Arguments: args}}},
		{ToolCalls: []model.ToolCall{{ID: "call-2", Name: mcpserver.ToolConversionFunnel, Arguments: args}}},
		{ToolCalls: []model.ToolCall{{ID: "call-3", Name: mcpserver.ToolTrackingHealth, Arguments: args}}},
		{ToolCalls: []model.ToolCall{{ID: "call-4", Name: mcpserver.ToolDataFreshness, Arguments: args}}},
		{FinalJSON: final},
	}}
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
