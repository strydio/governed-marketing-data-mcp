DELETE FROM scenario_ground_truth;
DELETE FROM source_freshness;
DELETE FROM tracking_incidents;
DELETE FROM funnel_daily_metrics;
DELETE FROM campaign_daily_metrics;
DELETE FROM campaigns;

INSERT INTO campaigns (campaign_id, slug, display_name, market, owner_team, active) VALUES
  (1, 'autumn-sale', 'Autumn Sale', 'UK', 'uk_marketing', 1),
  (2, 'summer-sale', 'Summer Sale', 'US', 'us_marketing', 1);

INSERT INTO campaign_daily_metrics (date, campaign_id, spend_minor_units, impressions, clicks, attributed_conversions, attribution_model, attribution_window_days, loaded_at) VALUES
  ('2026-09-07', 1, 61000, 34500, 1840, 124, 'last_non_direct_click', 7, '2026-09-14T04:00:00Z'),
  ('2026-09-08', 1, 61200, 34600, 1842, 125, 'last_non_direct_click', 7, '2026-09-14T04:00:00Z'),
  ('2026-09-09', 1, 60950, 34350, 1835, 123, 'last_non_direct_click', 7, '2026-09-14T04:00:00Z'),
  ('2026-09-10', 1, 61100, 34400, 1845, 126, 'last_non_direct_click', 7, '2026-09-14T04:00:00Z'),
  ('2026-09-11', 1, 60800, 34250, 1830, 124, 'last_non_direct_click', 7, '2026-09-14T04:00:00Z'),
  ('2026-09-12', 1, 61300, 34650, 1850, 125, 'last_non_direct_click', 7, '2026-09-14T04:00:00Z'),
  ('2026-09-13', 1, 61050, 34480, 1848, 127, 'last_non_direct_click', 7, '2026-09-14T04:00:00Z'),
  ('2026-09-14', 1, 61150, 34400, 1840, 88, 'last_non_direct_click', 7, '2026-09-21T04:00:00Z'),
  ('2026-09-15', 1, 61000, 34350, 1838, 87, 'last_non_direct_click', 7, '2026-09-21T04:00:00Z'),
  ('2026-09-16', 1, 61200, 34500, 1845, 86, 'last_non_direct_click', 7, '2026-09-21T04:00:00Z'),
  ('2026-09-17', 1, 60900, 34300, 1832, 89, 'last_non_direct_click', 7, '2026-09-21T04:00:00Z'),
  ('2026-09-18', 1, 61350, 34600, 1850, 87, 'last_non_direct_click', 7, '2026-09-21T04:00:00Z'),
  ('2026-09-19', 1, 61100, 34420, 1842, 88, 'last_non_direct_click', 7, '2026-09-21T04:00:00Z'),
  ('2026-09-20', 1, 61050, 34430, 1843, 87, 'last_non_direct_click', 7, '2026-09-21T04:00:00Z'),
  ('2026-09-14', 2, 70000, 39000, 2100, 140, 'last_non_direct_click', 7, '2026-09-21T04:00:00Z');

INSERT INTO funnel_daily_metrics (date, campaign_id, sessions, product_views, checkout_starts, purchases, loaded_at) VALUES
  ('2026-09-07', 1, 1800, 1328, 168, 123, '2026-09-14T04:10:00Z'),
  ('2026-09-08', 1, 1810, 1331, 169, 124, '2026-09-14T04:10:00Z'),
  ('2026-09-09', 1, 1790, 1317, 167, 122, '2026-09-14T04:10:00Z'),
  ('2026-09-10', 1, 1820, 1340, 170, 125, '2026-09-14T04:10:00Z'),
  ('2026-09-11', 1, 1805, 1329, 168, 123, '2026-09-14T04:10:00Z'),
  ('2026-09-12', 1, 1815, 1336, 169, 124, '2026-09-14T04:10:00Z'),
  ('2026-09-13', 1, 1812, 1334, 169, 124, '2026-09-14T04:10:00Z'),
  ('2026-09-14', 1, 1802, 1327, 168, 124, '2026-09-21T04:10:00Z'),
  ('2026-09-15', 1, 1814, 1335, 169, 125, '2026-09-21T04:10:00Z'),
  ('2026-09-16', 1, 1798, 1324, 168, 123, '2026-09-21T04:10:00Z'),
  ('2026-09-17', 1, 1809, 1332, 169, 124, '2026-09-21T04:10:00Z'),
  ('2026-09-18', 1, 1818, 1338, 170, 125, '2026-09-21T04:10:00Z'),
  ('2026-09-19', 1, 1801, 1326, 168, 124, '2026-09-21T04:10:00Z'),
  ('2026-09-20', 1, 1808, 1328, 168, 125, '2026-09-21T04:10:00Z'),
  ('2026-09-14', 2, 2000, 1500, 180, 138, '2026-09-21T04:10:00Z');

INSERT INTO tracking_incidents (incident_id, campaign_id, market, incident_type, affected_segment, first_observed_at, resolved_at, estimated_under_reporting_pct, public_summary) VALUES
  ('tracking-incident-017', 1, 'UK', 'missing_conversion_events', 'browser_family:safari', '2026-09-16T09:20:00Z', NULL, 28.4, 'Purchase conversion events are under-reported for Safari sessions.');

INSERT INTO source_freshness (source_name, complete_through, expected_lag_minutes, status) VALUES
  ('ad_reporting', '2026-09-21T04:00:00Z', 180, 'current'),
  ('analytics_warehouse', '2026-09-21T04:10:00Z', 180, 'current'),
  ('data_quality_monitor', '2026-09-21T04:15:00Z', 180, 'current');

INSERT INTO scenario_ground_truth (scenario_id, expected_cause, required_evidence_json, acceptable_causes_json) VALUES
  ('autumn-sale-tracking-gap', 'conversion_tracking_under_reporting', '["campaign_performance","conversion_funnel","tracking_health"]', '["safari_tracking_failure","missing_conversion_events"]');
