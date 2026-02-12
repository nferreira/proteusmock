BINARY := proteusmock
CMD    := ./cmd/proteusmock

.DEFAULT_GOAL := help

# ── Build & Run ──────────────────────────────────────────────────────

.PHONY: build build-dashboard run dev-dashboard

build-dashboard: ## Build the React dashboard SPA
	cd ui/dashboard && npm ci && npm run build

build: build-dashboard ## Build the binary to bin/proteusmock (includes dashboard)
	go build -o bin/$(BINARY) $(CMD)

run: ## Run the server (--root ./mock --port 8080)
	go run $(CMD) --root ./mock --port 8080

dev-dashboard: ## Run Go server + Vite dev server for dashboard development
	@echo "Start Vite dev server: cd ui/dashboard && npm run dev"
	@echo "Start Go server:       go run $(CMD) --root ./mock --port 8080"
	@echo "Dashboard at: http://localhost:5173/__ui/"

# ── Testing ──────────────────────────────────────────────────────────

.PHONY: test test-integration test-e2e test-all test-race test-cover

test: ## Run unit tests only (fast)
	go test ./...

test-integration: ## Run unit + integration tests
	go test -tags=integration -count=1 ./...

test-e2e: ## Run E2E tests only
	go test -tags=e2e -count=1 ./test/e2e/...

test-all: ## Run all tests (unit + integration + E2E) with race detector
	go test -tags="integration,e2e" -race -count=1 ./...

test-race: ## Run unit tests with race detector
	go test -race -count=1 ./...

test-cover: ## Run unit + integration tests with coverage report
	go test -tags=integration -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo "---"
	@echo "To view HTML report: go tool cover -html=coverage.out"

# ── Code Quality ─────────────────────────────────────────────────────

.PHONY: fmt imports vet lint golangci-lint

fmt: ## Format code with gofmt
	gofmt -w .

imports: ## Format imports with goimports
	@command -v goimports >/dev/null 2>&1 && goimports -w -local github.com/sophialabs/proteusmock . || echo "goimports not installed: go install golang.org/x/tools/cmd/goimports@latest"

vet: ## Run go vet
	go vet ./...

lint: vet ## Run staticcheck (includes go vet)
	staticcheck ./...

golangci-lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed: see https://golangci-lint.run/welcome/install/"

# ── Demo ─────────────────────────────────────────────────────────────

.PHONY: showcase

showcase: build ## Start server and run all showcase scenarios with pretty output
	@if curl -s -o /dev/null -w "" http://localhost:8080/api/v1/health 2>/dev/null; then \
		echo "Server already running on port 8080, skipping startup."; \
		./scripts/showcase.sh 8080; \
	else \
		echo "Starting ProteusMock server..."; \
		bin/$(BINARY) --root ./mock --port 8080 & \
		MOCK_PID=$$!; \
		trap "kill $$MOCK_PID 2>/dev/null; wait $$MOCK_PID 2>/dev/null" EXIT; \
		./scripts/showcase.sh 8080; \
		kill $$MOCK_PID 2>/dev/null; wait $$MOCK_PID 2>/dev/null; \
	fi

# ── Docker ───────────────────────────────────────────────────────────

IMAGE_NAME  := sophialabs/proteusmock
IMAGE_TAG   := latest

.PHONY: docker-build docker-run docker-test docker-push compose-up compose-down

docker-build: ## Build Docker production image
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

docker-run: ## Run Docker container (port 8080, bundled mocks)
	docker run --rm -p 8080:8080 $(IMAGE_NAME):$(IMAGE_TAG)

docker-test: ## Build Docker image running all tests
	docker build --build-arg TEST_TAGS="integration,e2e" -t $(IMAGE_NAME):test .

docker-push: ## Push image to registry
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

compose-up: ## Start services with docker compose
	docker compose up --build -d

compose-down: ## Stop docker compose services
	docker compose down

# ── Housekeeping ─────────────────────────────────────────────────────

.PHONY: clean help

clean: ## Remove build artifacts and coverage files
	rm -rf bin/ coverage.out ui/dashboard/dist/ ui/dashboard/node_modules/

help: ## Show this help
	@printf "\nUsage: make \033[36m<target>\033[0m\n"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
