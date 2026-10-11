package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewOpenAICompatibleProvider(t *testing.T) {
	provider, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{BaseURL: "https://example.com/v1/", APIKey: "key", Model: "test-model"}, nil)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	if provider.cfg.BaseURL != "https://example.com/v1" {
		t.Fatalf("base URL was not normalised: %s", provider.cfg.BaseURL)
	}
	if provider.httpClient == nil {
		t.Fatal("expected default HTTP client")
	}
}

func TestBuildOpenAICompatibleChatRequest(t *testing.T) {
	provider, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{BaseURL: "https://example.com/v1", APIKey: "secret-key", Model: "test-model"}, nil)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	req := provider.buildChatRequest(Request{
		Constraints: Constraints{Campaign: "autumn-sale", Market: "UK", StartDate: "2026-09-14", EndDate: "2026-09-20"},
		Messages: []Message{
			{Role: "user", Content: "Why did conversions fall?"},
			{Role: "assistant", ToolCalls: []ToolCall{{ID: "call-1", Name: "get_campaign_performance", Arguments: map[string]any{"campaign": "autumn-sale"}}}},
			{Role: "tool", ToolName: "get_campaign_performance", ToolCallID: "call-1", Content: `{"evidence_id":"ev-1"}`},
		},
		Tools: []ToolDefinition{{
			Name:        "get_campaign_performance",
			Description: "Return campaign performance.",
			InputSchema: map[string]any{"type": "object"},
		}},
	})

	if req.Model != "test-model" {
		t.Fatalf("unexpected model: %s", req.Model)
	}
	if len(req.Messages) != 4 {
		t.Fatalf("expected system + 3 messages, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != "system" {
		t.Fatalf("expected first message to be system, got %s", req.Messages[0].Role)
	}
	if !strings.Contains(req.Messages[0].Content, `campaign="autumn-sale"`) || !strings.Contains(req.Messages[0].Content, `market="UK"`) {
		t.Fatalf("expected system message to include constraints, got %q", req.Messages[0].Content)
	}
	if req.Messages[2].Role != "assistant" || len(req.Messages[2].ToolCalls) != 1 || req.Messages[2].ToolCalls[0].Function.Name != "get_campaign_performance" {
		t.Fatalf("assistant tool call message was not mapped correctly: %#v", req.Messages[2])
	}
	if req.Messages[3].Role != "tool" || req.Messages[3].ToolCallID != "call-1" || req.Messages[3].Name != "get_campaign_performance" {
		t.Fatalf("tool message was not mapped correctly: %#v", req.Messages[3])
	}
	if len(req.Tools) != 1 || req.Tools[0].Type != "function" || req.Tools[0].Function.Name != "get_campaign_performance" {
		t.Fatalf("tool definition was not mapped correctly: %#v", req.Tools)
	}
	if req.ToolChoice != "auto" {
		t.Fatalf("expected tool_choice auto, got %s", req.ToolChoice)
	}

	encoded, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if strings.Contains(string(encoded), "secret-key") {
		t.Fatal("request body must not contain API key")
	}
}

func TestParseOpenAICompatibleToolCallResponse(t *testing.T) {
	provider, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{BaseURL: "https://example.com/v1", APIKey: "key", Model: "model"}, nil)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	response, err := provider.parseChatResponse(openAIChatResponse{
		Choices: []openAIChoice{{Message: openAIResponseMessage{ToolCalls: []openAIToolCall{{
			ID: "call-1",
			Function: openAIToolFunction{
				Name:      "get_campaign_performance",
				Arguments: `{"campaign":"autumn-sale","market":"UK"}`,
			},
		}}}}},
		Usage: openAIUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	})
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if len(response.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %d", len(response.ToolCalls))
	}
	if response.ToolCalls[0].Name != "get_campaign_performance" || response.ToolCalls[0].Arguments["campaign"] != "autumn-sale" {
		t.Fatalf("unexpected tool call: %#v", response.ToolCalls[0])
	}
	if response.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected usage: %#v", response.Usage)
	}
}

func TestParseOpenAICompatibleFinalJSONResponse(t *testing.T) {
	provider, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{BaseURL: "https://example.com/v1", APIKey: "key", Model: "model"}, nil)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	response, err := provider.parseChatResponse(openAIChatResponse{Choices: []openAIChoice{{Message: openAIResponseMessage{Content: `{"cause":"conversion_tracking_under_reporting","evidence_ids":["ev-1"]}`}}}})
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if string(response.FinalJSON) == "" || len(response.ToolCalls) != 0 {
		t.Fatalf("unexpected final response: %#v", response)
	}
}

func TestParseOpenAICompatibleMalformedToolArguments(t *testing.T) {
	provider, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{BaseURL: "https://example.com/v1", APIKey: "key", Model: "model"}, nil)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	_, err = provider.parseChatResponse(openAIChatResponse{Choices: []openAIChoice{{Message: openAIResponseMessage{ToolCalls: []openAIToolCall{{ID: "call-1", Function: openAIToolFunction{Name: "get_campaign_performance", Arguments: `{bad json}`}}}}}}})
	if err == nil {
		t.Fatal("expected malformed arguments error")
	}
}

func TestOpenAICompatibleProviderNext(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Fatalf("unexpected model: %s", req.Model)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"cause\":\"conversion_tracking_under_reporting\",\"evidence_ids\":[\"ev-1\"]}"}}],"usage":{"prompt_tokens":3,"completion_tokens":4,"total_tokens":7}}`))
	}))
	defer server.Close()

	provider, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{BaseURL: server.URL + "/v1", APIKey: "secret-key", Model: "test-model"}, server.Client())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	response, err := provider.Next(context.Background(), Request{Messages: []Message{{Role: "user", Content: "test"}}})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if gotAuth != "Bearer secret-key" {
		t.Fatalf("unexpected auth header: %s", gotAuth)
	}
	if response.Usage.TotalTokens != 7 {
		t.Fatalf("unexpected usage: %#v", response.Usage)
	}
	if len(response.FinalJSON) == 0 {
		t.Fatal("expected final JSON")
	}
}

func TestOpenAICompatibleProviderNextStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()
	provider, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{BaseURL: server.URL, APIKey: "secret-key", Model: "test-model"}, server.Client())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	_, err = provider.Next(context.Background(), Request{Messages: []Message{{Role: "user", Content: "test"}}})
	if err == nil {
		t.Fatal("expected status error")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected response body in error, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Fatal("error must not contain API key")
	}
}

func TestNewOpenAICompatibleProviderValidation(t *testing.T) {
	cases := map[string]OpenAICompatibleConfig{
		"missing_base_url": {APIKey: "key", Model: "model"},
		"missing_api_key":  {BaseURL: "https://example.com/v1", Model: "model"},
		"missing_model":    {BaseURL: "https://example.com/v1", APIKey: "key"},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewOpenAICompatibleProvider(cfg, nil); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
