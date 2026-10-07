package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/strydio/governed-marketing-data-mcp/internal/model"
	"github.com/strydio/governed-marketing-data-mcp/internal/runs"
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
		response, err := r.Provider.Next(ctx, model.Request{
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
			return runs.Record{}, fmt.Errorf("model response: %w", err)
		}

		if len(response.ToolCalls) > 0 {
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
			var diagnosis runs.Diagnosis
			if err := json.Unmarshal(response.FinalJSON, &diagnosis); err != nil {
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
