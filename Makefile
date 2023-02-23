BINDIR ?= /usr/local/bin
PATH := $(BINDIR):$(PATH)

ARCH ?= amd64

GO ?= go

GOIMPORTS_VERSION ?= 0.3.0
GOIMPORTS ?= goimports
GOIMPORTS_LOCAL ?= github.com/fornellas/resonance/

STATICCHECK_VERSION ?= 2023.1
STATICCHECK ?= staticcheck
STATICCHECK_CACHE ?= $(HOME)/.cache/staticcheck

GOCYCLO ?= gocyclo
GOCYCLO_VERSION = v0.6.0
GOCYCLO_OVER ?= 10

GO_TEST ?= gotest
GO_TEST_FLAGS ?= -v -race -cover -count=1
GOTEST_VERSION ?= v0.0.6

RRB_VERSION ?= latest
RRB_DEBOUNCE ?= 500ms
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

.PHONY: install-deps-bindir
install-deps-bindir:
	mkdir -p $(BINDIR)

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

.PHONY: install-deps-goimports
install-deps-goimports: install-deps-bindir
	GOBIN=$(BINDIR) $(GO) install golang.org/x/tools/cmd/goimports@v$(GOIMPORTS_VERSION)
install-deps: install-deps-goimports

.PHONY: uninstall-deps-goimports
uninstall-deps-goimports:
	rm -f $(BINDIR)/goimports
uninstall-deps: uninstall-deps-goimports

.PHONY: goimports
goimports:
	$(BINDIR)/$(GOIMPORTS) -w -local $(GOIMPORTS_LOCAL) $$(find . -name \*.go ! -path './.cache/*')
lint: goimports

# go mod tidy

.PHONY: go-mod-tidy
go-mod-tidy: go-generate goimports
	$(GO) mod tidy
lint: go-mod-tidy

# staticcheck

.PHONY: install-deps-staticcheck
install-deps-staticcheck: install-deps-bindir
	rm -rf $(BINDIR)/staticcheck $(BINDIR)/staticcheck.tmp && \
		curl -sSfL  https://github.com/dominikh/go-tools/releases/download/$(STATICCHECK_VERSION)/staticcheck_linux_$(ARCH).tar.gz | \
		tar -zx -C $(BINDIR) staticcheck/staticcheck && \
		mv $(BINDIR)/staticcheck $(BINDIR)/staticcheck.tmp && \
		mv $(BINDIR)/staticcheck.tmp/staticcheck $(BINDIR)/ && \
		rmdir $(BINDIR)/staticcheck.tmp
install-deps: install-deps-staticcheck

.PHONY: uninstall-deps-staticcheck
uninstall-deps-staticcheck:
	rm -f $(BINDIR)/staticcheck
uninstall-deps: uninstall-deps-staticcheck

.PHONY: staticcheck
staticcheck: go-mod-tidy go-generate
	$(STATICCHECK) ./...
lint: staticcheck

.PHONY: clean-staticcheck
clean-staticcheck:
	rm -rf $(HOME)/.cache/staticcheck/
clean: clean-staticcheck

# gocyclo

.PHONY: install-deps-gocyclo
install-deps-gocyclo: install-deps-bindir
	GOBIN=$(BINDIR) $(GO) install github.com/fzipp/gocyclo/cmd/gocyclo@$(GOCYCLO_VERSION)
install-deps: install-deps-gocyclo

.PHONY: uninstall-deps-gocyclo
uninstall-deps-gocyclo:
	rm -f $(BINDIR)/gocyclo
uninstall-deps: uninstall-deps-gocyclo

.PHONY: gocyclo
gocyclo: go-generate go-mod-tidy
	$(BINDIR)/$(GOCYCLO) -over $(GOCYCLO_OVER) -avg .
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
	@echo 'test: runs all tests'
help: test-help

.PHONY: test
test:

# gotest

.PHONY: install-deps-gotest
install-deps-gotest: install-deps-bindir
	GOBIN=$(BINDIR) $(GO) install github.com/rakyll/gotest@$(GOTEST_VERSION)
install-deps: install-deps-gotest

.PHONY: uninstall-deps-gotest
uninstall-deps-gotest:
	rm -f $(BINDIR)/gotest
uninstall-deps: uninstall-deps-gotest

.PHONY: gotest
gotest: go-generate
	$(GO_TEST) ./... $(GO_TEST_FLAGS)
test: gotest

.PHONY: clean-gotest
clean-gotest:
	$(GO) clean -r -testcache
clean: clean-gotest

# ci

.PHONY: ci-help
ci-help:
	@echo 'ci: runs the whole build'
help: ci-help

.PHONY: ci-no-install-deps
ci-no-install-deps: lint test build

.PHONY: ci
ci: install-deps ci-no-install-deps

# rrb

.PHONY: install-deps-rrb
install-deps-rrb: install-deps-bindir
	GOBIN=$(BINDIR) $(GO) install github.com/fornellas/rrb@$(RRB_VERSION)
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
		--pattern $(RRB_PATTERN) \
		-- \
		sh -c "$(MAKE) $(MFLAGS) ci-no-install-deps && $(RRB_EXTRA_CMD)"

.PHONY: rrb
rrb:
	rrb \
		--debounce $(RRB_DEBOUNCE) \
		--pattern Makefile \
		-- \
		$(MAKE) $(MFLAGS) install-deps rrb-ci-no-install-deps

##
## Build
##

.PHONY: build-help
build-help:
	@echo 'build: build everything'
help: build-help

.PHONY: build
build: go-generate
	$(GO) build .

.PHONY: clean-build
clean-build:
	$(GO) clean -r -cache -modcache
clean: clean-build