package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/strydio/governed-marketing-data-mcp/internal/domain"
	_ "modernc.org/sqlite"
)

func seededStore(t *testing.T) *SQLiteStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	store := OpenSQLiteDB(db)
	if err := store.ApplyMigrations(context.Background(), "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return store
}

func TestGetCampaignPerformance(t *testing.T) {
	store := seededStore(t)
	got, err := store.GetCampaignPerformance(context.Background(), domain.CommonQuery{Campaign: "autumn-sale", Market: "UK", StartDate: "2026-09-14", EndDate: "2026-09-20"})
	if err != nil {
		t.Fatalf("get campaign performance: %v", err)
	}
	if got.EvidenceID != "ev-campaign-uk-autumn-sale-20260914-20260920" {
		t.Fatalf("unexpected evidence id: %s", got.EvidenceID)
	}
	if got.Metrics.AttributedConversions != 612 {
		t.Fatalf("unexpected conversions: %d", got.Metrics.AttributedConversions)
	}
	if got.Comparison.AttributedConversionsChangePct != -30.0 {
		t.Fatalf("unexpected conversion change: %.1f", got.Comparison.AttributedConversionsChangePct)
	}
	if got.Metrics.SpendGBP != "4277.50" {
		t.Fatalf("unexpected spend: %s", got.Metrics.SpendGBP)
	}
}

func TestGetConversionFunnel(t *testing.T) {
	store := seededStore(t)
	got, err := store.GetConversionFunnel(context.Background(), domain.CommonQuery{Campaign: "autumn-sale", Market: "UK", StartDate: "2026-09-14", EndDate: "2026-09-20"})
	if err != nil {
		t.Fatalf("get conversion funnel: %v", err)
	}
	if got.EvidenceID != "ev-funnel-uk-autumn-sale-20260914-20260920" {
		t.Fatalf("unexpected evidence id: %s", got.EvidenceID)
	}
	if got.Metrics.Purchases != 870 {
		t.Fatalf("unexpected purchases: %d", got.Metrics.Purchases)
	}
	if got.Comparison.PurchasesChangePct != 0.6 {
		t.Fatalf("unexpected purchase change: %.1f", got.Comparison.PurchasesChangePct)
	}
}

func TestGetTrackingHealth(t *testing.T) {
	store := seededStore(t)
	got, err := store.GetTrackingHealth(context.Background(), domain.CommonQuery{Campaign: "autumn-sale", Market: "UK"})
	if err != nil {
		t.Fatalf("get tracking health: %v", err)
	}
	if got.EvidenceID != "ev-tracking-incident-017" {
		t.Fatalf("unexpected evidence id: %s", got.EvidenceID)
	}
	if got.Status != "degraded" {
		t.Fatalf("unexpected status: %s", got.Status)
	}
	if len(got.Findings) != 1 || got.Findings[0].Segment != "browser_family:safari" {
		t.Fatalf("unexpected findings: %#v", got.Findings)
	}
}

func TestGetDataFreshness(t *testing.T) {
	store := seededStore(t)
	got, err := store.GetDataFreshness(context.Background())
	if err != nil {
		t.Fatalf("get data freshness: %v", err)
	}
	if got.EvidenceID != "ev-freshness-20260921T0415Z" {
		t.Fatalf("unexpected evidence id: %s", got.EvidenceID)
	}
	if len(got.Sources) != 3 {
		t.Fatalf("expected three sources, got %d", len(got.Sources))
	}
}
