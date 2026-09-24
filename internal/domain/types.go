package domain

import "time"

type QueryPeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type CommonQuery struct {
	Campaign  string
	Market    string
	StartDate string
	EndDate   string
}

type CampaignPerformance struct {
	EvidenceID            string                     `json:"evidence_id"`
	Source                string                     `json:"source"`
	Campaign              string                     `json:"campaign"`
	Market                string                     `json:"market"`
	Period                QueryPeriod                `json:"period"`
	Metrics               CampaignPerformanceMetrics `json:"metrics"`
	Comparison            CampaignPerformanceChange  `json:"comparison"`
	AttributionModel      string                     `json:"attribution_model"`
	AttributionWindowDays int                        `json:"attribution_window_days"`
	DataCompleteThrough   time.Time                  `json:"data_complete_through"`
}

type CampaignPerformanceMetrics struct {
	SpendGBP              string `json:"spend_gbp"`
	Impressions           int    `json:"impressions"`
	Clicks                int    `json:"clicks"`
	AttributedConversions int    `json:"attributed_conversions"`
}

type CampaignPerformanceChange struct {
	Period                         string  `json:"period"`
	AttributedConversionsChangePct float64 `json:"attributed_conversions_change_pct"`
}

type ConversionFunnel struct {
	EvidenceID          string                 `json:"evidence_id"`
	Source              string                 `json:"source"`
	Campaign            string                 `json:"campaign"`
	Market              string                 `json:"market"`
	Period              QueryPeriod            `json:"period"`
	Metrics             ConversionFunnelMetric `json:"metrics"`
	Comparison          FunnelChange           `json:"comparison"`
	DataCompleteThrough time.Time              `json:"data_complete_through"`
}

type ConversionFunnelMetric struct {
	Sessions       int `json:"sessions"`
	ProductViews   int `json:"product_views"`
	CheckoutStarts int `json:"checkout_starts"`
	Purchases      int `json:"purchases"`
}

type FunnelChange struct {
	Period             string  `json:"period"`
	PurchasesChangePct float64 `json:"purchases_change_pct"`
}

type TrackingHealth struct {
	EvidenceID          string            `json:"evidence_id"`
	Source              string            `json:"source"`
	Campaign            string            `json:"campaign"`
	Market              string            `json:"market"`
	Status              string            `json:"status"`
	Findings            []TrackingFinding `json:"findings"`
	DataCompleteThrough time.Time         `json:"data_complete_through"`
}

type TrackingFinding struct {
	Type                       string    `json:"type"`
	Segment                    string    `json:"segment"`
	FirstObservedAt            time.Time `json:"first_observed_at"`
	EstimatedUnderReportingPct float64   `json:"estimated_under_reporting_pct"`
	Summary                    string    `json:"summary"`
}

type DataFreshness struct {
	EvidenceID string            `json:"evidence_id"`
	Sources    []FreshnessSource `json:"sources"`
}

type FreshnessSource struct {
	Name            string    `json:"name"`
	CompleteThrough time.Time `json:"complete_through"`
	Status          string    `json:"status"`
}
