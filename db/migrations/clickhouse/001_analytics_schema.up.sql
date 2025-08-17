-- Create analytics schema for Feature Flag Platform events

-- Exposure events table (flag evaluations)
CREATE TABLE IF NOT EXISTS events_exposure (
    date Date DEFAULT today(),
    timestamp DateTime64(3) DEFAULT now64(),
    env_key String,
    flag_key String,
    variation_key String,
    user_key_hash String,
    bucketing_id String,
    experiment_key String,
    session_id String,
    context_json String DEFAULT '{}',
    meta_json String DEFAULT '{}',
    created_at DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY date
ORDER BY (env_key, flag_key, timestamp)
TTL date + INTERVAL 2 YEAR;

-- Metric events table (custom metrics tracking)
CREATE TABLE IF NOT EXISTS events_metric (
    date Date DEFAULT today(),
    timestamp DateTime64(3) DEFAULT now64(),
    env_key String,
    metric_key String,
    user_key_hash String,
    value Float64,
    unit String DEFAULT '',
    context_json String DEFAULT '{}',
    meta_json String DEFAULT '{}',
    created_at DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY date
ORDER BY (env_key, metric_key, timestamp)
TTL date + INTERVAL 2 YEAR;

-- Materialized view for experiment snapshots (pre-aggregated data)
CREATE MATERIALIZED VIEW IF NOT EXISTS experiments_snapshot_mv TO experiments_snapshot AS
SELECT
    env_key,
    experiment_key,
    variation_key,
    date,
    count() as exposures_count,
    uniq(user_key_hash) as unique_users,
    min(timestamp) as first_exposure,
    max(timestamp) as last_exposure
FROM events_exposure
WHERE experiment_key != ''
GROUP BY env_key, experiment_key, variation_key, date;

-- Target table for the materialized view
CREATE TABLE IF NOT EXISTS experiments_snapshot (
    env_key String,
    experiment_key String,
    variation_key String,
    date Date,
    exposures_count UInt64,
    unique_users UInt64,
    first_exposure DateTime64(3),
    last_exposure DateTime64(3)
) ENGINE = SummingMergeTree()
PARTITION BY date
ORDER BY (env_key, experiment_key, variation_key, date);

-- Aggregated metrics view for experiment results
CREATE MATERIALIZED VIEW IF NOT EXISTS experiment_metrics_mv TO experiment_metrics AS
SELECT
    ee.env_key,
    ee.experiment_key,
    ee.variation_key,
    em.metric_key,
    ee.date,
    count() as metric_events_count,
    uniq(em.user_key_hash) as unique_users,
    avg(em.value) as avg_value,
    sum(em.value) as sum_value,
    min(em.value) as min_value,
    max(em.value) as max_value,
    quantile(0.5)(em.value) as median_value,
    quantile(0.95)(em.value) as p95_value,
    quantile(0.99)(em.value) as p99_value
FROM events_exposure ee
JOIN events_metric em ON 
    ee.user_key_hash = em.user_key_hash 
    AND ee.env_key = em.env_key
    AND ee.date = em.date
    AND ee.timestamp <= em.timestamp
    AND em.timestamp <= ee.timestamp + INTERVAL 1 DAY
WHERE ee.experiment_key != ''
GROUP BY ee.env_key, ee.experiment_key, ee.variation_key, em.metric_key, ee.date;

-- Target table for experiment metrics aggregation
CREATE TABLE IF NOT EXISTS experiment_metrics (
    env_key String,
    experiment_key String,
    variation_key String,
    metric_key String,
    date Date,
    metric_events_count UInt64,
    unique_users UInt64,
    avg_value Float64,
    sum_value Float64,
    min_value Float64,
    max_value Float64,
    median_value Float64,
    p95_value Float64,
    p99_value Float64
) ENGINE = SummingMergeTree()
PARTITION BY date
ORDER BY (env_key, experiment_key, variation_key, metric_key, date);

-- Performance monitoring events
CREATE TABLE IF NOT EXISTS events_performance (
    date Date DEFAULT today(),
    timestamp DateTime64(3) DEFAULT now64(),
    service_name String,
    operation_name String,
    duration_ms Float64,
    status_code UInt16,
    env_key String,
    trace_id String,
    span_id String,
    tags_json String DEFAULT '{}',
    created_at DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY date
ORDER BY (service_name, operation_name, timestamp)
TTL date + INTERVAL 6 MONTH;

-- Error events tracking
CREATE TABLE IF NOT EXISTS events_error (
    date Date DEFAULT today(),
    timestamp DateTime64(3) DEFAULT now64(),
    service_name String,
    error_type String,
    error_message String,
    stack_trace String,
    env_key String,
    user_key_hash String,
    trace_id String,
    context_json String DEFAULT '{}',
    created_at DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY date
ORDER BY (service_name, error_type, timestamp)
TTL date + INTERVAL 1 YEAR;
