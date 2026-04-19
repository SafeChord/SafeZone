# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

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