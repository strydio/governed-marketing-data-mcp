package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/strydio/governed-marketing-data-mcp/internal/authn"
	"github.com/strydio/governed-marketing-data-mcp/internal/authz"
	"github.com/strydio/governed-marketing-data-mcp/internal/domain"
	"github.com/strydio/governed-marketing-data-mcp/internal/repository"
)

const (
	ToolCampaignPerformance = "get_campaign_performance"
	ToolConversionFunnel    = "get_conversion_funnel"
	ToolTrackingHealth      = "get_tracking_health"
	ToolDataFreshness       = "get_data_freshness"
)

type CommonToolInput struct {
	Campaign  string `json:"campaign" jsonschema:"Campaign slug, for example autumn-sale"`
	Market    string `json:"market" jsonschema:"Market code, for example UK"`
	StartDate string `json:"start_date" jsonschema:"Start date in YYYY-MM-DD format"`
	EndDate   string `json:"end_date" jsonschema:"End date in YYYY-MM-DD format"`
}

type Server struct {
	mcpServer     *mcp.Server
	store         *repository.SQLiteStore
	authenticator *authn.Authenticator
	policy        authz.Policy
}

func New(store *repository.SQLiteStore, authenticator *authn.Authenticator, policy authz.Policy) *Server {
	s := &Server{store: store, authenticator: authenticator, policy: policy}
	s.mcpServer = mcp.NewServer(&mcp.Implementation{Name: "governed-marketing-data-mcp", Version: "v0.1.0"}, nil)
	s.registerTools()
	return s
}

func (s *Server) Handler() http.Handler {
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server { return s.mcpServer }, nil)
	return handler
}

func (s *Server) registerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: ToolCampaignPerformance, Description: "Return aggregate advertising-side campaign performance metrics."}, s.campaignPerformance)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: ToolConversionFunnel, Description: "Return aggregate web conversion funnel metrics."}, s.conversionFunnel)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: ToolTrackingHealth, Description: "Return sanitised aggregate tracking health findings."}, s.trackingHealth)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: ToolDataFreshness, Description: "Return source freshness for governed marketing data."}, s.dataFreshness)
}

func (s *Server) campaignPerformance(ctx context.Context, req *mcp.CallToolRequest, input CommonToolInput) (*mcp.CallToolResult, domain.CampaignPerformance, error) {
	query, err := s.authorise(ctx, req, ToolCampaignPerformance, input)
	if err != nil {
		return nil, domain.CampaignPerformance{}, err
	}
	out, err := s.store.GetCampaignPerformance(ctx, query)
	return nil, out, err
}

func (s *Server) conversionFunnel(ctx context.Context, req *mcp.CallToolRequest, input CommonToolInput) (*mcp.CallToolResult, domain.ConversionFunnel, error) {
	query, err := s.authorise(ctx, req, ToolConversionFunnel, input)
	if err != nil {
		return nil, domain.ConversionFunnel{}, err
	}
	out, err := s.store.GetConversionFunnel(ctx, query)
	return nil, out, err
}

func (s *Server) trackingHealth(ctx context.Context, req *mcp.CallToolRequest, input CommonToolInput) (*mcp.CallToolResult, domain.TrackingHealth, error) {
	query, err := s.authorise(ctx, req, ToolTrackingHealth, input)
	if err != nil {
		return nil, domain.TrackingHealth{}, err
	}
	out, err := s.store.GetTrackingHealth(ctx, query)
	return nil, out, err
}

func (s *Server) dataFreshness(ctx context.Context, req *mcp.CallToolRequest, input CommonToolInput) (*mcp.CallToolResult, domain.DataFreshness, error) {
	_, err := s.authorise(ctx, req, ToolDataFreshness, input)
	if err != nil {
		return nil, domain.DataFreshness{}, err
	}
	out, err := s.store.GetDataFreshness(ctx)
	return nil, out, err
}

func (s *Server) authorise(ctx context.Context, _ *mcp.CallToolRequest, tool string, input CommonToolInput) (domain.CommonQuery, error) {
	principal, err := authn.PrincipalFromContext(ctx)
	if err != nil {
		return domain.CommonQuery{}, fmt.Errorf("authentication failed")
	}
	query := domain.CommonQuery{Campaign: input.Campaign, Market: input.Market, StartDate: input.StartDate, EndDate: input.EndDate}
	decision := s.policy.Authorise(authz.Request{Principal: principal, Tool: tool, Query: query})
	emitAuthorisationDecision(principal.Ref, tool, query, decision)
	if !decision.Allowed {
		return domain.CommonQuery{}, fmt.Errorf("authorisation failed")
	}
	return query, nil
}

func emitAuthorisationDecision(principalRef, tool string, query domain.CommonQuery, decision authz.Decision) {
	event := map[string]any{
		"event":         "authorisation_decision",
		"principal_ref": principalRef,
		"tool":          tool,
		"target": map[string]string{
			"campaign": query.Campaign,
			"market":   query.Market,
		},
		"decision": decisionName(decision.Allowed),
		"reason":   decision.Reason,
	}
	encoded, err := json.Marshal(event)
	if err == nil {
		fmt.Println(string(encoded))
	}
}

func decisionName(allowed bool) string {
	if allowed {
		return "allow"
	}
	return "deny"
}
