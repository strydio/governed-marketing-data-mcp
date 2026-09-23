package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func OpenSQLiteDB(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) ApplyMigrations(ctx context.Context, migrationsPath string) error {
	if migrationsPath == "" {
		return fmt.Errorf("migrations path is required")
	}

	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	migrationNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		migrationNames = append(migrationNames, entry.Name())
	}
	sort.Strings(migrationNames)

	for _, name := range migrationNames {
		contents, err := os.ReadFile(filepath.Join(migrationsPath, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := s.db.ExecContext(ctx, string(contents)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func ResetDatabase(path string) error {
	if path == "" {
		return fmt.Errorf("sqlite path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove sqlite database: %w", err)
	}
	return nil
}

func (s *SQLiteStore) CampaignExists(ctx context.Context, slug, market string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM campaigns WHERE slug = ? AND market = ? AND active = 1`, slug, market).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check campaign existence: %w", err)
	}
	return count > 0, nil
}

func (s *SQLiteStore) AttributedConversions(ctx context.Context, slug, market, startDate, endDate string) (int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(cdm.attributed_conversions), 0)
		FROM campaign_daily_metrics cdm
		JOIN campaigns c ON c.campaign_id = cdm.campaign_id
		WHERE c.slug = ? AND c.market = ? AND cdm.date BETWEEN ? AND ?`, slug, market, startDate, endDate).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum attributed conversions: %w", err)
	}
	return total, nil
}

func (s *SQLiteStore) Purchases(ctx context.Context, slug, market, startDate, endDate string) (int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(fdm.purchases), 0)
		FROM funnel_daily_metrics fdm
		JOIN campaigns c ON c.campaign_id = fdm.campaign_id
		WHERE c.slug = ? AND c.market = ? AND fdm.date BETWEEN ? AND ?`, slug, market, startDate, endDate).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum purchases: %w", err)
	}
	return total, nil
}

func (s *SQLiteStore) TrackingIncidentCount(ctx context.Context, slug, market string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM tracking_incidents ti
		JOIN campaigns c ON c.campaign_id = ti.campaign_id
		WHERE c.slug = ? AND ti.market = ? AND ti.resolved_at IS NULL`, slug, market).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count tracking incidents: %w", err)
	}
	return count, nil
}

func (s *SQLiteStore) CurrentFreshnessCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM source_freshness WHERE status = 'current'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count current freshness rows: %w", err)
	}
	return count, nil
}
