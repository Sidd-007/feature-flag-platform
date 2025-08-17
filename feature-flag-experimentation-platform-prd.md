# PRD: Feature Flag & Experimentation Platform (Go)

You are building a production-grade **Feature Flag & Experimentation Platform** composed of a **Control Plane**, an **Edge Evaluator**, an **Event Ingestor/Analytics pipeline**, and a **Web Admin UI**. Prioritize clean architecture, tests, observability, and clear docs. Generate all code, config, infra, Makefile, and CI in one monorepo.

## 0) Tech Constraints & Quality Bar
- **Language:** Go 1.22+  
- **Frameworks/Libraries (suggested):**
  - HTTP: `chi` (router), `net/http` primitives
  - Persistence: `pgx` for Postgres; `clickhouse-go/v2` for analytics
  - Caching: Redis (`go-redis/v9`)
  - Messaging: **NATS JetStream** (preferred) or Kafka (make it switchable behind an interface)
  - Telemetry: OpenTelemetry (traces), Prometheus (metrics), Zerolog (structured logs)
  - Config: Viper + env vars; 12-factor ready
  - AuthN/Z: OIDC/JWT, bcrypt for local users, **casbin** for RBAC policies
  - Migrations: `golang-migrate`
  - API: REST + gRPC (use `buf` for proto generation)
- **DBs:** Postgres (control plane), ClickHouse (analytics), Redis (edge cache), optional S3 for cold storage.
- **Build/Run:** Docker + docker-compose for local; Makefile targets (`make dev`, `make test`, `make lint`, `make seed`, `make up`)
- **CI:** GitHub Actions: lint, unit + integration tests, race detector, coverage badge, docker build.
- **Observability:** `/metrics`, pprof, trace exporter, log sampling in prod.
- **Security:** TLS everywhere, mTLS between services (local certs via mkcert), secret rotation hooks.

## 1) Problem Statement & Goals
Enable teams to:
1) Create **feature flags** (boolean, multivariate, JSON) with **targeting rules**, **segments**, and **% rollouts**.  
2) Run **experiments** (A/B/n) with **sticky bucketing**, exposure tracking, and outcome metrics.  
3) Serve flag decisions at **edge latency** with **realtime config updates**.  
4) Analyze experiment results with guardrails and clear **stop/ship** decisions.

**Non-Goals (v1):** WYSIWYG visual editor for rules, CDP integrations beyond webhooks, on-device mobile SDKs (scaffold stubs only).

## 2) Core Personas & Use Cases
- **Developer:** Toggle features per env, target cohorts, gradual rollouts, SDK integration.
- **Product/Data:** Create experiments, define metrics, monitor effects, stop on guardrails.
- **Admin:** Multi-tenant orgs, environments, RBAC, audit logs.

## 3) High-Level Architecture
- **Control Plane API (Go)**  
  CRUD for orgs/projects/environments/flags/segments/experiments; rule compilation; config distribution; RBAC; audit; token issuance.
- **Edge Evaluator (Go)**  
  In-memory compiled rule graph per env; hot-reload via streaming; ETag/If-None-Match; low-latency evaluation; local LRU.
- **Event Ingestor (Go)**  
  Collects `exposure`, `metric` events (HTTP/gRPC), validates, batches to NATS/Kafka → ClickHouse; DLQ on parse errors.
- **Analytics/Experiment Engine (Go)**  
  Aggregations over ClickHouse; computes metrics (binary/ratio/continuous); **Frequentist** (Welch t-test) and **Chi-square** for binary; optional **sequential monitoring** (Pocock alpha-spending); CUPED toggle.
- **Admin Web UI (Next.js/TS)**  
  Org/project setup, flag editor (JSON/Rule DSL), experiment designer, live rollouts, dashboards.
- **SDKs**  
  Go and Node/TS SDKs with local cache + streaming updates + offline/bootstrapped modes.

Provide an **architecture diagram (ASCII in README)** and per-service folder with OWNERS/README.

## 4) Data Model (conceptual; implement in Postgres + migrations)
Key entities (include created_at/updated_at/version/audit fields):
- **org(id, name, slug, billing_tier)**
- **project(id, org_id, name, key)**
- **environment(id, project_id, name, key, salt, is_prod)**
- **segment(id, env_id, name, rules_json)** – rule DSL stored canonicalized
- **flag(id, env_id, key, type[bool|multivariate|json], variations[], default_variation, rules_json, status[active|archived])**
- **experiment(id, env_id, key, flag_id, variations_map, hypothesis, primary_metric_id, secondary_metric_ids[], start_at, stop_at, traffic_allocation, status)**
- **api_token(id, env_id|org_id, scope, hashed_secret, expires_at)**
- **audit_log(id, actor, action, resource, diff_json)**

Analytics (ClickHouse, columnar):
- **events_exposure(date, env_key, flag_key, variation_key, user_key_hash, bucketing_id, meta JSON)**
- **events_metric(date, env_key, metric_key, user_key_hash, value Float64, meta JSON)**
- **experiments_snapshot(materialized view)** for fast queries.

## 5) Rule Evaluation & Bucketing
- **Rule DSL (JSON)**  
  - `if` conditions on user attributes / custom context (`eq`, `neq`, `in`, `regex`, `lt`, `gt`, `contains`, `semver`)  
  - `then` return variation or percentage allocation.
  - **Segments** can be referenced inside rules.
- **Priority:** flag rules (top→down) > segment rules > default variation.  
- **Sticky bucketing:** `bucketing_id = SHA256(env_salt + flag_key + user_key)`; use deterministic mapping to a 0–9999 bucket → assign variation ranges for % rollout. Document collision behavior.  
- **Mutual exclusivity:** optional exclusion groups; ensure a user is in only one experiment per group.

## 6) APIs (REST + gRPC) — define OpenAPI + Protos
- **Auth:** Bearer (API tokens), OIDC for UI sessions.
- **Admin (Control Plane) REST (prefix `/v1`)**  
  - POST `/orgs`, `/projects`, `/environments`  
  - CRUD `/flags`, `/segments`, `/experiments`  
  - POST `/flags/{key}/publish` (bumps config version)  
  - GET `/configs/{envKey}` → signed config doc (ETag, max-age)  
  - GET `/experiments/{key}/results?window=7d`
- **Edge Evaluate (low-latency)**  
  - POST `/evaluate` body: `{ envKey, flagKey, context }` → `{ variation, reason, configVersion }`  
  - GET `/stream/{envKey}` SSE: push config diffs  
  - Health `/ready`, `/live`
- **Events**  
  - POST `/events/exposure` (batch)  
  - POST `/events/metrics` (batch)  
  Validate schema, enqueue to NATS/Kafka with idempotency keys.
- Provide **gRPC equivalents** with streaming for config updates & event ingestion.

Return consistent error shapes, correlation IDs, and rate-limit headers.

## 7) SDKs (Go & Node/TS)
- **Init options:** `envKey`, `endpoint`, `bootstrapConfig`, `stream=true|false`, `pollInterval`, `timeout`, `cacheTTL`.
- **Public API:**  
  - `Variation(flagKey string, user Context, default any) (any, Reason, error)`  
  - `IsEnabled(flagKey, user) bool`  
  - Events: `TrackExposure(flagKey, variation, user)`, `TrackMetric(metricKey, value, user)`
- **Behavior:**  
  - Local in-memory config with background **SSE stream** (fallback to ETag polling).  
  - **Offline mode** supported (bootstrap only).  
  - Automatic exposure event on first evaluation per session (configurable).  
  - Retries with exponential backoff; circuit breaker.  
  - Pluggable store interface (memory/Redis).  
  - **Zero PII:** SDK hashes `user.key` client-side before sending.

Publish minimal examples (Go HTTP app, Node Express).

## 8) Experimentation & Metrics
- **Metric types:**  
  - Binary (conversion)  
  - Ratio/rate (events per user)  
  - Continuous (time, revenue)  
- **Stats:**  
  - Primary: **Welch’s t-test** for continuous; **Chi-square** for binary.  
  - Optional: **Sequential monitoring** with **Pocock** boundaries (alpha spending), to allow early stop with controlled Type-I error.  
  - **CUPED** (pre-period covariate) toggle to reduce variance (document formula).  
  - Report: lift, CI (95%), p-values, MDE calculator, power assumptions.
- **Guardrails:** e.g., error rate, latency p95. Auto-stop rules if breached.

Provide a results API & UI with clear interpretation and caveats.

## 9) Realtime Config Distribution
- **Publish flow:**  
  - On flag edit → validate DSL → compile to compact form → versioned config doc per env → sign (HMAC).  
  - Edge nodes receive diffs over SSE; fall back to polling with ETag.  
  - Cache layers: Redis (shared) + local LRU; cold start warmup on boot.
- **Consistency model:** eventual across regions, target **< 5s** convergence. Document guarantees.

## 10) Performance & SLOs
- **Edge Evaluate:** p99 < 10ms @ 2k RPS per pod, 512MB RAM.  
- **Control Plane:** p95 < 150ms for CRUD/list calls.  
- **Ingestion:** sustain 10k events/sec with ≤ 1 min freshness to ClickHouse.  
- **Availability targets:** 99.9% edge, 99.5% control plane.  
- Include **load test plan** (wrk/k6), flamegraphs, and optimization notes.

## 11) Security & Compliance
- RBAC: roles = **owner, admin, editor, viewer**, resource-scoped to org/project/env.  
- API tokens: env-scoped, hashed at rest, last-used timestamp.  
- PII: never store raw user identifiers in analytics; only salted hashes.  
- Audit log for all mutating actions.  
- Rate limiting per token & IP; WAF rules for events ingestion.  
- Secrets via env; rotate safely; no secrets in git.

## 12) Admin UI (Next.js + shadcn/ui)
- **Pages:**  
  - Org/Project/Env management  
  - Flags list + detail editor (JSON + rule builder form)  
  - Segments (JSON rules)  
  - Experiments designer (traffic split, metrics, targeting)  
  - **Live Rollout** view with % slider and preview by sample users  
  - Results dashboard with significance indicators & guardrails  
  - Audit logs  
  - Tokens & RBAC
- **DX:** Copyable SDK snippets per language, quickstart, and sample apps.

(Generate UI as a sibling package; separate CI job.)

## 13) Dev & Prod Deployment
- **Local:** docker-compose with Postgres, Redis, NATS JetStream, ClickHouse, MinIO (S3-compatible). Seed script creates a demo org, envs, flags, and a sample experiment.  
- **Prod (docs only):** k8s manifests/Helm charts; horizontal autoscaling hints; example AWS mapping (RDS, ElastiCache, MSK/NATS on EKS, S3, CloudFront).  
- **Config:** all creds via env; provide `.env.example`.

## 14) Testing Strategy
- Unit tests ≥ 80% for rule engine, bucketing, SDK logic.  
- Integration tests for API/DB/cache/message bus.  
- Contract tests for OpenAPI & gRPC with golden files.  
- Property tests (go-quickcheck) for bucketing determinism & partitioning.  
- Chaos tests: kill leader during publish; message loss; delayed ClickHouse writes.  
- Load tests with reproducible profiles.

## 15) Observability & Ops
- Metrics to expose:  
  - control_plane_publish_latency, config_version_gauge, edge_eval_latency_{p50,p95,p99}, cache_hit_ratio, ingestion_lag_seconds, events_dropped_total, experiment_active_count  
- Traces for: publish→edge propagation, evaluate, event ingest→warehouse commit.  
- Structured logs with correlation IDs; sampling in prod.  
- Runbook: common incidents (hot shard, publish stuck, ingestion lag).

## 16) Deliverables (must generate)
1. **Monorepo layout**
```
/cmd/control-plane
/cmd/edge-evaluator
/cmd/event-ingestor
/cmd/analytics-engine
/pkg/ (shared libraries: auth, rbac, dsl, bucketing, hashing, config, storage, messaging)
/sdk/go
/sdk/node
/web/admin
/deploy/docker-compose.yml
/deploy/k8s/ (manifests or Helm)
/db/migrations/postgres
/db/migrations/clickhouse
/proto/ (with buf.yaml)
/api/openapi.yaml
/Makefile
/README.md (top-level) + service READMEs
```
2. **OpenAPI spec** and **buf** proto set; generate clients.  
3. **Migrations** for Postgres/ClickHouse.  
4. **Seed script** to generate demo data.  
5. **CI** (GitHub Actions) with lint/test/build, coverage, and docker publish.  
6. **Load/chaos test harness** (k6 scripts, fault injection toggles).  
7. **Architecture diagram** (ASCII + mermaid).  
8. **Docs:** Quickstart, SDK usage, API reference, SLOs, limits, FAQs.

## 17) Acceptance Criteria (Definition of Done)
- Create a flag and publish; Edge receives new config within **≤5s**.  
- SDK can evaluate a % rollout with stable sticky bucketing across restarts.  
- Exposure + metric events flow end-to-end and appear in ClickHouse within **≤60s**.  
- Experiment results endpoint returns lift, CI, p-values for a demo dataset.  
- Rollback of a flag publish works; Edge reverts within **≤5s**.  
- All services emit Prometheus metrics and basic traces; health/readiness endpoints pass.  
- `make up` runs full local stack; `make seed` creates demo org/flags/experiment; `make test` green.  
- README lets a new engineer go from zero → demo in < 15 minutes.

## 18) Nice-to-Haves (scaffold but optional)
- **Exclusion groups** for mutually exclusive experiments.  
- **CUPED** computation with pre-period window.  
- **Computed segments** via SQL saved views.  
- **Webhooks** on publish/event thresholds.  
- **Multi-region read-replicas** doc with latency budget.

## 19) Implementation Notes (explicit directions to generate clean code)
- Keep domain logic pure in `/pkg`, keep handlers thin.  
- Encapsulate storage behind interfaces; provide Postgres/Redis/ClickHouse/NATS implementations.  
- Rule DSL: generate a **compiled plan** (e.g., bytecode-like struct) for edge.  
- Use context propagation, deadlines, and idempotency keys on publish/events.  
- Hashing: SHA-256; expose a `DeterministicBucket(id string) int` helper returning 0–9999.  
- Provide `docker/Dockerfile.*` per service with multi-stage builds.  
- Add sample dashboards (Grafana JSON) for metrics.
