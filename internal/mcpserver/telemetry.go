package mcpserver

import (
	"context"

	"github.com/strydio/governed-marketing-data-mcp/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func startToolSpan(ctx context.Context, tool string, input CommonToolInput) (context.Context, trace.Span) {
	return telemetry.Tracer().Start(ctx, "MCP tools/call "+tool,
		trace.WithAttributes(
			attribute.String(telemetry.AttrMCPMethodName, telemetry.MCPMethodToolsCall),
			attribute.String(telemetry.AttrGenAIToolName, tool),
			attribute.String(telemetry.AttrCampaignMarket, input.Market),
			attribute.String(telemetry.AttrCampaignIDHash, telemetry.CampaignIDHash(input.Campaign, input.Market)),
		),
	)
}
