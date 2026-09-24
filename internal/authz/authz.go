package authz

import (
	"fmt"
	"os"

	"github.com/strydio/governed-marketing-data-mcp/internal/authn"
	"github.com/strydio/governed-marketing-data-mcp/internal/domain"
	"gopkg.in/yaml.v3"
)

type Policy struct {
	Principals map[string]PrincipalPolicy `yaml:"principals"`
}

type PrincipalPolicy struct {
	Role              string   `yaml:"role"`
	AllowedMarkets    []string `yaml:"allowed_markets"`
	AllowedCampaigns  []string `yaml:"allowed_campaigns"`
	AllowedTools      []string `yaml:"allowed_tools"`
	MaxDateRangeDays  int      `yaml:"max_date_range_days"`
	CustomerLevelData bool     `yaml:"customer_level_data"`
}

type Request struct {
	Principal         authn.Principal
	Tool              string
	Query             domain.CommonQuery
	CustomerLevelData bool
}

type Decision struct {
	Allowed bool
	Reason  string
}

func LoadPolicy(path string) (Policy, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read policy: %w", err)
	}
	var policy Policy
	if err := yaml.Unmarshal(contents, &policy); err != nil {
		return Policy{}, fmt.Errorf("parse policy: %w", err)
	}
	if len(policy.Principals) == 0 {
		return Policy{}, fmt.Errorf("policy has no principals")
	}
	return policy, nil
}

func (p Policy) Authorise(req Request) Decision {
	principalPolicy, ok := p.Principals[req.Principal.Ref]
	if !ok {
		return Decision{Allowed: false, Reason: "unknown_principal"}
	}
	if !contains(principalPolicy.AllowedTools, req.Tool) {
		return Decision{Allowed: false, Reason: "tool_not_allowed"}
	}
	if err := domain.ValidateCommonQuery(req.Query, principalPolicy.MaxDateRangeDays); err != nil {
		return Decision{Allowed: false, Reason: "invalid_request"}
	}
	if !contains(principalPolicy.AllowedMarkets, req.Query.Market) {
		return Decision{Allowed: false, Reason: "market_not_allowed"}
	}
	if !contains(principalPolicy.AllowedCampaigns, req.Query.Campaign) {
		return Decision{Allowed: false, Reason: "campaign_not_allowed"}
	}
	if req.CustomerLevelData || !principalPolicy.CustomerLevelData && req.CustomerLevelData {
		return Decision{Allowed: false, Reason: "customer_level_data_not_allowed"}
	}
	return Decision{Allowed: true, Reason: "policy_match"}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
