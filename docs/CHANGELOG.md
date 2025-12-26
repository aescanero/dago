# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- NATS event bus support
- PostgreSQL for long-term storage
- Advanced monitoring dashboard
- Graph versioning
- Execution replay

Note: Multi-provider LLM support and auto-scaling worker pools are features of worker services.

## [1.0.0] - 2025-12-02

### Added
- Initial MVP release
- Core orchestration engine
  - Orchestrator manager for graph coordination (event-driven)
  - Graph validator for structure and cycle detection
  - Event publishing to Redis Streams (executor.work, router.work)
  - Event listening for completion (node.completed)
- Infrastructure adapters
  - Redis Streams event bus
  - Redis state storage
  - Prometheus metrics collector
- API interfaces
  - HTTP REST API for graph management
  - WebSocket API for real-time updates
  - gRPC API for service-to-service communication
- Deployment support
  - Docker image with multi-stage build
  - Docker Compose for local development
  - Helm chart for Kubernetes deployment
  - GitHub Actions CI/CD pipeline
- Documentation
  - Architecture and usage guide
  - API documentation
  - Deployment guide
  - Comprehensive README

### Dependencies
- Go 1.21
- dago-libs v1.0.0
- Redis 7.0+ for event bus and storage

Note: Anthropic API is a dependency of worker services, not dago core.

### Notes
- MVP uses Redis for all infrastructure (events, storage, cache)
- Pure orchestrator - does NOT execute nodes or call LLMs
- Worker pools run as separate services (dago-node-executor, dago-node-router)
- Basic health checks and metrics

## [0.1.0] - 2025-11-15

### Added
- Project initialization
- Basic project structure
- Initial documentation

[Unreleased]: https://github.com/aescanero/dago/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/aescanero/dago/releases/tag/v1.0.0
[0.1.0]: https://github.com/aescanero/dago/releases/tag/v0.1.0
