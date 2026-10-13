package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/strydio/governed-marketing-data-mcp/internal/model"
	"github.com/strydio/governed-marketing-data-mcp/internal/runs"
	"github.com/strydio/governed-marketing-data-mcp/internal/telemetry"
	"go.opentelemetry.io/otel/codes"
)

type ToolExecutor interface {
	ListTools(ctx context.Context) ([]model.ToolDefinition, error)
	CallTool(ctx context.Context, name string, arguments map[string]any) (ToolResult, error)
}

type ToolResult struct {
	EvidenceID string
	Content    any
}

type Config struct {
	Question     string
	Campaign     string
	Market       string
	StartDate    string
	EndDate      string
	MaxTurns     int
	MaxToolCalls int
}

type Runner struct {
	Provider model.Provider
	Tools    ToolExecutor
	Now      func() time.Time
}

func (r Runner) Run(ctx context.Context, cfg Config) (runs.Record, error) {
	ctx, rootSpan := telemetry.Tracer().Start(ctx, "invoke_agent campaign_investigation")
	defer rootSpan.End()
	if cfg.MaxTurns == 0 {
		cfg.MaxTurns = 12
	}
	if cfg.MaxToolCalls == 0 {
		cfg.MaxToolCalls = 8
	}
	if r.Now == nil {
		r.Now = func() time.Time { return time.Now().UTC() }
	}
	if r.Provider == nil {
		return runs.Record{}, fmt.Errorf("model provider is required")
	}
	if r.Tools == nil {
		return runs.Record{}, fmt.Errorf("tool executor is required")
	}

	tools, err := r.Tools.ListTools(ctx)
	if err != nil {
		return runs.Record{}, fmt.Errorf("list tools: %w", err)
	}
	allowedTools := map[string]bool{}
	for _, tool := range tools {
		allowedTools[tool.Name] = true
	}

	record := runs.Record{RunID: "agent-" + r.Now().UTC().Format("20060102T150405Z"), Mode: "agent", Campaign: cfg.Campaign, Market: cfg.Market, StartDate: cfg.StartDate, EndDate: cfg.EndDate}
	messages := []model.Message{{Role: "user", Content: cfg.Question}}
	toolCalls := 0

	for turn := 0; turn < cfg.MaxTurns; turn++ {
		modelCtx, modelSpan := telemetry.Tracer().Start(ctx, "model.next")
		response, err := r.Provider.Next(modelCtx, model.Request{
			Question: cfg.Question,
			Messages: messages,
			Tools:    tools,
			Constraints: model.Constraints{
				Campaign:  cfg.Campaign,
				Market:    cfg.Market,
				StartDate: cfg.StartDate,
				EndDate:   cfg.EndDate,
			},
		})
		if err != nil {
			modelSpan.SetStatus(codes.Error, "model response failed")
			modelSpan.End()
			rootSpan.SetStatus(codes.Error, "agent failed")
			return runs.Record{}, fmt.Errorf("model response: %w", err)
		}
		modelSpan.End()

		if len(response.ToolCalls) > 0 {
			messages = append(messages, model.Message{Role: "assistant", ToolCalls: response.ToolCalls})
			for _, call := range response.ToolCalls {
				if !allowedTools[call.Name] {
					return runs.Record{}, fmt.Errorf("model requested unknown tool: %s", call.Name)
				}
				toolCalls++
				if toolCalls > cfg.MaxToolCalls {
					return runs.Record{}, fmt.Errorf("maximum tool calls exceeded")
				}
				result, err := r.Tools.CallTool(ctx, call.Name, call.Arguments)
				if err != nil {
					return runs.Record{}, fmt.Errorf("call tool %s: %w", call.Name, err)
				}
				record.ToolsCalled = append(record.ToolsCalled, call.Name)
				if result.EvidenceID != "" {
					record.EvidenceIDs = append(record.EvidenceIDs, result.EvidenceID)
				}
				encoded, _ := json.Marshal(result.Content)
				messages = append(messages, model.Message{Role: "tool", ToolName: call.Name, ToolCallID: call.ID, Content: string(encoded)})
			}
			continue
		}

		if len(response.FinalJSON) > 0 {
			diagnosis, err := parseDiagnosis(response.FinalJSON)
			if err != nil {
				return runs.Record{}, fmt.Errorf("parse final diagnosis: %w", err)
			}
			if diagnosis.Cause == "" || len(diagnosis.EvidenceIDs) == 0 {
				return runs.Record{}, fmt.Errorf("final diagnosis is incomplete")
			}
			record.Diagnosis = diagnosis
			return record, nil
		}

		return runs.Record{}, fmt.Errorf("model response contained neither tool calls nor final diagnosis")
	}

	return runs.Record{}, fmt.Errorf("maximum model turns exceeded")
}

func parseDiagnosis(raw []byte) (runs.Diagnosis, error) {
	finalJSON := cleanFinalJSON(raw)
	var flexible struct {
		Cause       string          `json:"cause"`
		Confidence  json.RawMessage `json:"confidence"`
		Summary     string          `json:"summary"`
		EvidenceIDs []string        `json:"evidence_ids"`
		Limitations json.RawMessage `json:"limitations"`
	}
	if err := json.Unmarshal(finalJSON, &flexible); err != nil {
		return runs.Diagnosis{}, err
	}
	confidence, err := parseConfidence(flexible.Confidence)
	if err != nil {
		return runs.Diagnosis{}, err
	}
	limitations, err := parseLimitations(flexible.Limitations)
	if err != nil {
		return runs.Diagnosis{}, err
	}
	return runs.Diagnosis{Cause: flexible.Cause, Confidence: confidence, Summary: flexible.Summary, EvidenceIDs: flexible.EvidenceIDs, Limitations: limitations}, nil
}

func parseLimitations(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		return values, nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil, fmt.Errorf("limitations must be a string array or string")
	}
	if single == "" {
		return nil, nil
	}
	return []string{single}, nil
}

func parseConfidence(raw json.RawMessage) (float64, error) {
	if len(raw) == 0 {
		return 0, nil
	}
	var numeric float64
	if err := json.Unmarshal(raw, &numeric); err == nil {
		return numeric, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, fmt.Errorf("confidence must be a number")
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("confidence must be numeric")
	}
	return parsed, nil
}

func cleanFinalJSON(raw []byte) []byte {
	text := strings.TrimSpace(string(raw))
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimSpace(strings.TrimPrefix(text, "```json"))
		text = strings.TrimSpace(strings.TrimSuffix(text, "```"))
		return []byte(text)
	}
	if strings.HasPrefix(text, "```") {
		text = strings.TrimSpace(strings.TrimPrefix(text, "```"))
		text = strings.TrimSpace(strings.TrimSuffix(text, "```"))
		return []byte(text)
	}
	return []byte(text)
}
