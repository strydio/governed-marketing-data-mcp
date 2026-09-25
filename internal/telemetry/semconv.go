package telemetry

const (
	TracerName = "github.com/strydio/governed-marketing-data-mcp"

	AttrMCPMethodName         = "mcp.method.name"
	AttrGenAIToolName         = "gen_ai.tool.name"
	AttrCampaignMarket        = "campaign.market"
	AttrCampaignIDHash        = "campaign.id_hash"
	AttrDataSource            = "data.source"
	AttrDataCompleteThrough   = "data.complete_through"
	AttrResultRecordCount     = "result.record_count"
	AttrAuthorisationDecision = "authorisation.decision"
	AttrAuthorisationReason   = "authorisation.reason"
	AttrEvidenceCount         = "evidence.count"
)

const (
	MCPMethodToolsCall = "tools/call"
)
