package runs

import (
	"encoding/json"
	"strconv"
)

type Record struct {
	RunID       string    `json:"run_id"`
	Mode        string    `json:"mode"`
	Campaign    string    `json:"campaign"`
	Market      string    `json:"market"`
	StartDate   string    `json:"start_date"`
	EndDate     string    `json:"end_date"`
	ToolsCalled []string  `json:"tools_called"`
	EvidenceIDs []string  `json:"evidence_ids"`
	Diagnosis   Diagnosis `json:"diagnosis"`
}

type Diagnosis struct {
	Cause       string   `json:"cause"`
	Confidence  float64  `json:"confidence"`
	Summary     string   `json:"summary"`
	EvidenceIDs []string `json:"evidence_ids"`
	Limitations []string `json:"limitations"`
}

func (d *Diagnosis) UnmarshalJSON(data []byte) error {
	type rawDiagnosis struct {
		Cause       string          `json:"cause"`
		Confidence  json.RawMessage `json:"confidence"`
		Summary     string          `json:"summary"`
		EvidenceIDs []string        `json:"evidence_ids"`
		Limitations json.RawMessage `json:"limitations"`
	}
	var raw rawDiagnosis
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	confidence, err := parseFloat(raw.Confidence)
	if err != nil {
		return err
	}
	limitations, err := parseStringList(raw.Limitations)
	if err != nil {
		return err
	}

	d.Cause = raw.Cause
	d.Confidence = confidence
	d.Summary = raw.Summary
	d.EvidenceIDs = raw.EvidenceIDs
	d.Limitations = limitations
	return nil
}

func parseFloat(data json.RawMessage) (float64, error) {
	if len(data) == 0 || string(data) == "null" {
		return 0, nil
	}
	var value float64
	if err := json.Unmarshal(data, &value); err == nil {
		return value, nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return 0, err
	}
	if text == "" {
		return 0, nil
	}
	return strconv.ParseFloat(text, 64)
}

func parseStringList(data json.RawMessage) ([]string, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(data, &values); err == nil {
		return values, nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	return []string{value}, nil
}
