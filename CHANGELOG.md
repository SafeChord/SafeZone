# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

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