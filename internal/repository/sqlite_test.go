package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSeedScenario(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	defer db.Close()

	store := OpenSQLiteDB(db)
	if err := store.ApplyMigrations(ctx, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	exists, err := store.CampaignExists(ctx, "autumn-sale", "UK")
	if err != nil {
		t.Fatalf("check UK campaign: %v", err)
	}
	if !exists {
		t.Fatal("expected UK autumn-sale campaign to exist")
	}

	usExists, err := store.CampaignExists(ctx, "summer-sale", "US")
	if err != nil {
		t.Fatalf("check US campaign: %v", err)
	}
	if !usExists {
		t.Fatal("expected US summer-sale campaign to exist")
	}

	baselineConversions, err := store.AttributedConversions(ctx, "autumn-sale", "UK", "2026-09-07", "2026-09-13")
	if err != nil {
		t.Fatalf("baseline conversions: %v", err)
	}
	incidentConversions, err := store.AttributedConversions(ctx, "autumn-sale", "UK", "2026-09-14", "2026-09-20")
	if err != nil {
		t.Fatalf("incident conversions: %v", err)
	}

	changePct := float64(incidentConversions-baselineConversions) / float64(baselineConversions) * 100
	if changePct > -28 || changePct < -32 {
		t.Fatalf("expected conversions to drop by about 30%%, got %.2f%%", changePct)
	}

	baselinePurchases, err := store.Purchases(ctx, "autumn-sale", "UK", "2026-09-07", "2026-09-13")
	if err != nil {
		t.Fatalf("baseline purchases: %v", err)
	}
	incidentPurchases, err := store.Purchases(ctx, "autumn-sale", "UK", "2026-09-14", "2026-09-20")
	if err != nil {
		t.Fatalf("incident purchases: %v", err)
	}

	purchaseChangePct := float64(incidentPurchases-baselinePurchases) / float64(baselinePurchases) * 100
	if purchaseChangePct < -2 || purchaseChangePct > 2 {
		t.Fatalf("expected purchases to remain stable, got %.2f%%", purchaseChangePct)
	}

	incidents, err := store.TrackingIncidentCount(ctx, "autumn-sale", "UK")
	if err != nil {
		t.Fatalf("tracking incident count: %v", err)
	}
	if incidents != 1 {
		t.Fatalf("expected one active tracking incident, got %d", incidents)
	}

	freshnessRows, err := store.CurrentFreshnessCount(ctx)
	if err != nil {
		t.Fatalf("freshness count: %v", err)
	}
	if freshnessRows != 3 {
		t.Fatalf("expected three current freshness rows, got %d", freshnessRows)
	}
}
