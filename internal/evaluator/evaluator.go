package evaluator

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/strydio/governed-marketing-data-mcp/internal/runs"
)

type GroundTruth struct {
	ScenarioID             string   `json:"scenario_id"`
	ExpectedCause          string   `json:"expected_cause"`
	RequiredEvidenceTypes  []string `json:"required_evidence_types"`
	AcceptableCauseAliases []string `json:"acceptable_cause_aliases"`
}

type Result struct {
	RunID                    string   `json:"run_id"`
	ValidSchema              bool     `json:"valid_schema"`
	CauseCorrect             bool     `json:"cause_correct"`
	EvidenceValid            bool     `json:"evidence_valid"`
	RequiredEvidenceComplete bool     `json:"required_evidence_complete"`
	MissingEvidenceTypes     []string `json:"missing_evidence_types"`
	ReportedCause            string   `json:"reported_cause"`
	ExpectedCause            string   `json:"expected_cause"`
}

func LoadGroundTruth(path string) (GroundTruth, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return GroundTruth{}, fmt.Errorf("read ground truth: %w", err)
	}
	var groundTruth GroundTruth
	if err := json.Unmarshal(contents, &groundTruth); err != nil {
		return GroundTruth{}, fmt.Errorf("parse ground truth: %w", err)
	}
	if groundTruth.ExpectedCause == "" {
		return GroundTruth{}, fmt.Errorf("ground truth expected cause is required")
	}
	return groundTruth, nil
}

func Evaluate(record runs.Record, groundTruth GroundTruth) Result {
	result := Result{
		RunID:         record.RunID,
		ReportedCause: record.Diagnosis.Cause,
		ExpectedCause: groundTruth.ExpectedCause,
		EvidenceValid: true,
	}

	result.ValidSchema = record.RunID != "" && record.Diagnosis.Cause != "" && runs.IsAllowedCause(record.Diagnosis.Cause) && len(record.Diagnosis.EvidenceIDs) > 0
	result.CauseCorrect = causeMatches(record.Diagnosis.Cause, groundTruth)
	result.EvidenceValid = citedEvidenceBelongsToRun(record.Diagnosis.EvidenceIDs, record.EvidenceIDs)

	presentTypes := map[string]bool{}
	for _, evidenceID := range record.Diagnosis.EvidenceIDs {
		if evidenceType := EvidenceType(evidenceID); evidenceType != "" {
			presentTypes[evidenceType] = true
		}
	}
	for _, required := range groundTruth.RequiredEvidenceTypes {
		if !presentTypes[required] {
			result.MissingEvidenceTypes = append(result.MissingEvidenceTypes, required)
		}
	}
	result.RequiredEvidenceComplete = len(result.MissingEvidenceTypes) == 0
	return result
}

func EvidenceType(evidenceID string) string {
	switch {
	case strings.HasPrefix(evidenceID, "ev-campaign-"):
		return "campaign_performance"
	case strings.HasPrefix(evidenceID, "ev-funnel-"):
		return "conversion_funnel"
	case strings.HasPrefix(evidenceID, "ev-tracking-"):
		return "tracking_health"
	case strings.HasPrefix(evidenceID, "ev-freshness-"):
		return "data_freshness"
	default:
		return ""
	}
}

func causeMatches(cause string, groundTruth GroundTruth) bool {
	if cause == groundTruth.ExpectedCause {
		return true
	}
	for _, alias := range groundTruth.AcceptableCauseAliases {
		if cause == alias {
			return true
		}
	}
	return false
}

func citedEvidenceBelongsToRun(cited []string, returned []string) bool {
	returnedSet := map[string]bool{}
	for _, evidenceID := range returned {
		returnedSet[evidenceID] = true
	}
	for _, evidenceID := range cited {
		if !returnedSet[evidenceID] {
			return false
		}
	}
	return true
}
