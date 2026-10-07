package runs

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
