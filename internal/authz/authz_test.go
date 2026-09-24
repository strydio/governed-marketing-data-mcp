package authz

import (
	"testing"

	"github.com/strydio/governed-marketing-data-mcp/internal/authn"
	"github.com/strydio/governed-marketing-data-mcp/internal/domain"
)

func demoPolicy() Policy {
	return Policy{Principals: map[string]PrincipalPolicy{
		"uk_marketing_demo": {
			Role:              "marketing_analyst",
			AllowedMarkets:    []string{"UK"},
			AllowedCampaigns:  []string{"autumn-sale"},
			AllowedTools:      []string{"get_campaign_performance", "get_conversion_funnel", "get_tracking_health", "get_data_freshness"},
			MaxDateRangeDays:  31,
			CustomerLevelData: false,
		},
	}}
}

func validRequest() Request {
	return Request{
		Principal: authn.Principal{Ref: "uk_marketing_demo"},
		Tool:      "get_campaign_performance",
		Query:     domain.CommonQuery{Campaign: "autumn-sale", Market: "UK", StartDate: "2026-09-14", EndDate: "2026-09-20"},
	}
}

func TestAuthoriseAllowsValidRequest(t *testing.T) {
	decision := demoPolicy().Authorise(validRequest())
	if !decision.Allowed || decision.Reason != "policy_match" {
		t.Fatalf("expected allow, got %#v", decision)
	}
}

func TestAuthoriseDenials(t *testing.T) {
	cases := map[string]func(Request) Request{
		"unknown_tool": func(req Request) Request { req.Tool = "run_sql"; return req },
		"us_market": func(req Request) Request {
			req.Query.Market = "US"
			return req
		},
		"unknown_campaign": func(req Request) Request {
			req.Query.Campaign = "winter-sale"
			return req
		},
		"overlong_date_range": func(req Request) Request {
			req.Query.StartDate = "2026-09-01"
			req.Query.EndDate = "2026-10-15"
			return req
		},
		"sql_campaign": func(req Request) Request {
			req.Query.Campaign = "autumn-sale;drop-table"
			return req
		},
		"customer_level": func(req Request) Request {
			req.CustomerLevelData = true
			return req
		},
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			decision := demoPolicy().Authorise(mutate(validRequest()))
			if decision.Allowed {
				t.Fatalf("expected denial, got %#v", decision)
			}
		})
	}
}

func TestLoadPolicy(t *testing.T) {
	policy, err := LoadPolicy("../../config/policy.demo.yaml")
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if len(policy.Principals) != 1 {
		t.Fatalf("expected one principal, got %d", len(policy.Principals))
	}
}
