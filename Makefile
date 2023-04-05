help:

MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))

SHELL := /bin/bash
.ONESHELL:

XDG_CACHE_HOME ?= $(HOME)/.cache
CACHE_DIR ?= $(XDG_CACHE_HOME)/resonance/build-cache
BINDIR := $(CACHE_DIR)/bin
BINDIR := $(BINDIR)
.PHONY: BINDIR
BINDIR:
	@echo $(BINDIR)
PATH := $(BINDIR):$(PATH)

GO := go
export GOBIN := $(BINDIR)
.PHONY: GOBIN
GOBIN:
	@echo $(GOBIN)
export GOCACHE := $(CACHE_DIR)/go-build
.PHONY: GOCACHE
GOCACHE:
	@echo $(GOCACHE)
export GOMODCACHE := $(CACHE_DIR)/go-mod
.PHONY: GOMODCACHE
GOMODCACHE:
	@echo $(GOMODCACHE)
GOARCH := $(shell go env GOARCH)
ifneq ($(.SHELLSTATUS),0)
  $(error shell command failed! output was $(var))
endif
GOOS := $(shell go env GOOS)
ifneq ($(.SHELLSTATUS),0)
  $(error shell command failed! output was $(var))
endif
# osusergo have Lookup and LookupGroup to use pure Go implementation to enable
# management of local users
GO_BUILD_FLAGS := -tags osusergo
GOARCHS_BUILD := 386 amd64 arm arm64

GOIMPORTS_LOCAL := github.com/fornellas/resonance/

STATICCHECK_CACHE := $(CACHE_DIR)/staticcheck

GOCYCLO_OVER := 15

GO_TEST := gotest
GO_TEST_FLAGS := -race -coverprofile cover.txt -coverpkg ./... -count=1 -failfast
ifeq ($(V),1)
GO_TEST_FLAGS := -v $(GO_TEST_FLAGS)
endif
GOTEST_VERSION := v0.0.6

RRB_VERSION := latest
RRB_DEBOUNCE ?= 500ms
RRB_LOG_LEVEL ?= info
RRB_PATTERN ?= '**/*.{go}'
RRB_EXTRA_CMD ?= true

##
## Help
##

.PHONY: help
help:

##
## Clean
##

.PHONY: clean
clean:
clean-help:
	@echo 'clean: clean all files'
help: clean-help


##
## Install Deps
##

.PHONY: install-deps-help
install-deps-help:
	@echo 'install-deps: install dependencies required by the build at BINDIR=$(BINDIR)'
help: install-deps-help

$(BINDIR):
	@mkdir -p $(BINDIR)

.PHONY: install-deps
install-deps:

.PHONY: uninstall-deps-help
uninstall-deps-help:
	@echo 'uninstall-deps: uninstall dependencies required by the build'
help: uninstall-deps-help

.PHONY: uninstall-deps
uninstall-deps:

##
## Lint
##

# lint

.PHONY: lint-help
lint-help:
	@echo 'lint: runs all linters'
help: lint-help

.PHONY: lint
lint:

# Generate

.PHONY: go-generate
go-generate:
	$(GO) generate ./...

# goimports

.PHONY: goimports
goimports:
	$(GO) run golang.org/x/tools/cmd/goimports -w -local $(GOIMPORTS_LOCAL) $$(find . -name \*.go ! -path './.cache/*')
lint: goimports

# go mod tidy

.PHONY: go-mod-tidy
go-mod-tidy: go-generate goimports
	$(GO) mod tidy
lint: go-mod-tidy

# go
.PHONY: go-get-u
go-get-u: go-mod-tidy
	$(GO) get -u ./...

# staticcheck

.PHONY: staticcheck
staticcheck: go-mod-tidy go-generate goimports
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...
lint: staticcheck

.PHONY: clean-staticcheck
clean-staticcheck:
	rm -rf $(HOME)/.cache/staticcheck/
clean: clean-staticcheck

# misspell

.PHONY: misspell
misspell: go-mod-tidy go-generate
	$(GO) run github.com/client9/misspell/cmd/misspell -error .
lint: misspell

.PHONY: clean-misspell
clean-misspell:
	rm -rf $(HOME)/.cache/misspell/
clean: clean-misspell

# gocyclo

.PHONY: gocyclo
gocyclo: go-generate go-mod-tidy
	$(GO) run github.com/fzipp/gocyclo/cmd/gocyclo -over $(GOCYCLO_OVER) -avg .
lint: gocyclo

# go vet

.PHONY: go-vet
go-vet: go-mod-tidy go-generate
	$(GO) vet ./...
lint: go-vet

##
## Test
##

# test

.PHONY: test-help
test-help:
	@echo 'test: runs all tests; use V=1 for verbose'
help: test-help

.PHONY: test
test:

# gotest

.PHONY: gotest
gotest: go-generate
	$(GO) run github.com/rakyll/gotest ./... $(GO_TEST_FLAGS) $(GO_BUILD_FLAGS)
test: gotest

.PHONY: clean-gotest
clean-gotest:
	$(GO) clean -r -testcache
	rm -f cover.txt cover.html
clean: clean-gotest

# cover.html

.PHONY: cover.html
cover.html: gotest
	$(GO) tool cover -html cover.txt -o cover.html
test: cover.html

.PHONY: clean-cover.html
clean-cover.html:
	rm -f cover.html
clean: clean-cover.html

# cover-func

.PHONY: cover-func
cover-func: cover.html
	@echo -n "Coverage: "
	@$(GO) tool cover -func cover.txt | awk '/^total:/{print $$NF}'
test: cover-func

##
## Build
##

.PHONY: build-help
build-help:
	@echo 'build: build everything'
help: build-help

.PHONY: build-goarchs
build-goarchs:
	@echo $(foreach GOARCH,$(GOARCHS_BUILD),$(GOARCH))

.PHONY: build-%
build-%: go-generate
	GOARCH=$* $(GO) build -o resonance.$(GOOS).$* $(GO_BUILD_FLAGS) .

.PHONY: build
build: $(foreach GOARCH,$(GOARCHS_BUILD),build-$(GOARCH))

.PHONY: clean-build-%
clean-build-%:
	rm -f resonance.$(GOOS).$*
clean: $(foreach GOARCH,$(GOARCHS_BUILD),clean-build-$(GOARCH))

.PHONY: clean-build
clean-build:
	$(GO) clean -r -cache -modcache
	rm -f version/.version
clean: clean-build

##
## ci
##

.PHONY: ci-help
ci-help:
	@echo 'ci: runs the whole build'
help: ci-help

.PHONY: ci-no-install-deps
ci-no-install-deps: lint test build

.PHONY: ci
ci: install-deps ci-no-install-deps

##
## rrb
##

.PHONY: install-deps-rrb
install-deps-rrb: $(BINDIR)
	@if [ $(BINDIR)/rrb -ot $(MAKEFILE_PATH) ] ; then \
		echo Installing rrb ; \
		$(GO) install github.com/fornellas/rrb@$(RRB_VERSION) ; \
	fi
install-deps: install-deps-rrb

.PHONY: uninstall-deps-rrb
uninstall-deps-rrb:
	rm -f $(BINDIR)/rrb
uninstall-deps: uninstall-deps-rrb

.PHONY: rrb-help
rrb-help:
	@echo 'rrb: rerun build automatically on file changes then runs RRB_EXTRA_CMD'
help: rrb-help

.PHONY: rrb-ci-no-install-deps
rrb-ci-no-install-deps:
	rrb \
		--debounce $(RRB_DEBOUNCE) \
		--ignore-pattern '.cache/**/*' \
		--log-level $(RRB_LOG_LEVEL) \
		--pattern $(RRB_PATTERN) \
		-- \
		sh -c "$(MAKE) $(MFLAGS) ci-no-install-deps && $(RRB_EXTRA_CMD)"

.PHONY: rrb
rrb: install-deps-rrb
	rrb \
		--debounce $(RRB_DEBOUNCE) \
		--log-level $(RRB_LOG_LEVEL) \
		--pattern Makefile \
		-- \
		$(MAKE) $(MFLAGS) install-deps rrb-ci-no-install-deps