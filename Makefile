export GO111MODULE ?= on
PACKAGES = $(shell go list ./...)
PACKAGES_PATH = $(shell go list -f '{{ .Dir }}' ./...)

.PHONY: all
all: check_tools ensure-deps gofumpt imports linter test

.PHONY: check_tools
check_tools:
	@type "golangci-lint" > /dev/null 2>&1 || echo 'Please install golangci-lint: https://golangci-lint.run/usage/install/#local-installation'
	@type "goimports" > /dev/null 2>&1 || echo 'Please install goimports: go get golang.org/x/tools/cmd/goimports'
	@type "gofumpt" > /dev/null 2>&1 || echo 'Please install gofumpt: go install mvdan.cc/gofumpt@latest'

.PHONY: ensure-deps
ensure-deps:
	@echo "=> Syncing dependencies with go mod tidy"
	@go mod tidy

.PHONY: gofumpt
gofumpt:
	@echo "=> Executing gofumpt"
	@gofumpt -l -w .

.PHONY: imports
imports:
	@echo "=> Executing goimports"
	@goimports -w $(PACKAGES_PATH)

# Runs golangci-lint with arguments if provided.
.PHONY: linter
linter:
	@echo "=> Executing golangci-lint$(if $(FLAGS), with flags: $(FLAGS))"
	@golangci-lint run ./... $(FLAGS)

.PHONY: test
test:
	@echo "=> Running tests"
	@go test ./... -covermode=atomic -coverpkg=./... -count=1 -race

# Acceptance tests (Gherkin/godog). They start the API in-process against a Postgres
# container, so Docker must be running (or set ACCEPTANCE_DATABASE_URL). See SPEC-008.
# GODOG_TAGS and GODOG_CONCURRENCY tune the run, e.g. GODOG_TAGS='@update-all-next'.
.PHONY: test-acceptance
test-acceptance:
	@echo "=> Running acceptance tests"
	@go test -tags acceptance -count=1 ./test/acceptance/...

# Only scenarios under construction (@wip), verbose: for writing a new scenario.
.PHONY: test-acceptance-wip
test-acceptance-wip:
	@echo "=> Running @wip acceptance scenarios"
	@GODOG_TAGS='@wip' go test -tags acceptance -count=1 -v ./test/acceptance/...

# Only @known-bug scenarios: desired behavior the API does not have yet. A scenario that
# passes here means the bug was fixed: remove its tag.
.PHONY: test-acceptance-known-bugs
test-acceptance-known-bugs:
	@echo "=> Running @known-bug acceptance scenarios"
	@GODOG_TAGS='@known-bug' go test -tags acceptance -count=1 ./test/acceptance/...
