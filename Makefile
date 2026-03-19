
.PHONY: help install i test test-v test-coverage lint lint-go format clean build run go-install uninstall sync-schema release patch minor major prep-release

MODULE_PATH=github.com/ariel-frischer/rentalot-cli
VERSION?=$(shell git tag --sort=-v:refname 2>/dev/null | head -1)
ifeq ($(VERSION),)
  VERSION=dev
endif
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags="-X ${MODULE_PATH}/internal/version.Version=${VERSION} \
                   -X ${MODULE_PATH}/internal/version.Commit=${COMMIT} \
                   -X ${MODULE_PATH}/internal/version.BuildDate=${BUILD_DATE} \
                   -s -w"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Download dependencies
	go mod download

i: install ## Alias for install

go-install: ## Install rentalot to GOPATH/bin
	go install ${LDFLAGS} ./cmd/rentalot/

test: ## Run tests
	go test ./...

test-v: ## Run tests (verbose)
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -race -coverprofile=coverage.out ./...

lint: lint-go ## Run all linters

lint-go: ## Run Go linters
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running go vet"; \
		go vet ./...; \
	fi

format: ## Format code
	go fmt ./...

clean: ## Clean build artifacts
	go clean
	rm -rf bin/ coverage.out

build: ## Build binary with version info
	go build ${LDFLAGS} -o bin/rentalot ./cmd/rentalot/

run: ## Run main package
	go run ${LDFLAGS} ./cmd/rentalot/

uninstall: ## Uninstall rentalot
	@./uninstall.sh

sync-schema: ## Sync OpenAPI spec from running rentalot dev server (make dev in ../rentalot)
	@mkdir -p internal/api
	@curl -sf http://localhost:3000/api/v1/openapi.json | jq . > internal/api/openapi.json
	@echo "Wrote internal/api/openapi.json"

##@ Release
prep-release: ## Full release flow (usage: make prep-release VERSION=v0.1.0)
	@./scripts/release.sh $(VERSION)

release: ## Create a release tag and push (usage: make release VERSION=v1.0.0)
	@if [ "$(VERSION)" = "dev" ] || [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required (e.g., make release VERSION=v1.0.0)"; \
		exit 1; \
	fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)

patch: ## Bump patch version and release
	$(eval CURRENT=$(shell git tag --sort=-v:refname | head -1 | sed 's/^v//'))
	$(eval NEXT=v$(shell echo $(CURRENT) | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3+1}'))
	@echo "Bumping $(CURRENT) -> $(NEXT)"
	$(MAKE) release VERSION=$(NEXT)

minor: ## Bump minor version and release
	$(eval CURRENT=$(shell git tag --sort=-v:refname | head -1 | sed 's/^v//'))
	$(eval NEXT=v$(shell echo $(CURRENT) | awk -F. '{printf "%d.%d.0", $$1, $$2+1}'))
	@echo "Bumping $(CURRENT) -> $(NEXT)"
	$(MAKE) release VERSION=$(NEXT)

major: ## Bump major version and release
	$(eval CURRENT=$(shell git tag --sort=-v:refname | head -1 | sed 's/^v//'))
	$(eval NEXT=v$(shell echo $(CURRENT) | awk -F. '{printf "%d.0.0", $$1+1}'))
	@echo "Bumping $(CURRENT) -> $(NEXT)"
	$(MAKE) release VERSION=$(NEXT)
