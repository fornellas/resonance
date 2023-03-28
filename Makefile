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

ARCH := amd64

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
# osusergo have Lookup and LookupGroup to use pure Go implementation to enable
# management of local users
GO_BUILD_FLAGS := -tags osusergo
GOARCHS := 386 amd64 arm arm64

GOIMPORTS_VERSION := 0.3.0
GOIMPORTS := goimports
GOIMPORTS_LOCAL := github.com/fornellas/resonance/

STATICCHECK_VERSION := 2023.1
STATICCHECK := staticcheck
STATICCHECK_CACHE := $(CACHE_DIR)/staticcheck

MISSPELL_VERSION := v0.3.4
MISSPELL := misspell

GOCYCLO := gocyclo
GOCYCLO_VERSION := v0.6.0
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

.PHONY: install-deps-goimports
install-deps-goimports: $(BINDIR)
	@if [ $(BINDIR)/goimports -ot $(MAKEFILE_PATH) ] ; then \
		echo Installing goimports ; \
		$(GO) install golang.org/x/tools/cmd/goimports@v$(GOIMPORTS_VERSION) ; \
	fi
install-deps: install-deps-goimports

.PHONY: uninstall-deps-goimports
uninstall-deps-goimports:
	rm -f $(BINDIR)/goimports
uninstall-deps: uninstall-deps-goimports

.PHONY: goimports
goimports: install-deps-goimports
	$(BINDIR)/$(GOIMPORTS) -w -local $(GOIMPORTS_LOCAL) $$(find . -name \*.go ! -path './.cache/*')
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
lint: go-get-u

# staticcheck

.PHONY: install-deps-staticcheck
install-deps-staticcheck: $(BINDIR)
	@if [ $(BINDIR)/staticcheck -ot $(MAKEFILE_PATH) ] || ! [ -f $(BINDIR)/staticcheck ]; then \
		echo Installing staticcheck ; \
		rm -rf $(BINDIR)/staticcheck $(BINDIR)/staticcheck.tmp && \
			curl -sSfL  https://github.com/dominikh/go-tools/releases/download/$(STATICCHECK_VERSION)/staticcheck_linux_$(ARCH).tar.gz | \
			tar -zx -C $(BINDIR) staticcheck/staticcheck && \
			mv $(BINDIR)/staticcheck $(BINDIR)/staticcheck.tmp && \
			touch $(BINDIR)/staticcheck.tmp/staticcheck && \
			mv $(BINDIR)/staticcheck.tmp/staticcheck $(BINDIR)/ && \
			rmdir $(BINDIR)/staticcheck.tmp ; \
	fi
install-deps: install-deps-staticcheck

.PHONY: uninstall-deps-staticcheck
uninstall-deps-staticcheck:
	rm -f $(BINDIR)/staticcheck
uninstall-deps: uninstall-deps-staticcheck

.PHONY: staticcheck
staticcheck: install-deps-staticcheck go-mod-tidy go-generate goimports
	$(STATICCHECK) ./...
lint: staticcheck

.PHONY: clean-staticcheck
clean-staticcheck:
	rm -rf $(HOME)/.cache/staticcheck/
clean: clean-staticcheck

# misspell

.PHONY: install-deps-misspell
install-deps-misspell: $(BINDIR)
	@if [ $(BINDIR)/misspell -ot $(MAKEFILE_PATH) ] ; then \
		echo Installing misspell ; \
		$(GO) install github.com/client9/misspell/cmd/misspell@$(MISSPELL_VERSION) ; \
	fi
install-deps: install-deps-misspell

.PHONY: uninstall-deps-misspell
uninstall-deps-misspell:
	rm -f $(BINDIR)/misspell
uninstall-deps: uninstall-deps-misspell

.PHONY: misspell
misspell: install-deps-misspell go-mod-tidy go-generate
	$(MISSPELL) -error .
lint: misspell

.PHONY: clean-misspell
clean-misspell:
	rm -rf $(HOME)/.cache/misspell/
clean: clean-misspell

# gocyclo

.PHONY: install-deps-gocyclo
install-deps-gocyclo: $(BINDIR)
	@if [ $(BINDIR)/gocyclo -ot $(MAKEFILE_PATH) ] ; then \
		echo Installing gocyclo ; \
		$(GO) install github.com/fzipp/gocyclo/cmd/gocyclo@$(GOCYCLO_VERSION) ; \
	fi
install-deps: install-deps-gocyclo

.PHONY: uninstall-deps-gocyclo
uninstall-deps-gocyclo:
	rm -f $(BINDIR)/gocyclo
uninstall-deps: uninstall-deps-gocyclo

.PHONY: gocyclo
gocyclo: install-deps-gocyclo go-generate go-mod-tidy
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
	@echo 'test: runs all tests; use V=1 for verbose'
help: test-help

.PHONY: test
test:

# gotest

.PHONY: install-deps-gotest
install-deps-gotest: $(BINDIR)
	@if [ $(BINDIR)/gotest -ot $(MAKEFILE_PATH) ] ; then \
		echo Installing gotest ; \
		$(GO) install github.com/rakyll/gotest@$(GOTEST_VERSION) ; \
	fi
install-deps: install-deps-gotest

.PHONY: uninstall-deps-gotest
uninstall-deps-gotest:
	rm -f $(BINDIR)/gotest
uninstall-deps: uninstall-deps-gotest

.PHONY: gotest
gotest: install-deps-gotest go-generate
	$(GO_TEST) ./... $(GO_TEST_FLAGS) $(GO_BUILD_FLAGS)
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
	@echo $(foreach GOARCH,$(GOARCHS),$(GOARCH))

.PHONY: build-agent
build-agent:

.PHONY: host/agent/agent_linux_%
host/agent/agent_linux_%: go-generate
	GOARCH=$* GOOS=linux $(GO) build -o host/agent/agent_linux_$* $(GO_BUILD_FLAGS) ./host/agent/
build-agent: $(foreach GOARCH,$(GOARCHS),host/agent/agent_linux_$(GOARCH))

.PHONY: clean-host/agent/agent_linux_%
clean-host/agent/agent_linux_%:
	rm -f host/agent/agent_linux_$*
clean: $(foreach GOARCH,$(GOARCHS),clean-host/agent/agent_linux_-$(GOARCH))


.PHONY: host/agent/agent_linux_%.gz
host/agent/agent_linux_%.gz: host/agent/agent_linux_%
	gzip < host/agent/agent_linux_$* > host/agent/agent_linux_$*.gz
build-agent: $(foreach GOARCH,$(GOARCHS),host/agent/agent_linux_$(GOARCH).gz)

.PHONY: clean-host/agent/agent_linux_%.gz
clean-host/agent/agent_linux_%.gz:
	rm -f host/agent/agent_linux_$*.gz
clean: $(foreach GOARCH,$(GOARCHS),clean-host/agent/agent_linux_-$(GOARCH).gz)

.PHONY: host/agent_linux_%_gz.go
host/agent_linux_%_gz.go: host/agent/agent_linux_%.gz
	cat << EOF > host/agent_linux_$*_gz.go
	package host
	import _ "embed"
	//go:embed agent/agent_linux_$*.gz
	var agent_linux_$* []byte
	func init() {
		AgentBinGz["linux.$*"] = agent_linux_$*
	}
	EOF
build-agent: $(foreach GOARCH,$(GOARCHS),host/agent_linux_$(GOARCH)_gz.go)

.PHONY: clean-host/agent_linux_%_gz.go
clean-host/agent_linux_%_gz.go:
	rm -f host/agent_linux_$*_gz.go
clean: $(foreach GOARCH,$(GOARCHS),clean-host/agent_linux_$(GOARCH)_gz.go)
build: $(foreach GOARCH,$(GOARCHS),clean-host/agent_linux_$(GOARCH)_gz.go)
go-generate: $(foreach GOARCH,$(GOARCHS),clean-host/agent_linux_$(GOARCH)_gz.go)
goimports: $(foreach GOARCH,$(GOARCHS),clean-host/agent_linux_$(GOARCH)_gz.go)
go-mod-tidy: $(foreach GOARCH,$(GOARCHS),clean-host/agent_linux_$(GOARCH)_gz.go)
go-get-u: $(foreach GOARCH,$(GOARCHS),clean-host/agent_linux_$(GOARCH)_gz.go)
staticcheck: $(foreach GOARCH,$(GOARCHS),clean-host/agent_linux_$(GOARCH)_gz.go)
misspell: $(foreach GOARCH,$(GOARCHS),clean-host/agent_linux_$(GOARCH)_gz.go)
gocyclo: $(foreach GOARCH,$(GOARCHS),clean-host/agent_linux_$(GOARCH)_gz.go)
go-vet: $(foreach GOARCH,$(GOARCHS),clean-host/agent_linux_$(GOARCH)_gz.go)

.PHONY: build-%
build-%: go-generate build-agent
	GOARCH=$* $(GO) build -o resonance.$$(go env GOOS).$* $(GO_BUILD_FLAGS) .

.PHONY: build
build: $(foreach GOARCH,$(GOARCHS),build-$(GOARCH))

.PHONY: clean-build-%
clean-build-%:
	rm -f resonance.$$(go env GOOS).$*
clean: $(foreach GOARCH,$(GOARCHS),clean-build-$(GOARCH))

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
		--ignore-pattern 'host/agent_*_*_gz.go' \
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