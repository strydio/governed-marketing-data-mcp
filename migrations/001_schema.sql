CREATE TABLE IF NOT EXISTS campaigns (
  campaign_id INTEGER PRIMARY KEY,
  slug TEXT NOT NULL,
  display_name TEXT NOT NULL,
  market TEXT NOT NULL,
  owner_team TEXT NOT NULL,
  active INTEGER NOT NULL,
  UNIQUE(slug, market)
);

CREATE TABLE IF NOT EXISTS campaign_daily_metrics (
  date TEXT NOT NULL,
  campaign_id INTEGER NOT NULL,
  spend_minor_units INTEGER NOT NULL,
  impressions INTEGER NOT NULL,
  clicks INTEGER NOT NULL,
  attributed_conversions INTEGER NOT NULL,
  attribution_model TEXT NOT NULL,
  attribution_window_days INTEGER NOT NULL,
  loaded_at TEXT NOT NULL,
  PRIMARY KEY(date, campaign_id),
  FOREIGN KEY(campaign_id) REFERENCES campaigns(campaign_id)
);

CREATE TABLE IF NOT EXISTS funnel_daily_metrics (
  date TEXT NOT NULL,
  campaign_id INTEGER NOT NULL,
  sessions INTEGER NOT NULL,
  product_views INTEGER NOT NULL,
  checkout_starts INTEGER NOT NULL,
  purchases INTEGER NOT NULL,
  loaded_at TEXT NOT NULL,
  PRIMARY KEY(date, campaign_id),
  FOREIGN KEY(campaign_id) REFERENCES campaigns(campaign_id)
);

CREATE TABLE IF NOT EXISTS tracking_incidents (
  incident_id TEXT PRIMARY KEY,
  campaign_id INTEGER NOT NULL,
  market TEXT NOT NULL,
  incident_type TEXT NOT NULL,
  affected_segment TEXT NOT NULL,
  first_observed_at TEXT NOT NULL,
  resolved_at TEXT,
  estimated_under_reporting_pct REAL NOT NULL,
  public_summary TEXT NOT NULL,
  FOREIGN KEY(campaign_id) REFERENCES campaigns(campaign_id)
);

CREATE TABLE IF NOT EXISTS source_freshness (
  source_name TEXT PRIMARY KEY,
  complete_through TEXT NOT NULL,
  expected_lag_minutes INTEGER NOT NULL,
  status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS scenario_ground_truth (
  scenario_id TEXT PRIMARY KEY,
  expected_cause TEXT NOT NULL,
  required_evidence_json TEXT NOT NULL,
  acceptable_causes_json TEXT NOT NULL
);
