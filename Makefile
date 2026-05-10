MAKEFLAGS += --warn-undefined-variables
SHELL := bash
GO_VERSION ?=
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := all
.DELETE_ON_ERROR:
.SUFFIXES:

.PHONY: audit
audit: tidy fmt
	go vet ./...
	go tool -modfile=go.tool.mod staticcheck ./...
	go tool -modfile=go.tool.mod govulncheck ./...
	golangci-lint run -v


.PHONY: build
build:
	goreleaser build --clean --single-target --snapshot

.PHONY: clean
clean:
ifneq (,$(wildcard ./dist))
	rm -rf dist/

endif

ifneq (,$(wildcard ./coverage))
	rm -rf coverage/

endif

.PHONY: fmt
fmt:
	golines --base-formatter=gofumpt -w .
	go tool -modfile=go.tool.mod gofumpt -l -w -extra .

.PHONY: lint
lint:
	golangci-lint run -v

.PHONY: modernize
modernize:
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix -test ./...

.PHONY: test
test:
	go test -covermode=atomic -coverprofile=coverdata/coverage.out ./... && echo 'Coverage data collected'
	go tool cover -html=coverdata/coverage.out -o coverdata/coverage.html

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: update
update:
	go get -u ./...
	go get -u -modfile=go.tool.mod tool
	go mod tidy

.PHONY: update-go-version
update-go-version:
	@if [ -z "$(or $(GO_VERSION),$(version))" ]; then \
		echo "Usage: make update-go-version GO_VERSION=1.25.10"; \
		echo "   or: make update-go-version version=1.25.10"; \
		exit 1; \
	fi
	./scripts/update-go-version.sh "$(or $(GO_VERSION),$(version))"
