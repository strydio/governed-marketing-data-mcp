package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/strydio/governed-marketing-data-mcp/internal/domain"
	"github.com/strydio/governed-marketing-data-mcp/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func (s *SQLiteStore) GetCampaignPerformance(ctx context.Context, q domain.CommonQuery) (domain.CampaignPerformance, error) {
	ctx, span := telemetry.Tracer().Start(ctx, "db.query campaign_performance")
	defer span.End()
	span.SetAttributes(attribute.String(telemetry.AttrDataSource, "ad_reporting"))
	start, end, prevStart, prevEnd, err := periods(q.StartDate, q.EndDate)
	if err != nil {
		return domain.CampaignPerformance{}, err
	}

	var spendMinor, impressions, clicks, conversions int
	var attributionModel string
	var attributionWindowDays int
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(cdm.spend_minor_units), 0), COALESCE(SUM(cdm.impressions), 0),
		       COALESCE(SUM(cdm.clicks), 0), COALESCE(SUM(cdm.attributed_conversions), 0),
		       COALESCE(MAX(cdm.attribution_model), ''), COALESCE(MAX(cdm.attribution_window_days), 0)
		FROM campaign_daily_metrics cdm
		JOIN campaigns c ON c.campaign_id = cdm.campaign_id
		WHERE c.slug = ? AND c.market = ? AND cdm.date BETWEEN ? AND ?`, q.Campaign, q.Market, q.StartDate, q.EndDate).
		Scan(&spendMinor, &impressions, &clicks, &conversions, &attributionModel, &attributionWindowDays)
	if err != nil {
		span.SetStatus(codes.Error, "query campaign performance")
		return domain.CampaignPerformance{}, fmt.Errorf("query campaign performance: %w", err)
	}

	previousConversions, err := s.AttributedConversions(ctx, q.Campaign, q.Market, prevStart, prevEnd)
	if err != nil {
		return domain.CampaignPerformance{}, err
	}
	freshness, err := s.sourceFreshness(ctx, "ad_reporting")
	if err != nil {
		return domain.CampaignPerformance{}, err
	}

	span.SetAttributes(attribute.Int(telemetry.AttrResultRecordCount, 1), attribute.String(telemetry.AttrDataCompleteThrough, freshness.CompleteThrough.Format(time.RFC3339)))
	return domain.CampaignPerformance{
		EvidenceID: fmt.Sprintf("ev-campaign-%s-%s-%s-%s", lowerMarket(q.Market), q.Campaign, compactDate(start), compactDate(end)),
		Source:     "ad_reporting",
		Campaign:   q.Campaign,
		Market:     q.Market,
		Period:     domain.QueryPeriod{Start: q.StartDate, End: q.EndDate},
		Metrics: domain.CampaignPerformanceMetrics{
			SpendGBP:              formatMinorUnits(spendMinor),
			Impressions:           impressions,
			Clicks:                clicks,
			AttributedConversions: conversions,
		},
		Comparison: domain.CampaignPerformanceChange{
			Period:                         "previous_7_days",
			AttributedConversionsChangePct: percentChange(previousConversions, conversions),
		},
		AttributionModel:      attributionModel,
		AttributionWindowDays: attributionWindowDays,
		DataCompleteThrough:   freshness.CompleteThrough,
	}, nil
}

func (s *SQLiteStore) GetConversionFunnel(ctx context.Context, q domain.CommonQuery) (domain.ConversionFunnel, error) {
	ctx, span := telemetry.Tracer().Start(ctx, "db.query conversion_funnel")
	defer span.End()
	span.SetAttributes(attribute.String(telemetry.AttrDataSource, "analytics_warehouse"))
	start, end, prevStart, prevEnd, err := periods(q.StartDate, q.EndDate)
	if err != nil {
		return domain.ConversionFunnel{}, err
	}

	var sessions, productViews, checkoutStarts, purchases int
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(fdm.sessions), 0), COALESCE(SUM(fdm.product_views), 0),
		       COALESCE(SUM(fdm.checkout_starts), 0), COALESCE(SUM(fdm.purchases), 0)
		FROM funnel_daily_metrics fdm
		JOIN campaigns c ON c.campaign_id = fdm.campaign_id
		WHERE c.slug = ? AND c.market = ? AND fdm.date BETWEEN ? AND ?`, q.Campaign, q.Market, q.StartDate, q.EndDate).
		Scan(&sessions, &productViews, &checkoutStarts, &purchases)
	if err != nil {
		span.SetStatus(codes.Error, "query conversion funnel")
		return domain.ConversionFunnel{}, fmt.Errorf("query conversion funnel: %w", err)
	}

	previousPurchases, err := s.Purchases(ctx, q.Campaign, q.Market, prevStart, prevEnd)
	if err != nil {
		return domain.ConversionFunnel{}, err
	}
	freshness, err := s.sourceFreshness(ctx, "analytics_warehouse")
	if err != nil {
		return domain.ConversionFunnel{}, err
	}

	span.SetAttributes(attribute.Int(telemetry.AttrResultRecordCount, 1), attribute.String(telemetry.AttrDataCompleteThrough, freshness.CompleteThrough.Format(time.RFC3339)))
	return domain.ConversionFunnel{
		EvidenceID: fmt.Sprintf("ev-funnel-%s-%s-%s-%s", lowerMarket(q.Market), q.Campaign, compactDate(start), compactDate(end)),
		Source:     "analytics_warehouse",
		Campaign:   q.Campaign,
		Market:     q.Market,
		Period:     domain.QueryPeriod{Start: q.StartDate, End: q.EndDate},
		Metrics: domain.ConversionFunnelMetric{
			Sessions:       sessions,
			ProductViews:   productViews,
			CheckoutStarts: checkoutStarts,
			Purchases:      purchases,
		},
		Comparison:          domain.FunnelChange{Period: "previous_7_days", PurchasesChangePct: percentChange(previousPurchases, purchases)},
		DataCompleteThrough: freshness.CompleteThrough,
	}, nil
}

func (s *SQLiteStore) GetTrackingHealth(ctx context.Context, q domain.CommonQuery) (domain.TrackingHealth, error) {
	ctx, span := telemetry.Tracer().Start(ctx, "db.query tracking_health")
	defer span.End()
	span.SetAttributes(attribute.String(telemetry.AttrDataSource, "data_quality_monitor"))
	rows, err := s.db.QueryContext(ctx, `
		SELECT ti.incident_id, ti.incident_type, ti.affected_segment, ti.first_observed_at,
		       ti.estimated_under_reporting_pct, ti.public_summary
		FROM tracking_incidents ti
		JOIN campaigns c ON c.campaign_id = ti.campaign_id
		WHERE c.slug = ? AND ti.market = ? AND ti.resolved_at IS NULL
		ORDER BY ti.first_observed_at`, q.Campaign, q.Market)
	if err != nil {
		span.SetStatus(codes.Error, "query tracking health")
		return domain.TrackingHealth{}, fmt.Errorf("query tracking health: %w", err)
	}
	defer rows.Close()

	findings := []domain.TrackingFinding{}
	firstIncidentID := "none"
	for rows.Next() {
		var incidentID, incidentType, segment, observedText, summary string
		var underReporting float64
		if err := rows.Scan(&incidentID, &incidentType, &segment, &observedText, &underReporting, &summary); err != nil {
			return domain.TrackingHealth{}, fmt.Errorf("scan tracking health: %w", err)
		}
		if firstIncidentID == "none" {
			firstIncidentID = incidentID
		}
		observedAt, err := time.Parse(time.RFC3339, observedText)
		if err != nil {
			return domain.TrackingHealth{}, fmt.Errorf("parse incident timestamp: %w", err)
		}
		findings = append(findings, domain.TrackingFinding{Type: incidentType, Segment: segment, FirstObservedAt: observedAt, EstimatedUnderReportingPct: underReporting, Summary: summary})
	}
	if err := rows.Err(); err != nil {
		return domain.TrackingHealth{}, fmt.Errorf("iterate tracking health: %w", err)
	}

	freshness, err := s.sourceFreshness(ctx, "data_quality_monitor")
	if err != nil {
		return domain.TrackingHealth{}, err
	}
	status := "healthy"
	if len(findings) > 0 {
		status = "degraded"
	}

	span.SetAttributes(attribute.Int(telemetry.AttrResultRecordCount, len(findings)), attribute.String(telemetry.AttrDataCompleteThrough, freshness.CompleteThrough.Format(time.RFC3339)))
	return domain.TrackingHealth{
		EvidenceID:          fmt.Sprintf("ev-%s", firstIncidentID),
		Source:              "data_quality_monitor",
		Campaign:            q.Campaign,
		Market:              q.Market,
		Status:              status,
		Findings:            findings,
		DataCompleteThrough: freshness.CompleteThrough,
	}, nil
}

func (s *SQLiteStore) GetDataFreshness(ctx context.Context) (domain.DataFreshness, error) {
	ctx, span := telemetry.Tracer().Start(ctx, "db.query data_freshness")
	defer span.End()
	span.SetAttributes(attribute.String(telemetry.AttrDataSource, "source_freshness"))
	rows, err := s.db.QueryContext(ctx, `SELECT source_name, complete_through, status FROM source_freshness ORDER BY source_name`)
	if err != nil {
		return domain.DataFreshness{}, fmt.Errorf("query data freshness: %w", err)
	}
	defer rows.Close()

	freshness := domain.DataFreshness{Sources: []domain.FreshnessSource{}}
	var latest time.Time
	for rows.Next() {
		var sourceName, completeThroughText, status string
		if err := rows.Scan(&sourceName, &completeThroughText, &status); err != nil {
			return domain.DataFreshness{}, fmt.Errorf("scan data freshness: %w", err)
		}
		completeThrough, err := time.Parse(time.RFC3339, completeThroughText)
		if err != nil {
			return domain.DataFreshness{}, fmt.Errorf("parse freshness timestamp: %w", err)
		}
		if completeThrough.After(latest) {
			latest = completeThrough
		}
		freshness.Sources = append(freshness.Sources, domain.FreshnessSource{Name: sourceName, CompleteThrough: completeThrough, Status: status})
	}
	if err := rows.Err(); err != nil {
		return domain.DataFreshness{}, fmt.Errorf("iterate data freshness: %w", err)
	}
	span.SetAttributes(attribute.Int(telemetry.AttrResultRecordCount, len(freshness.Sources)), attribute.String(telemetry.AttrDataCompleteThrough, latest.UTC().Format(time.RFC3339)))
	freshness.EvidenceID = fmt.Sprintf("ev-freshness-%s", latest.UTC().Format("20060102T1504Z"))
	return freshness, nil
}

type sourceFreshnessRow struct {
	CompleteThrough time.Time
	Status          string
}

func (s *SQLiteStore) sourceFreshness(ctx context.Context, sourceName string) (sourceFreshnessRow, error) {
	var completeThroughText, status string
	err := s.db.QueryRowContext(ctx, `SELECT complete_through, status FROM source_freshness WHERE source_name = ?`, sourceName).Scan(&completeThroughText, &status)
	if err == sql.ErrNoRows {
		return sourceFreshnessRow{}, fmt.Errorf("source freshness not found")
	}
	if err != nil {
		return sourceFreshnessRow{}, fmt.Errorf("query source freshness: %w", err)
	}
	completeThrough, err := time.Parse(time.RFC3339, completeThroughText)
	if err != nil {
		return sourceFreshnessRow{}, fmt.Errorf("parse freshness timestamp: %w", err)
	}
	return sourceFreshnessRow{CompleteThrough: completeThrough, Status: status}, nil
}

func periods(startDate, endDate string) (time.Time, time.Time, string, string, error) {
	start, err := time.Parse(time.DateOnly, startDate)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", fmt.Errorf("parse start date: %w", err)
	}
	end, err := time.Parse(time.DateOnly, endDate)
	if err != nil {
		return time.Time{}, time.Time{}, "", "", fmt.Errorf("parse end date: %w", err)
	}
	days := int(end.Sub(start).Hours()/24) + 1
	previousEnd := start.AddDate(0, 0, -1)
	previousStart := previousEnd.AddDate(0, 0, -(days - 1))
	return start, end, previousStart.Format(time.DateOnly), previousEnd.Format(time.DateOnly), nil
}

func percentChange(previous, current int) float64 {
	if previous == 0 {
		return 0
	}
	change := (float64(current-previous) / float64(previous)) * 100
	return math.Round(change*10) / 10
}

func compactDate(t time.Time) string {
	return t.Format("20060102")
}

func lowerMarket(market string) string {
	if market == "UK" {
		return "uk"
	}
	if market == "US" {
		return "us"
	}
	return market
}

func formatMinorUnits(minor int) string {
	return fmt.Sprintf("%.2f", float64(minor)/100)
}
