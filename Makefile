.PHONY: help bootstrap install generate fmt license vet lint lint-md \
        test test-generated test-race test-bench test-coverage \
        tidy check-tidy \
        check check-coverage check-vuln \
        build clean

# ─── Colors ──────────────────────────────────────────────────────
BLUE   := $(shell printf "\033[0;36m")
GREEN  := $(shell printf "\033[0;32m")
RED    := $(shell printf "\033[0;31m")
YELLOW := $(shell printf "\033[0;33m")
NC     := $(shell printf "\033[0m")

# ─── Go settings ─────────────────────────────────────────────────
GO      := go
FLAGS   ?=

# ─── Module directories ─────────────────────────────────────────
# Each directory containing a go.mod is a module. Go commands run
# inside each module so multi-module workspaces work correctly.
MODULES := . cmd gen container httptest oteltest clitest

# ─── Paths ───────────────────────────────────────────────────────
BIN_DIR      := bin
COVERAGE_DIR := coverage

# ─── Test tuning ─────────────────────────────────────────────────
# TEST_TIMEOUT applies to test, test-race, and test-bench. Override
# from the command line for slower runners or longer suites:
#   make test TEST_TIMEOUT=30m
TEST_CPU         ?= 4
TEST_COUNT       ?= 3
TEST_TIMEOUT     ?= 10m
TEST_RACE_COUNT  ?= 3

# ─── Build variables ─────────────────────────────────────────────
VERSION    ?= dev
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS    := -ldflags="-X go.thesmos.sh/testkit/cmd/internal/version.buildVersion=$(VERSION) -X go.thesmos.sh/testkit/cmd/internal/version.buildCommit=$(COMMIT) -X go.thesmos.sh/testkit/cmd/internal/version.buildDate=$(BUILD_TIME)"

# ─── License header ──────────────────────────────────────────────
GO_FILES := $(shell find . -type f -name '*.go' \
	! -path './vendor/*' \
	! -path './dist/*' \
	! -path './.git/*' \
	! -name '*.gen.go' \
	! -name '*.gen_test.go')

# ─── Helper: run a command in each module ────────────────────────
# Usage: $(call foreach_module,go test ./...)
define foreach_module
	@for mod in $(MODULES); do \
		echo "$(BLUE)[$${mod}] $(1)$(NC)"; \
		(cd $$mod && $(1)) || exit 1; \
	done
endef

# ─── Help ────────────────────────────────────────────────────────
help:
	@echo "$(BLUE)testkit Build System$(NC)"
	@echo ""
	@echo "$(GREEN)Setup:$(NC)"
	@echo "  bootstrap          Install development tools"
	@echo "  install            Download and verify Go dependencies"
	@echo ""
	@echo "$(GREEN)Development:$(NC)"
	@echo "  fmt                Format Go + Markdown"
	@echo "  license            Apply license headers to all Go files"
	@echo "  lint               Full lint suite (fmt + vet + golangci-lint + markdownlint)"
	@echo "  lint-md            Lint Markdown files only"
	@echo "  vet                Run go vet across all modules"
	@echo "  tidy               Run go mod tidy across all modules"
	@echo "  generate           Run go generate + fmt"
	@echo ""
	@echo "$(GREEN)Testing:$(NC)"
	@echo "  test               Run tests with coverage across all modules"
	@echo "  test-race          Run tests with race detector"
	@echo "  test-bench         Run all benchmarks"
	@echo "  test-coverage      Generate HTML coverage report"
	@echo ""
	@echo "$(GREEN)Quality gates:$(NC)"
	@echo "  check              Full pre-merge gate (tidy + lint + test + coverage)"
	@echo "  check-tidy         Fail if go mod tidy produces uncommitted changes"
	@echo "  check-coverage     Enforce coverage thresholds"
	@echo "  check-vuln         Run govulncheck across all modules"
	@echo ""
	@echo "$(GREEN)Building:$(NC)"
	@echo "  build              Build all modules"
	@echo "  clean              Remove build artifacts and caches"
	@echo ""
	@echo "$(YELLOW)Modules:$(NC) $(MODULES)"
	@echo "$(YELLOW)Flags:$(NC)   FLAGS=\"-run TestFoo\"          extra flags for test commands"
	@echo "          TEST_TIMEOUT=30m              per-package go test deadline"
	@echo "          TEST_COUNT=1                  iteration count for plain tests"
	@echo "          TEST_RACE_COUNT=3             iteration count for test-race"
	@echo "          TEST_CPU=8                    -cpu=N for parallel scheduling"
	@echo ""
	@echo "$(RED)Naming:$(NC)  test-* runs tests; check-* enforces a quality gate"

.DEFAULT_GOAL := help

# ─── Setup ───────────────────────────────────────────────────────

bootstrap:
	@echo "$(BLUE)Installing development tools...$(NC)"
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install github.com/daixiang0/gci@latest
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	$(GO) install github.com/palantir/go-license@latest
	$(GO) install golang.org/x/perf/cmd/benchstat@latest
	@command -v markdownlint-cli2 >/dev/null 2>&1 || { \
		command -v npm >/dev/null 2>&1 && npm install -g markdownlint-cli2 || \
		echo "$(YELLOW)Install markdownlint-cli2: brew install markdownlint-cli2 (or npm install -g markdownlint-cli2)$(NC)"; \
	}
	@echo "$(GREEN)Done. Run 'pre-commit install --hook-type pre-commit --hook-type pre-push --hook-type commit-msg'$(NC)"

install:
	@echo "$(BLUE)Installing dependencies...$(NC)"
	$(call foreach_module,$(GO) mod download && $(GO) mod verify)
	@echo "$(GREEN)Done$(NC)"

# ─── Code generation ─────────────────────────────────────────────

generate:
	@echo "$(BLUE)Building testkit binary...$(NC)"
	cd cmd && $(GO) install ./testkit/
	@echo "$(BLUE)Running code generation...$(NC)"
	$(call foreach_module,$(GO) generate ./...)
	@$(MAKE) fmt
	@echo "$(GREEN)Done$(NC)"

# ─── Formatting ──────────────────────────────────────────────────

fmt: license
	@echo "$(BLUE)Formatting Go...$(NC)"
	gofumpt -l -w -extra .
	gci write --section standard --section default --section "prefix(go.thesmos.sh/testkit)" --custom-order --skip-generated .
	@echo "$(BLUE)Formatting Markdown...$(NC)"
	markdownlint-cli2 --fix "**/*.md" "#vendor" "#dist" "#node_modules" 2>/dev/null || true
	@echo "$(GREEN)Done$(NC)"

license:
	@echo "$(BLUE)Applying license headers...$(NC)"
	@go-license --config=.go-license.yml $(GO_FILES)
	@echo "$(GREEN)Done$(NC)"

# ─── Linting ─────────────────────────────────────────────────────

vet:
	$(call foreach_module,$(GO) vet ./...)

lint: fmt vet lint-md
	@echo "$(BLUE)Running golangci-lint...$(NC)"
	$(call foreach_module,golangci-lint run --timeout=5m ./...)
	@echo "$(BLUE)Verifying license headers...$(NC)"
	@go-license --config=.go-license.yml --verify $(GO_FILES)
	@echo "$(GREEN)Lint passed$(NC)"

lint-md:
	@echo "$(BLUE)Linting Markdown...$(NC)"
	markdownlint-cli2 "**/*.md" "#vendor" "#dist" "#node_modules"

# ─── Generated testdata packages ────────────────────────────────
# go test ./... excludes testdata/ by convention. These packages
# contain generated code with tests that verify the generators
# produce correct, compilable, 100%-covered output.
GEN_TESTDATA := \
	gen/stub/testdata/basic/storetest \
	gen/stub/testdata/directives/storetest \
	gen/stub/testdata/noerror/cachetest \
	gen/stub/testdata/variadic/findertest \
	gen/stub/testdata/namedreturns/servicetest \
	gen/stub/testdata/interfaces/processortest \
	gen/stub/testdata/nocontext/closertest \
	gen/stub/testdata/multireturns/servicetest \
	gen/stub/testdata/companion/storetest \
	gen/stub/testdata/newdirectives/runnertest \
	gen/stub/testdata/iterators/scannertest \
	gen/builder/testdata/basic/basictest \
	gen/builder/testdata/defaults/defaultstest \
	gen/builder/testdata/fielddefaults/fielddefaultstest \
	gen/builder/testdata/generics/genericstest \
	gen/builder/testdata/nested/nestedtest \
	gen/sentinel/testdata/basic \
	gen/enum/testdata/basic \
	gen/suite/testdata/basic/storetest \
	gen/suite/testdata/nocontext/cachetest \
	gen/suite/testdata/multireturn/servicetest \
	gen/suite/testdata/mixed/processortest \
	gen/suite/testdata/erroronly/closertest \
	gen/suite/testdata/iterators/scannertest \
	gen/suite/testdata/readers/registrytest \
	gen/suite/testdata/writers/storetest \
	gen/suite/testdata/allshapes/servicetest \
	gen/suite/testdata/weird/weirdtest

# ─── Testing ─────────────────────────────────────────────────────

test: test-generated
	@echo "$(BLUE)Running tests (timeout=$(TEST_TIMEOUT))...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	$(call foreach_module,$(GO) test -coverprofile=$(CURDIR)/$(COVERAGE_DIR)/$$(basename $$PWD).out -covermode=atomic -cpu=$(TEST_CPU) -count=$(TEST_COUNT) -timeout=$(TEST_TIMEOUT) $(FLAGS) ./...)
	@echo "$(GREEN)Tests passed$(NC)"

test-generated:
	@echo "$(BLUE)Running generated testdata tests...$(NC)"
	@for pkg in $(GEN_TESTDATA); do \
		relpath=$$(echo $$pkg | sed 's|^gen/||'); \
		echo "$(BLUE)  $$relpath$(NC)"; \
		(cd gen && GOWORK=off $(GO) test -count=1 ./$$relpath/) || exit 1; \
	done
	@echo "$(GREEN)Generated tests passed$(NC)"

test-race:
	@echo "$(BLUE)Running tests with race detector (count=$(TEST_RACE_COUNT), timeout=$(TEST_TIMEOUT))...$(NC)"
	$(call foreach_module,$(GO) test -race -count=$(TEST_RACE_COUNT) -timeout=$(TEST_TIMEOUT) $(FLAGS) ./...)
	@echo "$(GREEN)No races detected$(NC)"

test-bench:
	@echo "$(BLUE)Running benchmarks (timeout=$(TEST_TIMEOUT))...$(NC)"
	$(call foreach_module,$(GO) test -bench=. -run=^$$ -benchmem -timeout=$(TEST_TIMEOUT) $(FLAGS) ./...)

test-coverage: test
	@echo "$(BLUE)Generating coverage reports...$(NC)"
	@for mod in $(MODULES); do \
		name=$$(basename $$mod); \
		if [ "$$mod" = "." ]; then name="testkit"; fi; \
		if [ -f $(COVERAGE_DIR)/$$name.out ]; then \
			$(GO) tool cover -html=$(COVERAGE_DIR)/$$name.out -o $(COVERAGE_DIR)/$$name.html; \
			echo "$(GREEN)Report: $(COVERAGE_DIR)/$$name.html$(NC)"; \
		fi \
	done

# ─── Quality gates ───────────────────────────────────────────────

check-coverage:
	@printf "$(BLUE)Running tests …$(NC) "
	@logf=$$(mktemp); \
	if $(MAKE) --no-print-directory -s test >$$logf 2>&1; then \
		printf "$(GREEN)ok$(NC)\n\n"; \
		rm -f $$logf; \
	else \
		printf "$(RED)FAILED$(NC)\n\n"; \
		cat $$logf; \
		rm -f $$logf; \
		exit 1; \
	fi

check-vuln:
	$(call foreach_module,govulncheck ./...)

# ─── Building ────────────────────────────────────────────────────

build:
	$(call foreach_module,$(GO) build ./...)

# ─── Cleanup ─────────────────────────────────────────────────────

clean:
	@echo "$(BLUE)Cleaning...$(NC)"
	rm -rf $(BIN_DIR) $(COVERAGE_DIR) dist/
	$(call foreach_module,$(GO) clean -cache -testcache)
	@echo "$(GREEN)Clean$(NC)"

# ─── Module hygiene ─────────────────────────────────────────────

tidy:
	$(call foreach_module,$(GO) mod tidy)

check-tidy: tidy
	@if ! git diff --quiet -- '*/go.mod' '*/go.sum' go.mod go.sum go.work.sum; then \
		echo "$(RED)go mod tidy produced changes. Run 'make tidy' and commit.$(NC)"; \
		git diff --stat -- '*/go.mod' '*/go.sum' go.mod go.sum go.work.sum; \
		exit 1; \
	fi

# ─── CI gate ─────────────────────────────────────────────────────

check: generate check-tidy lint test check-coverage
	@echo "$(GREEN)All checks passed$(NC)"
