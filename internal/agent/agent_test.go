package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/strydio/governed-marketing-data-mcp/internal/model"
)

type fakeTools struct {
	tools []model.ToolDefinition
}

func (f fakeTools) ListTools(context.Context) ([]model.ToolDefinition, error) { return f.tools, nil }

func (f fakeTools) CallTool(_ context.Context, name string, _ map[string]any) (ToolResult, error) {
	switch name {
	case "get_campaign_performance":
		return ToolResult{EvidenceID: "ev-campaign-uk-autumn-sale-20260914-20260920", Content: map[string]any{"evidence_id": "ev-campaign-uk-autumn-sale-20260914-20260920"}}, nil
	case "get_conversion_funnel":
		return ToolResult{EvidenceID: "ev-funnel-uk-autumn-sale-20260914-20260920", Content: map[string]any{"evidence_id": "ev-funnel-uk-autumn-sale-20260914-20260920"}}, nil
	case "get_tracking_health":
		return ToolResult{EvidenceID: "ev-tracking-incident-017", Content: map[string]any{"evidence_id": "ev-tracking-incident-017"}}, nil
	default:
		return ToolResult{}, fmt.Errorf("unknown tool")
	}
}

func toolDefs() []model.ToolDefinition {
	return []model.ToolDefinition{{Name: "get_campaign_performance"}, {Name: "get_conversion_funnel"}, {Name: "get_tracking_health"}}
}

func TestRunnerCorrectFlow(t *testing.T) {
	final := mustJSON(t, map[string]any{
		"cause":        "conversion_tracking_under_reporting",
		"confidence":   0.91,
		"summary":      "Actual purchases are stable; Safari conversion events are under-reported.",
		"evidence_ids": []string{"ev-campaign-uk-autumn-sale-20260914-20260920", "ev-funnel-uk-autumn-sale-20260914-20260920", "ev-tracking-incident-017"},
		"limitations":  []string{"The result covers the requested campaign, market, and date range only."},
	})
	runner := Runner{
		Provider: &model.FakeProvider{Responses: []model.Response{
			{ToolCalls: []model.ToolCall{{ID: "1", Name: "get_campaign_performance", Arguments: map[string]any{}}}},
			{ToolCalls: []model.ToolCall{{ID: "2", Name: "get_conversion_funnel", Arguments: map[string]any{}}}},
			{ToolCalls: []model.ToolCall{{ID: "3", Name: "get_tracking_health", Arguments: map[string]any{}}}},
			{FinalJSON: final},
		}},
		Tools: fakeTools{tools: toolDefs()},
		Now:   func() time.Time { return time.Date(2026, 9, 21, 4, 15, 0, 0, time.UTC) },
	}
	record, err := runner.Run(context.Background(), Config{Question: "Why did conversions fall?", Campaign: "autumn-sale", Market: "UK", StartDate: "2026-09-14", EndDate: "2026-09-20"})
	if err != nil {
		t.Fatalf("run agent: %v", err)
	}
	if record.RunID != "agent-20260921T041500Z" {
		t.Fatalf("unexpected run id: %s", record.RunID)
	}
	if len(record.ToolsCalled) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(record.ToolsCalled))
	}
	if record.Diagnosis.Cause != "conversion_tracking_under_reporting" {
		t.Fatalf("unexpected cause: %s", record.Diagnosis.Cause)
	}
}

func TestRunnerRejectsUnknownTool(t *testing.T) {
	runner := Runner{Provider: &model.FakeProvider{Responses: []model.Response{{ToolCalls: []model.ToolCall{{ID: "1", Name: "run_sql"}}}}}, Tools: fakeTools{tools: toolDefs()}}
	_, err := runner.Run(context.Background(), Config{Question: "test"})
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
}

func TestRunnerRejectsMalformedFinalJSON(t *testing.T) {
	runner := Runner{Provider: &model.FakeProvider{Responses: []model.Response{{FinalJSON: []byte(`{"cause":`)}}}, Tools: fakeTools{tools: toolDefs()}}
	_, err := runner.Run(context.Background(), Config{Question: "test"})
	if err == nil {
		t.Fatal("expected malformed JSON error")
	}
}

func TestRunnerEnforcesMaxToolCalls(t *testing.T) {
	runner := Runner{Provider: &model.FakeProvider{Responses: []model.Response{
		{ToolCalls: []model.ToolCall{{ID: "1", Name: "get_campaign_performance"}}},
		{ToolCalls: []model.ToolCall{{ID: "2", Name: "get_campaign_performance"}}},
	}}, Tools: fakeTools{tools: toolDefs()}}
	_, err := runner.Run(context.Background(), Config{Question: "test", MaxToolCalls: 1})
	if err == nil {
		t.Fatal("expected max tool calls error")
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return encoded
}
