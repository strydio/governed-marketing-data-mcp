package model

import (
	"context"
	"encoding/json"
)

type Provider interface {
	Next(ctx context.Context, req Request) (Response, error)
}

type Request struct {
	Question    string
	Messages    []Message
	Tools       []ToolDefinition
	Constraints Constraints
}

type Constraints struct {
	Campaign  string
	Market    string
	StartDate string
	EndDate   string
}

type Message struct {
	Role       string
	Content    string
	ToolName   string
	ToolCallID string
}

type ToolDefinition struct {
	Name        string
	Description string
	InputSchema any
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

type Response struct {
	ToolCalls []ToolCall
	FinalJSON json.RawMessage
	Usage     Usage
}

type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}
