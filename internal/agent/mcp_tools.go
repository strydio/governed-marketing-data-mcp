package agent

import (
	"context"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/strydio/governed-marketing-data-mcp/internal/model"
	"github.com/strydio/governed-marketing-data-mcp/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/oauth2"
)

type MCPToolExecutor struct {
	session *mcp.ClientSession
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

func NewMCPToolExecutor(ctx context.Context, endpoint, token string) (*MCPToolExecutor, func() error, error) {
	if endpoint == "" {
		return nil, nil, fmt.Errorf("MCP endpoint is required")
	}
	if token == "" {
		return nil, nil, fmt.Errorf("bearer token is required")
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "governed-marketing-data-mcp-agent-runner", Version: "v0.1.0"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: endpoint, OAuthHandler: staticOAuthHandler{token: token}, DisableStandaloneSSE: true}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("connect MCP client: %w", err)
	}
	return &MCPToolExecutor{session: session}, session.Close, nil
}

func (e *MCPToolExecutor) ListTools(ctx context.Context) ([]model.ToolDefinition, error) {
	result, err := e.session.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list MCP tools: %w", err)
	}
	tools := make([]model.ToolDefinition, 0, len(result.Tools))
	for _, tool := range result.Tools {
		tools = append(tools, model.ToolDefinition{Name: tool.Name, Description: tool.Description, InputSchema: tool.InputSchema})
	}
	return tools, nil
}

func (e *MCPToolExecutor) CallTool(ctx context.Context, name string, arguments map[string]any) (ToolResult, error) {
	ctx, span := telemetry.Tracer().Start(ctx, "execute_tool "+name)
	defer span.End()
	span.SetAttributes(attribute.String(telemetry.AttrGenAIToolName, name))
	meta := telemetry.InjectTraceContext(ctx, nil)
	result, err := e.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments, Meta: meta})
	if err != nil {
		span.SetStatus(codes.Error, "MCP tool call failed")
		return ToolResult{}, err
	}
	if result.IsError {
		span.SetStatus(codes.Error, "MCP tool returned an error")
		return ToolResult{}, fmt.Errorf("MCP tool returned an error")
	}
	span.SetAttributes(attribute.Int(telemetry.AttrEvidenceCount, 1))
	evidenceID, _ := evidenceID(result.StructuredContent)
	return ToolResult{EvidenceID: evidenceID, Content: result.StructuredContent}, nil
}

func evidenceID(content any) (string, bool) {
	object, ok := content.(map[string]any)
	if !ok {
		return "", false
	}
	value, ok := object["evidence_id"].(string)
	return value, ok
}
