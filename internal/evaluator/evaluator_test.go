package evaluator

import (
	"testing"

	"github.com/strydio/governed-marketing-data-mcp/internal/runs"
)

func correctRecord() runs.Record {
	return runs.Record{
		RunID: "run-0001",
		EvidenceIDs: []string{
			"ev-campaign-uk-autumn-sale-20260914-20260920",
			"ev-funnel-uk-autumn-sale-20260914-20260920",
			"ev-tracking-incident-017",
			"ev-freshness-20260921T0415Z",
		},
		Diagnosis: runs.Diagnosis{
			Cause: "conversion_tracking_under_reporting",
			EvidenceIDs: []string{
				"ev-campaign-uk-autumn-sale-20260914-20260920",
				"ev-funnel-uk-autumn-sale-20260914-20260920",
				"ev-tracking-incident-017",
			},
		},
	}
}

func groundTruth() GroundTruth {
	return GroundTruth{
		ScenarioID:             "autumn-sale-tracking-gap",
		ExpectedCause:          "conversion_tracking_under_reporting",
		RequiredEvidenceTypes:  []string{"campaign_performance", "conversion_funnel", "tracking_health"},
		AcceptableCauseAliases: []string{"safari_tracking_failure", "missing_conversion_events"},
	}
}

func TestEvaluateCorrectRecord(t *testing.T) {
	result := Evaluate(correctRecord(), groundTruth())
	if !result.ValidSchema || !result.CauseCorrect || !result.EvidenceValid || !result.RequiredEvidenceComplete {
		t.Fatalf("expected correct result, got %#v", result)
	}
}

func TestEvaluateIncorrectCause(t *testing.T) {
	record := correctRecord()
	record.Diagnosis.Cause = "lower_quality_traffic"
	result := Evaluate(record, groundTruth())
	if result.CauseCorrect {
		t.Fatalf("expected cause to be incorrect: %#v", result)
	}
	if !result.ValidSchema {
		t.Fatalf("expected known-but-incorrect cause to remain schema-valid: %#v", result)
	}
	if !result.EvidenceValid || !result.RequiredEvidenceComplete {
		t.Fatalf("expected evidence checks to remain valid: %#v", result)
	}
}

func TestEvaluateUnknownCauseLabelInvalidSchema(t *testing.T) {
	record := correctRecord()
	record.Diagnosis.Cause = "Significant under-reporting of conversion events for Safari browser sessions."
	result := Evaluate(record, groundTruth())
	if result.ValidSchema {
		t.Fatalf("expected prose cause to be schema-invalid: %#v", result)
	}
	if result.CauseCorrect {
		t.Fatalf("expected prose cause not to be correct: %#v", result)
	}
}

func TestEvaluateAliasCause(t *testing.T) {
	record := correctRecord()
	record.Diagnosis.Cause = "safari_tracking_failure"
	result := Evaluate(record, groundTruth())
	if !result.CauseCorrect {
		t.Fatalf("expected alias cause to pass: %#v", result)
	}
}

func TestEvaluateFabricatedEvidence(t *testing.T) {
	record := correctRecord()
	record.Diagnosis.EvidenceIDs = append(record.Diagnosis.EvidenceIDs, "ev-fake-123")
	result := Evaluate(record, groundTruth())
	if result.EvidenceValid {
		t.Fatalf("expected fabricated evidence to fail validation: %#v", result)
	}
}

func TestEvaluateMissingRequiredEvidence(t *testing.T) {
	record := correctRecord()
	record.Diagnosis.EvidenceIDs = []string{
		"ev-campaign-uk-autumn-sale-20260914-20260920",
		"ev-funnel-uk-autumn-sale-20260914-20260920",
	}
	result := Evaluate(record, groundTruth())
	if result.RequiredEvidenceComplete {
		t.Fatalf("expected required evidence to be incomplete: %#v", result)
	}
	if len(result.MissingEvidenceTypes) != 1 || result.MissingEvidenceTypes[0] != "tracking_health" {
		t.Fatalf("unexpected missing evidence types: %#v", result.MissingEvidenceTypes)
	}
}

func TestLoadGroundTruth(t *testing.T) {
	loaded, err := LoadGroundTruth("../../testdata/ground_truth.json")
	if err != nil {
		t.Fatalf("load ground truth: %v", err)
	}
	if loaded.ExpectedCause != "conversion_tracking_under_reporting" {
		t.Fatalf("unexpected expected cause: %s", loaded.ExpectedCause)
	}
}
