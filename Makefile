# Check for .env file and include it
ifneq (,$(wildcard .env))
	include .env
	export
endif

# % is a wildcard character that matches anything, so if make doesn't find a rule
# for a target, it uses this rule. The @ beneath is a no-op command. Using these
# together allows us to define a rule that matches anything, but doesn't do anything.
# This is particularly useful when you have targets that take arbitrary user input.
%:
	@:

# ============================================================================= #
# HELPERS
# ============================================================================= #

## help: Print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ============================================================================= #
# QUALITY CONTROL
# ============================================================================= #

## check_staticcheck: Check if staticcheck is installed
.PHONY: check_staticcheck
check_staticcheck:
	@if ! command -v staticcheck >/dev/null 2>&1; then \
		echo "Error: 'staticcheck' is not installed. Installing..."; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi

## security/scan: Run security scan
.PHONY: security/scan
security/scan:
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Error: 'gosec' is not installed. Installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	gosec ./...

## format: Format all Go code with goimports
.PHONY: format
format:
	@echo 'Formatting Go code...'
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@goimports -w -local github.com/bashhack/gojira-mcp $$(find . -name '*.go' -not -path "./vendor/*")
	@echo '✅ Code formatted'

## format/check: Check if code is properly formatted (non-destructive)
.PHONY: format/check
format/check:
	@echo 'Checking Go code formatting...'
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@if [ -n "$$(goimports -l -local github.com/bashhack/gojira-mcp $$(find . -name '*.go' -not -path './vendor/*'))" ]; then \
		echo "❌ The following files need formatting:"; \
		goimports -l -local github.com/bashhack/gojira-mcp $$(find . -name '*.go' -not -path './vendor/*'); \
		echo "Run 'make format' to fix"; \
		exit 1; \
	fi
	@echo '✅ All files properly formatted'

## lint: Run linters without tests
.PHONY: lint
lint:
	@echo 'Formatting code...'
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@goimports -w -local github.com/bashhack/gojira-mcp $$(find . -name '*.go' -not -path "./vendor/*")
	@echo 'Vetting code...'
	go vet ./...
	$(MAKE) check_staticcheck
	staticcheck ./...
	@echo '✅ Linting complete'

## lint/golangci: Run golangci-lint (comprehensive linting tool)
.PHONY: lint/golangci
lint/golangci:
	@echo 'Running golangci-lint...'
	@REQUIRED_VERSION="2.6.2"; \
	INSTALL_NEEDED=false; \
	if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found"; \
		INSTALL_NEEDED=true; \
	else \
		CURRENT_VERSION=$$(golangci-lint --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1); \
		if [ "$$CURRENT_VERSION" != "$$REQUIRED_VERSION" ]; then \
			echo "golangci-lint version $$CURRENT_VERSION found, but v$$REQUIRED_VERSION required"; \
			INSTALL_NEEDED=true; \
		fi; \
	fi; \
	if [ "$$INSTALL_NEEDED" = "true" ]; then \
		echo "Installing golangci-lint v$$REQUIRED_VERSION..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin v$$REQUIRED_VERSION; \
	fi
	@golangci-lint run ./...
	@echo '✅ golangci-lint complete'

## lint/fieldalignment: Show fieldalignment suggestions safely (runs in /tmp, non-destructive)
.PHONY: lint/fieldalignment
lint/fieldalignment:
	@echo '🔍 Checking fieldalignment (safe mode - no files modified)...'
	@if ! command -v fieldalignment >/dev/null 2>&1; then \
		echo "Installing fieldalignment..."; \
		go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest; \
	fi
	@./scripts/fieldalignment_check.sh

## audit: Tidy dependencies and format, vet and test all code
.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@goimports -w -local github.com/bashhack/gojira-mcp $$(find . -name '*.go' -not -path "./vendor/*")
	@echo 'Vetting code...'
	go vet ./...
	$(MAKE) check_staticcheck
	staticcheck ./...
	@echo 'Running tests...'
	go test -short -vet=off ./...
	@echo '✅ Audit complete'

## audit/security: Run security audit
.PHONY: audit/security
audit/security:
	@echo 'Checking for security vulnerabilities...'
	$(MAKE) security/scan
	@echo '✅ Audit complete'

## pre-commit: Run pre-commit checks on all files
.PHONY: pre-commit
pre-commit: format/check
	@echo '🔨 Checking compilation...'
	@go build ./...
	@echo '🔬 Running go vet...'
	@go vet ./...
	@$(MAKE) lint/golangci
	@echo '✅ All pre-commit checks passed!'

## dev/setup/hooks: Install git hooks for pre-commit checks
.PHONY: dev/setup/hooks
dev/setup/hooks:
	@echo 'Installing git hooks...'
	@if [ ! -f .githooks/pre-commit ]; then \
		echo '❌ Error: .githooks/pre-commit not found'; \
		exit 1; \
	fi
	@mkdir -p .git/hooks
	@cp .githooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo '✅ Git hooks installed'
	@echo 'Pre-commit hook will now check formatting and run go vet before each commit'

# ============================================================================= #
# DEVELOPMENT
# ============================================================================= #

## run/tests: Run test suite
.PHONY: run/tests
run/tests:
	go test -v ./...

## run/tests/race: Run test suite with race detection
.PHONY: run/tests/race
run/tests/race:
	go test -v -race -count=1 ./...

## coverage: Run test suite with coverage
.PHONY: coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# ============================================================================= #
# BUILD
# ============================================================================= #

## build: Build gojira-mcp binary
.PHONY: build
build:
	@echo 'Building gojira-mcp...'
	go build -o=./bin/gojira-mcp .

## build/optimize: Build optimized gojira-mcp binary (sans DWARF + symbol table)
.PHONY: build/optimize
build/optimize:
	@echo 'Building optimized gojira-mcp...'
	go build -ldflags='-s -w' -o=./bin/gojira-mcp .

## install: Build and install to GOPATH/bin
.PHONY: install
install:
	@echo 'Installing gojira-mcp...'
	go install .
	@echo '✅ Installed to $$(go env GOPATH)/bin/gojira-mcp'
