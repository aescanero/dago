GO           ?= go
GOLANGCI_VER ?= v1.62.0
ATLAS_VER    ?= v0.27.0
MODULE       := github.com/aescanero/dago

BIN_DIR := bin

.DEFAULT_GOAL := help

.PHONY: help
help: ## List all available targets with descriptions
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-25s %s\n", $$1, $$2}'

.PHONY: build-all
build-all: ## Compile all 8 service binaries into bin/
	$(GO) build -o $(BIN_DIR)/orchestrator   ./services/orchestrator/cmd/
	$(GO) build -o $(BIN_DIR)/executor       ./services/executor/cmd/
	$(GO) build -o $(BIN_DIR)/router         ./services/router/cmd/
	$(GO) build -o $(BIN_DIR)/planner        ./services/planner/cmd/
	$(GO) build -o $(BIN_DIR)/auth-server    ./services/auth-server/cmd/
	$(GO) build -o $(BIN_DIR)/catalog        ./services/catalog/cmd/
	$(GO) build -o $(BIN_DIR)/mcp-registry   ./services/mcp-registry/cmd/
	$(GO) build -o $(BIN_DIR)/agent-registry ./services/agent-registry/cmd/

.PHONY: build-orchestrator
build-orchestrator: ## Compile the orchestrator service
	$(GO) build -o $(BIN_DIR)/orchestrator ./services/orchestrator/cmd/

.PHONY: build-executor
build-executor: ## Compile the executor service
	$(GO) build -o $(BIN_DIR)/executor ./services/executor/cmd/

.PHONY: build-router
build-router: ## Compile the router service
	$(GO) build -o $(BIN_DIR)/router ./services/router/cmd/

.PHONY: build-planner
build-planner: ## Compile the planner service
	$(GO) build -o $(BIN_DIR)/planner ./services/planner/cmd/

.PHONY: build-auth-server
build-auth-server: ## Compile the auth-server service
	$(GO) build -o $(BIN_DIR)/auth-server ./services/auth-server/cmd/

.PHONY: build-catalog
build-catalog: ## Compile the catalog service
	$(GO) build -o $(BIN_DIR)/catalog ./services/catalog/cmd/

.PHONY: build-mcp-registry
build-mcp-registry: ## Compile the mcp-registry service
	$(GO) build -o $(BIN_DIR)/mcp-registry ./services/mcp-registry/cmd/

.PHONY: build-agent-registry
build-agent-registry: ## Compile the agent-registry service
	$(GO) build -o $(BIN_DIR)/agent-registry ./services/agent-registry/cmd/

.PHONY: test-llm
test-llm: ## Run LLM adapter unit tests (anthropic + fake + ollama)
	$(GO) test -count=1 -timeout 30s ./adapters/llm/...

## test-executor: executor service unit tests
.PHONY: test-executor
test-executor: ## Run executor service unit tests (no network, no credentials)
	$(GO) test -count=1 -timeout 30s ./services/executor/...

.PHONY: test
test: test-llm ## Run all tests with go test ./...
	$(GO) test ./...

.PHONY: test-unit
test-unit: ## Run unit tests in tests/unit/, libs/, adapters/
	$(GO) test ./tests/unit/... ./libs/... ./adapters/...

.PHONY: test-integration-eventbus
test-integration-eventbus: ## Run eventbus integration tests (requires Docker)
	$(GO) test -tags integration -count=1 -timeout 120s \
	  ./adapters/eventbus/...

.PHONY: test-integration
test-integration: test-integration-eventbus ## Run all integration tests with -tags=integration
	@echo "All integration tests passed"

.PHONY: test-smoke
test-smoke: ## Run smoke tests with -tags=smoke
	$(GO) test -tags=smoke ./tests/smoke/...

.PHONY: lint
lint: ## Run golangci-lint on all packages
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code with goimports
	goimports -w .

.PHONY: generate
generate: ## Run go generate for Ent code generation
	$(GO) generate ./...

.PHONY: migrate-diff
migrate-diff: ## Compute Atlas migration diff from Ent schema
	atlas migrate diff --env local

.PHONY: migrate-apply
migrate-apply: ## Apply Atlas migrations to local database
	atlas migrate apply --env local

.PHONY: migrate-test
migrate-test: ## Execute all migrations against a throwaway postgres (runtime check, not just lint)
	@echo "migrate-test: starting throwaway postgres on port 5433…"
	@docker rm -f dago-migrate-test 2>/dev/null || true
	@docker run -d --name dago-migrate-test \
		-e POSTGRES_USER=dago -e POSTGRES_PASSWORD=dago -e POSTGRES_DB=dago \
		-p 5433:5432 pgvector/pgvector:pg16 >/dev/null
	@echo "migrate-test: waiting for postgres to be ready…"
	@until docker exec dago-migrate-test pg_isready -U dago -q 2>/dev/null; do sleep 1; done
	@echo "migrate-test: applying migrations…"
	@atlas migrate apply \
		--dir "file://migrations?format=golang-migrate" \
		--url "postgres://dago:dago@localhost:5433/dago?sslmode=disable" \
		--allow-dirty; \
	STATUS=$$?; docker rm -f dago-migrate-test >/dev/null 2>&1; \
	[ $$STATUS -eq 0 ] && echo "migrate-test: OK" || echo "migrate-test: FAILED"; \
	exit $$STATUS

.PHONY: tools
tools: ## Install pinned versions of golangci-lint and atlas CLI
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VER)
	curl -sSf https://atlasgo.sh | sh -s -- --version $(ATLAS_VER) -y

.PHONY: docker-up
docker-up: ## Start PostgreSQL and Valkey via Docker Compose
	docker compose up -d

.PHONY: docker-down
docker-down: ## Stop and remove Docker Compose services
	docker compose down

.PHONY: bootstrap
bootstrap: ## Install tools, start docker services, download Go dependencies
	$(MAKE) tools
	$(MAKE) docker-up
	$(GO) mod download

.PHONY: ci
ci: ## Run full CI pipeline: lint + build-all + test
	$(MAKE) lint
	$(MAKE) build-all
	$(MAKE) test

.PHONY: dashboard-dev
dashboard-dev: ## Start the dashboard development server
	cd dashboard && npm run dev

.PHONY: gen-api-types
gen-api-types: ## Generate TypeScript types from specs/openapi.yaml
	cd dashboard && npm run gen:api

.PHONY: dashboard-check
dashboard-check: ## Run dashboard smoke checks: build, type-check, lint, tests
	cd dashboard && bash scripts/smoke.sh

.PHONY: compose-infra
compose-infra: ## Start only postgres + valkey
	docker compose --profile infra up -d

.PHONY: compose-up
compose-up: ## Start full stack (all profiles) with build
	docker compose --profile all up -d --build

.PHONY: compose-down
compose-down: ## Stop and remove all containers
	docker compose --profile all down

.PHONY: compose-logs
compose-logs: ## Tail logs for all running services
	docker compose logs -f

.PHONY: compose-ps
compose-ps: ## Show status of all containers
	docker compose ps

.PHONY: clean
clean: ## Remove compiled binaries from bin/
	rm -rf $(BIN_DIR)
