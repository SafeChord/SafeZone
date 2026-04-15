.PHONY: help build-all test-all push-all build-% test-% build-tool-% push-% \
        test-worker-golang test-dashboard build-tool-cli build-tool-all smoke-test

# ------------------------
# 0. global variables
# -------------------------
VERSION ?= latest

# -------------------------
# 1. components and their properties
# -------------------------
SERVICE_NAMES := data-ingestor pandemic-simulator analytics-api dashboard worker-golang

data-ingestor_IMAGE_NAME        = safezone-data-ingestor
data-ingestor_PATH              = ./services/data-ingestor

pandemic-simulator_IMAGE_NAME      = safezone-pandemic-simulator
pandemic-simulator_PATH            = ./services/pandemic-simulator

analytics-api_IMAGE_NAME      = safezone-analytics-api
analytics-api_PATH            = ./services/analytics-api

dashboard_IMAGE_NAME      = safezone-dashboard
dashboard_PATH            = ./services/dashboard

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

test-%:
	@echo "====== Testing: $* ======"
	@IMAGE_NAME=$($*_IMAGE_NAME) IMAGE_TAG=$(VERSION)_test BUILD_PATH=$($*_PATH) \
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

# special case for worker-golang
test-worker-golang:
	@echo "====== Testing worker: golang ======"
	@echo "Not implemented yet, skipping..."
	@echo "====== Done: worker-golang ======"

# special case for data-ingestor (only integration-test)
test-data-ingestor:
	@echo "====== Testing: data-ingestor ======"
	@IMAGE_NAME=$(data-ingestor_IMAGE_NAME) IMAGE_TAG=$(VERSION)_test BUILD_PATH=$(data-ingestor_PATH) bash scripts/data-ingestor/unit-test.sh
	@echo "====== Done: data-ingestor ======"

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
# 4. Build/Test ALL
# -------------------------
build-all: $(addprefix build-, $(SERVICE_NAMES))
	@echo "[INFO] ALL SERVICE IMAGES BUILT!"

test-all: $(addprefix test-, $(SERVICE_NAMES))
	@echo "[INFO] ALL TESTS PASSED!"
	
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
	@echo "  test-<service>       Run tests for a specific service (e.g., test-CovidDataIngestor)"
	@echo "  build-<service>      Build a specific service (e.g., build-CovidDataIngestor)"
	@echo "  build-tool-<tool>    Build a specific tool"
	@echo "  test-all             Run tests for all services"
	@echo "  build-all            Build all services/tools"
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