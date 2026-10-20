# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.3.8] - 2026-10-20

### Added

- **Worker health endpoint (#71)**: The worker serves `GET /health`, which reports whether its consume loop is still turning within `LIVENESS_WINDOW` (default 30s). It does not query Kafka or PostgreSQL, so a dependency outage does not make every worker unhealthy at once.

### Changed

- **Worker batching (#64)**: The worker writes once per poll instead of buffering events across polls. `BATCH_SIZE` is now the most events one poll returns, and `FLUSH_INTERVAL` how long one poll waits.
- **Contract enforcement (#70)**: The event and table contracts are now checked by tests in every service that produces or consumes them.
- **Internal**: The shared utils are a regular directory instead of a submodule (#66), the worker's packages are regrouped (#81), and local Compose uses the platform's infra images and a three-partition Kafka topic (#27, #74). No application-facing changes.

### Fixed

- **Worker offset commits (#64)**: Offsets are committed only after the events are stored, and a worker that loses a partition can no longer commit for it. This fixes a consumer-group lag on staging that never drained and held the autoscaler at three replicas.
- **Worker rides out outages (#98)**: A failed poll, write or offset commit no longer stops the worker. It retries with backoff, and `DEPENDENCY_TIMEOUT` (default 10s, must stay below `LIVENESS_WINDOW`) bounds every write and commit. An outage now shows as consumer lag and ERROR logs rather than pod restarts.
- **Worker skips events the case table cannot hold (#85)**: Events with a non-calendar date or more cases than the table can store are logged and skipped, instead of failing the whole batch and stalling the partition.
- **Worker startup and logging (#83, #95)**: The worker exits with a clear error when it cannot load its area tables, every log line carries the worker ID, `LOG_LEVEL` is honoured, and the image no longer bakes in a `.env` file.
- **Ingestor validation and publish errors (#82)**: The ingest API rejects records outside the event contract (non-calendar dates, case counts above the table's limit) and answers 500 when a publish fails or is not acknowledged within `KAFKA_PUBLISH_TIMEOUT` (default 4s). The default `KAFKA_TOPIC` is now `covid.case.data`.
- **Simulator error reporting (#84)**: A trigger fails with 500 when the ingest API rejects a record or does not answer, and a trigger with no data answers 404.

## [0.3.7] - 2026-06-25

### Changed

- **Safer data reset (#50)**: The case-data reset command now requires an explicit scope (a single year, or an explicit full wipe) instead of clearing all case data by default, preventing accidental data loss.
- **Per-environment datasets (#50)**: Test, development, and showcase data are now isolated from one another, so running smoke tests can no longer overwrite seeded showcase data.
- **Release pipeline**: Release candidates are now built and validated from dedicated release branches before promotion to production.

### Fixed

- **Region drill-down lookups (#48)**: Fixed a character mismatch (臺/台) in region names that prevented some areas from resolving during map drill-down.

## [0.3.6] - 2026-06-13

### Changed

- **Internal delivery/CI improvements** to the release pipeline (image promotion and release-candidate tracking). No application-facing changes.

## [0.3.5] - 2026-05-25

This release introduces the modern React SPA Dashboard (v2) replacing the legacy Python Dash implementation, decouples Nginx configuration for runtime flexibility, and optimizes the repository's aggregate CI orchestration pipelines.

### Added

- **React SPA Dashboard v2 (#38, closes #33)**: Introduced a fast, modern single-page application built on Vite + React 19 + TypeScript.
- **Geospatial Mapping & Analytics**: Integrated MapLibre GL for fluid vector map zooms/drill-down, and Recharts for responsive infection trends and Top 10 scoreboard analytics.
- **Unit Test Coverage**: Added comprehensive Vitest suite covering key React services (`apiClient`, `caseService`, and `timeService`) achieving robust test coverage.

### Changed

- **Nginx Decoupling**: Refactored the dashboard deployment strategy to keep the Docker image stateless and environment-agnostic. Decoupled `nginx.conf` so it is mounted dynamically via local Compose volumes or K8s ConfigMaps.
- **Language-Aware CI Orchestration**: Optimized Makefile aggregate targets (`ci-all`, `ci-%`). Redefined the test sequence based on compiler dependencies (`test -> build` for TS/Go, `build -> test` for Python overlays) to speed up local verification loops.

### Fixed

- **TopCities Scoreboard 7-Day Lock**: Resolved an inconsistency where the Top Cities scoreboard was incorrectly affected by the global time filter. Extracted a specialized `useTopCities` hook that locks all queries to a 7-day rolling window as per system requirements.
- **Dashboard Test Target Dependency**: Fixed a missing dependency in the Python `test-dashboard` target to ensure production image compilation is run beforehand.

## [0.3.2] - 2026-05-02

### Added

- **Makefile Automation**: `make test-*` targets now automatically trigger `make build-*`, ensuring tests always run against the latest image.

### Changed

- **Template Propagation**: The Python Microservice Scaffold (v0.3.1) has been successfully propagated to `data-ingestor` and `pandemic-simulator`.
- **Directory Restructuring**:
    - `data-ingestor`: Extracted `services/ingest_service.py` (zero FastAPI imports) and added `api/dependencies.py` for Kafka DI.
    - `pandemic-simulator`: Merged `pipeline/` modules into the `services/` layer and structured test directories into `unit/` and `integration/`.
- **Cleanup**: Removed legacy service READMEs in favor of the central KDD Knowledge Base in `Docs/`.

### Fixed

- **Test Backfill**: Added comprehensive unit tests for `data-ingestor` using mocked Kafka producers, achieving 87% coverage.

## [0.3.1] - 2026-04-22

### Added

- **Python Microservice Scaffold**: Established the canonical `api/core/services/exceptions` layered architecture within `analytics-api` as the project's blueprint.
- **Pure ASGI Middleware**: Replaced `BaseHTTPMiddleware` with a pure ASGI implementation to fix `ContextVar` isolation issues, ensuring `X-Cache-Status` headers are correctly propagated.
- **Layered Dependency Injection**: Decoupled the `redis_cache` decorator from FastAPI `Request` objects, moving to explicit DI providers in `api/dependencies.py`.
- **Cache Stampede Protection**: Implemented Double-Check Locking within the cache service to prevent database overwhelming during concurrent cache misses.

## [0.3.0] - 2026-04-19

This milestone release, titled "Tooling & Portability", focuses on decoupling the testing infrastructure from the host environment, enhancing protocol-level observability, and hardening the core asynchronous processing engine. It establishes a portable "Lab Environment" for the next phases of architectural evolution.

### Added

- **Container-Native Assertions (#31, #32)**: Introduced a modular, CSV-driven assertion engine (`smoke_test.py`) running within a dedicated Ops container. This replaces host-side dependencies (like jq/bash) with Python-native logic, ensuring identical test behavior across Local, CI, and K8s environments.
- **Protocol-Level Observability (#29, #30)**: Implemented `X-Cache-Status` header exposure in the `Analytics API` and forwarded it through the `cli-relay`. This allows real-time, non-invasive monitoring of cache HIT/MISS behavior via the CLI.
- **Midnight Sync for Time Server**: Refactored the mock time calculation logic to align `mock_update_time` with physical midnight (00:00:00). This ensures the simulated date rolls over consistently with physical time, preventing drift in daily simulation cronjobs.

### Changed

- **Go Worker Hardening (#17)**: Performed a deep refactor of the Go Worker to implement a cleaner Dependency Injection (DI) pattern, address potential memory behavior issues, and significantly increase unit test coverage.
- **CI/CD Pipeline v2**: Upgraded the `smoke-test.yml` workflow to utilize the new container-native runner, improving the reliability and reproducibility of PR validation.
- **CLI Relay Logic**: Enhanced the relay service to dynamically forward custom headers from downstream microservices to the CLI client.

### Fixed

- **Time Server Drift**: Fixed a bug where the simulation date rollover was tied to the service start/set time rather than a natural day boundary.

## [0.2.1] - 2025-09-12

This is a critical stability and modernization patch for the v0.2.x series. It resolves a fundamental compatibility issue that prevented the successful deployment of the asynchronous architecture on a modern, KRaft-based Kafka infrastructure. This release ensures that the core features introduced in v0.2.0 are robust, reliable, and production-ready.

### Changed

- **Upgraded Kafka Integration to be KRaft-native**: The entire Kafka client integration has been modernized. The underlying Go client library was migrated from `segmentio/kafka-go` to the more robust and KRaft-compatible `twmb/franz-go`.
- **Hardened Consumer Offset Management**: The Go worker's consumer logic was completely refactored to disable auto-commit and implement manual offset management. This guarantees "at-least-once" message processing and prevents the silent data loss that occurred in v0.2.0 under graceful shutdown or rebalancing scenarios.
- **Optimized Producer Partitioning Strategy**: The producer's partition key logic was updated to use a "natural key" (e.g., city-region), leveraging Kafka's native Murmur2 partitioner for better data distribution and performance.

### Fixed

- **Resolved KRaft Compatibility Bug**: Fixed a critical "silent failure" bug where the consumer would fail to receive messages from a KRaft-based Kafka cluster due to an upstream library issue, which was blocking the deployment of the v0.2.0 architecture.
- **Stabilized Test Environment**: The `smoke-test.sh` framework was hardened by replacing fixed `sleep` commands with a robust polling mechanism (`wait_for_infra_services`). This eliminates race conditions during startup and ensures the reliability of the entire CI pipeline.

## [0.2.0] - 2025-09-01

This version marks a major milestone, evolving the SafeChord project from an MVP into a feature-complete platform with industrial-grade automation and observability capabilities. Updates span infrastructure, network architecture, asynchronous application-layer processing, and the developer toolchain.

### Added

- **Observability Foundation**: Established a foundation for observability by introducing a `Trace ID` mechanism for end-to-end data flow tracing. Log outputs were standardized to JSON, **enabling seamless integration capabilities with external tools like Loki and Prometheus**.
- **Async Dataflow Architecture**: Introduced an event-driven architecture centered around Kafka. The `Data Ingestor` service was refactored to act as an event producer, while the `Pandemic Simulator` was upgraded to make asynchronous requests (`asyncio` + `httpx`), **providing the system with higher throughput and greater resilience**.
- **Go Worker**: Introduced a new `worker` service implemented in Go, acting as a Kafka consumer responsible for asynchronously batch-writing events to PostgreSQL.
- **API Caching Mechanism**: Implemented a Redis caching layer for the `Analytics API`, **providing the foundation for future performance optimizations**.
- **Time Server**: Added a `time-server` utility service for centralized time management and simulation, ensuring a consistent time baseline for all services in test and simulation scenarios.
- **Automated Testing**: Established a comprehensive end-to-end (E2E) smoke test framework (`smoke-test.sh`) integrated into the GitHub Actions CI pipeline as a core quality gate for pull requests.

### Changed

- **Service Renaming**: To improve general applicability, core services were renamed (e.g., `coviddatasimulator` -> `pandemic-simulator`) to decouple them from a specific event (COVID).
- **CI/CD Pipeline**: The CI/CD pipeline was completely refactored to use dynamic, short Git SHAs as image tags for PR builds. This resolves concurrency conflicts on the `self-hosted` runner and introduces a `release.yml` workflow **to support automated releases**.
- **Unified Data Contracts**: Centralized Pydantic models from various services into a shared `utils` submodule, creating unified data contracts and ensuring consistency across microservices.
- **Standardized Build Process**: Standardized the build interface for all services by abstracting Docker commands into a `Makefile` and adopting a consistent two-file (`Dockerfile` and `Dockerfile.test`) build pattern.
- **Toolkit (CLI)**: The `szcli` tool underwent a major architectural refactor, introducing structured JSON output (`--output json`) and a centralized logging framework to significantly improve its usability for automation and operations.

### Fixed

- **Worker (Go)**: Fixed a SQL parameter indexing bug in the Go worker that occurred when skipping events during batch database inserts.
- **API**: Resolved a service crash caused by an inconsistent Pydantic `HealthResponse` model format across services.

## [0.1.0] - 2025-05-16

- Initial MVP (Minimum Viable Product) release of the project. Included core services like `coviddatasimulator`, `coviddataingestor`, and `safezoneanalyticsapi` to validate the basic synchronous data flow.