-- Drop analytics schema for Feature Flag Platform

-- Drop materialized views first
DROP VIEW IF EXISTS experiment_metrics_mv;
DROP VIEW IF EXISTS experiments_snapshot_mv;

-- Drop tables
DROP TABLE IF EXISTS events_error;
DROP TABLE IF EXISTS events_performance;
DROP TABLE IF EXISTS experiment_metrics;
DROP TABLE IF EXISTS experiments_snapshot;
DROP TABLE IF EXISTS events_metric;
DROP TABLE IF EXISTS events_exposure;
