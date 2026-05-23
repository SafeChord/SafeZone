.PHONY: help build-all push-all build-% test-% build-tool-% push-% \
        test-worker-golang test-dashboard build-tool-cli build-tool-all smoke-test \
        ci-all ci-% ci-dashboard-v2 ci-worker-golang

# ------------------------
# 0. global variables
# -------------------------
VERSION ?= latest

# -------------------------
# 1. components and their properties
# -------------------------
SERVICE_NAMES := data-ingestor pandemic-simulator analytics-api dashboard dashboard-v2 worker-golang

data-ingestor_IMAGE_NAME        = safezone-data-ingestor
data-ingestor_PATH              = ./services/data-ingestor

pandemic-simulator_IMAGE_NAME      = safezone-pandemic-simulator
pandemic-simulator_PATH            = ./services/pandemic-simulator

analytics-api_IMAGE_NAME      = safezone-analytics-api
analytics-api_PATH            = ./services/analytics-api

dashboard_IMAGE_NAME      = safezone-dashboard
dashboard_PATH            = ./services/dashboard

dashboard-v2_IMAGE_NAME      = safezone-dashboard-v2
dashboard-v2_PATH            = ./services/dashboard-v2

worker-golang_IMAGE_NAME      = safezone-worker
worker-golang_PATH            = ./services/worker-golang

# -------------------------
# 2. tools and their properties
# -------------------------
TOOL_NAMES := cli time-server

# the building process of cli is complex, so we handle it separately
# refer to scripts/cli/*.sh for details 
cli_IMAGE_NAME           = safezone-cli-command # not used in the makefile
cli_PATH                 = ./toolkit/cli/command # not used in the makefile

time-server_IMAGE_NAME        = safezone-time-server
time-server_PATH              = ./toolkit/time-server

# -------------------------
# 3. smoke test and its properties
# -------------------------
SMOKE_COMPOSE_FILE      = ./docker-compose/local-compose-all.yml

# -------------------------
# 3. Pattern Rule for build/test
# -------------------------
build-%:
	@echo "====== Building: $* ======"
	@IMAGE_NAME=$($*_IMAGE_NAME) IMAGE_TAG=$(VERSION) BUILD_PATH=$($*_PATH) \
		bash scripts/build-image.sh
	@echo "====== Done: $* ======"

test-%: build-%
	@echo "====== Testing: $* ======"
	@IMAGE_NAME=$($*_IMAGE_NAME) IMAGE_TAG=$(VERSION)_test BUILD_PATH=$($*_PATH) VERSION=$(VERSION) \
		bash scripts/unit-test.sh
	@echo "====== Done: $* ======"

build-tool-%:
	@echo "====== Building tool: $* ======"
	@IMAGE_NAME=$($*_IMAGE_NAME) IMAGE_TAG=$(VERSION) BUILD_PATH=$($*_PATH) \
		bash scripts/build-image.sh
	@echo "====== Done tool: $* ======"

push-%:
	@echo "====== Pushing: $* ======"
	@IMAGE_NAME=$($*_IMAGE_NAME) IMAGE_TAG=$(VERSION) bash scripts/push-image.sh
	@echo "====== Done: $* ======"

# special case for worker-golang (Go: build test image from source, not base binary)
test-worker-golang:
	@echo "====== Testing: worker-golang ======"
	@docker buildx build -t safezone-worker:$(VERSION)_test -f services/worker-golang/Dockerfile.test .
	@docker run --rm safezone-worker:$(VERSION)_test
	@docker rmi safezone-worker:$(VERSION)_test || true
	@echo "====== Done: worker-golang ======"

# special case for dashboard-v2 (Node/vitest)
test-dashboard-v2:
	@echo "====== Testing: dashboard-v2 ======"
	@IMAGE_NAME=$(dashboard-v2_IMAGE_NAME) IMAGE_TAG=$(VERSION)_test BUILD_PATH=$(dashboard-v2_PATH) VERSION=$(VERSION) \
		bash scripts/dashboard-v2/unit-test.sh
	@echo "====== Done: dashboard-v2 ======"

# special case for dashboard (only unit-test)
test-dashboard:
	@echo "====== Testing: dashboard ======"
	@IMAGE_NAME=$(dashboard_IMAGE_NAME) IMAGE_TAG=$(VERSION)_test BUILD_PATH=$(dashboard_PATH) bash scripts/dashboard/unit-test.sh
	@echo "====== Done: dashboard ======"

# special case for cli
build-tool-cli:
	@echo "====== Building tool: cli ======"
	@IMAGE_TAG=$(VERSION) bash scripts/cli/build.sh
	@echo "====== Done tool: cli ======"

push-cli:
	@echo "====== Pushing tool: cli ======"
	@IMAGE_TAG=$(VERSION) bash scripts/cli/push-image.sh
	@echo "====== Done tool: cli ======"

# -------------------------
# 4. Aggregate targets
# -------------------------
build-all: $(addprefix build-, $(SERVICE_NAMES))
	@echo "[INFO] ALL SERVICE IMAGES BUILT!"

# -------------------------
# 4.1 CI orchestration (hides language-specific build/test ordering)
# -------------------------
# Python: test-% depends on build-% (overlay pattern), so ci = test
ci-data-ingestor ci-pandemic-simulator ci-analytics-api ci-dashboard: ci-%: test-%

# TypeScript: test first (independent), then build prod image
ci-dashboard-v2: test-dashboard-v2 build-dashboard-v2

# Go: test first (independent), then build prod image
ci-worker-golang: test-worker-golang build-worker-golang

ci-all: $(addprefix ci-, $(SERVICE_NAMES))
	@echo "[INFO] ALL CI PIPELINES PASSED!"
	
build-tool-all: $(addprefix build-tool-, $(TOOL_NAMES))
	@echo "[INFO] ALL TOOL IMAGES BUILDED!"

push-all: $(addprefix push-, $(SERVICE_NAMES)) $(addprefix push-, $(TOOL_NAMES))
	@echo "[INFO] ALL SERVICE/TOOL IMAGES PUSHED!"

# -------------------------
# 5. End-to-End Tests
# -------------------------
smoke-test:
	@echo "====== Running: End-to-End Smoke Test ======"
	@COMPOSE_FILE=$(SMOKE_COMPOSE_FILE) VERSION=$(VERSION) \
		bash scripts/smoke-test.sh
	@echo "====== Done: Smoke Test ======"

# -------------------------
# 6. Local CI
# ------------------------- 
local-ci: 
	act -W .github/workflows/smoke-test.local.yml
	@echo "[INFO] LOCAL CI COMPLETED!"
	@tput cnorm

# -------------------------
# 7. Local Development Environment
# -------------------------
dev-up:
	@echo "====== Starting Local Dev Environment ======"
	@COMPOSE_FILE=$(SMOKE_COMPOSE_FILE) VERSION=latest \
		bash scripts/dev-up.sh

dev-down:
	@echo "====== Stopping Local Dev Environment ======"
	@COMPOSE_FILE=$(SMOKE_COMPOSE_FILE) VERSION=latest \
		bash scripts/dev-down.sh

dev-restart: dev-down dev-up
	@echo "====== Environment Restarted ======"

dev-logs:
	@echo "====== Following Logs (Ctrl+C to exit) ======"
	@docker compose -f $(SMOKE_COMPOSE_FILE) logs -f

# -------------------------
# 8. Help
# -------------------------
help:
	@echo "Available targets:"
	@echo "  local-ci             Run local CI tests"
	@echo "  ci-<service>         Run full CI for a service (e.g., ci-analytics-api)"
	@echo "  ci-all               Run CI for all services (correct build/test order per language)"
	@echo "  test-<service>       Run tests for a specific service (e.g., test-analytics-api)"
	@echo "  build-<service>      Build a specific service (e.g., build-analytics-api)"
	@echo "  build-tool-<tool>    Build a specific tool"
	@echo "  build-tool-all       Build all tools"
	@echo "  push-<service/tool>  Push a specific service/tool image to the registry"
	@echo "  push-all             Push all service/tool images to the registry"
	@echo "  local-ci-compose     Run local CI with compose job"
	@echo "  local-ci-down        Bring down the local CI environment"
	@echo "  help                 Show this help message"
	@echo "  dev-up               Start the local development environment"
	@echo "  dev-down             Stop the local development environment"
	@echo "  dev-restart          Restart the local development environment"
	@echo "  dev-logs             Follow logs of the local development environment"