package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/strydio/governed-marketing-data-mcp/internal/runs"
)

type openAIChatRequest struct {
	Model      string          `json:"model"`
	Messages   []openAIMessage `json:"messages"`
	Tools      []openAITool    `json:"tools,omitempty"`
	ToolChoice string          `json:"tool_choice,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Message openAIResponseMessage `json:"message"`
}

type openAIResponseMessage struct {
	Content   string           `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAICompatibleConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

type OpenAICompatibleProvider struct {
	cfg        OpenAICompatibleConfig
	httpClient *http.Client
}

func NewOpenAICompatibleProvider(cfg OpenAICompatibleConfig, httpClient *http.Client) (*OpenAICompatibleProvider, error) {
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("MODEL_BASE_URL is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("MODEL_API_KEY is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("MODEL_NAME is required")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &OpenAICompatibleProvider{cfg: cfg, httpClient: httpClient}, nil
}

func (p *OpenAICompatibleProvider) Next(ctx context.Context, req Request) (Response, error) {
	chatReq := p.buildChatRequest(req)
	body, err := json.Marshal(chatReq)
	if err != nil {
		return Response{}, fmt.Errorf("marshal model request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("create model request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := p.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("call model provider: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return Response{}, fmt.Errorf("model provider returned status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var chatResponse openAIChatResponse
	if err := json.NewDecoder(res.Body).Decode(&chatResponse); err != nil {
		return Response{}, fmt.Errorf("decode model response: %w", err)
	}
	return p.parseChatResponse(chatResponse)
}

func (p *OpenAICompatibleProvider) buildChatRequest(req Request) openAIChatRequest {
	messages := make([]openAIMessage, 0, len(req.Messages)+1)
	messages = append(messages, openAIMessage{Role: "system", Content: systemInstruction(req.Constraints)})
	for _, message := range req.Messages {
		mapped := openAIMessage{Role: message.Role, Content: message.Content}
		if message.Role == "assistant" && len(message.ToolCalls) > 0 {
			mapped.ToolCalls = make([]openAIToolCall, 0, len(message.ToolCalls))
			for _, call := range message.ToolCalls {
				arguments, _ := json.Marshal(call.Arguments)
				mapped.ToolCalls = append(mapped.ToolCalls, openAIToolCall{ID: call.ID, Type: "function", Function: openAIToolFunction{Name: call.Name, Arguments: string(arguments)}})
			}
		}
		if message.Role == "tool" {
			mapped.ToolCallID = message.ToolCallID
			mapped.Name = message.ToolName
		}
		messages = append(messages, mapped)
	}

	tools := make([]openAITool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		tools = append(tools, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	chatReq := openAIChatRequest{Model: p.cfg.Model, Messages: messages, Tools: tools}
	if len(tools) > 0 {
		chatReq.ToolChoice = "auto"
	}
	return chatReq
}

func (p *OpenAICompatibleProvider) parseChatResponse(chatResponse openAIChatResponse) (Response, error) {
	if len(chatResponse.Choices) == 0 {
		return Response{}, fmt.Errorf("model response contained no choices")
	}
	message := chatResponse.Choices[0].Message
	response := Response{Usage: Usage{InputTokens: chatResponse.Usage.PromptTokens, OutputTokens: chatResponse.Usage.CompletionTokens, TotalTokens: chatResponse.Usage.TotalTokens}}
	if len(message.ToolCalls) > 0 {
		for _, toolCall := range message.ToolCalls {
			arguments := map[string]any{}
			if toolCall.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
					return Response{}, fmt.Errorf("parse tool arguments for %s: %w", toolCall.Function.Name, err)
				}
			}
			response.ToolCalls = append(response.ToolCalls, ToolCall{ID: toolCall.ID, Name: toolCall.Function.Name, Arguments: arguments})
		}
		return response, nil
	}
	if strings.TrimSpace(message.Content) == "" {
		return Response{}, fmt.Errorf("model response contained no tool calls or final content")
	}
	response.FinalJSON = json.RawMessage(message.Content)
	return response, nil
}

func systemInstruction(constraints Constraints) string {
	instruction := "You are a marketing analytics investigation agent. Use only the provided MCP tools. Return the final answer as raw JSON only, with no Markdown fences or prose. The JSON object must contain cause, confidence, summary, evidence_ids, and limitations. The cause field must be a short canonical label, not a sentence. Allowed cause labels are: " + strings.Join(runs.AllowedCauses(), ", ") + ". If the evidence shows reported conversions fell while purchases were stable and tracking health shows missing Safari conversion events, use cause " + runs.CauseConversionTrackingUnderReporting + ". Put explanatory prose in summary, not cause. confidence must be a JSON number between 0 and 1, for example 0.91, not a string or word. limitations must be an array of strings. Cite only evidence IDs returned by tools."
	if constraints.Campaign == "" && constraints.Market == "" && constraints.StartDate == "" && constraints.EndDate == "" {
		return instruction
	}
	return fmt.Sprintf("%s Investigation constraints: campaign=%q, market=%q, start_date=%q, end_date=%q. Use these exact values when calling tools.", instruction, constraints.Campaign, constraints.Market, constraints.StartDate, constraints.EndDate)
}
