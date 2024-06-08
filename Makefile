help:

SHELL := /bin/bash
.ONESHELL:

XDG_CACHE_HOME ?= $(HOME)/.cache
RESONANCE_CACHE ?= $(XDG_CACHE_HOME)/rrb

export GOVERSION := go$(shell cat go.mod | awk '/^go /{print $$2}')
ifneq ($(.SHELLSTATUS),0)
  $(error cat .goversion failed! output was $(GOVERSION))
endif

GOOS_SHELL := case $$(uname -s) in Linux) echo linux;; Darwin) echo darwin;; *) echo Unknown system $$(uname -s) 1>&2 ; exit 1 ;; esac
export GOOS ?= $(shell $(GOOS_SHELL))
ifneq ($(.SHELLSTATUS),0)
  $(error GOOS failed! output was $(GOOS))
endif
.PHONY: GOOS
GOOS:
	@echo $(GOOS)

GOARCH_NATIVE_SHELL := case $$(uname -m) in i[23456]86) echo 386;; x86_64) echo amd64;; armv6l|armv7l) echo arm;; aarch64) echo arm64;; *) echo Unknown machine $$(uname -m) 1>&2 ; exit 1 ;; esac
GOARCH_NATIVE := $(shell $(GOARCH_NATIVE_SHELL))
ifneq ($(.SHELLSTATUS),0)
  $(error GOARCH failed! output was $(GOARCH))
endif

export GOARCH ?= $(GOARCH_NATIVE)
ifneq ($(.SHELLSTATUS),0)
  $(error GOARCH_NATIVE failed! output was $(GOARCH_NATIVE))
endif

GOARCH_DOWNLOAD_SHELL := case $(GOARCH_NATIVE) in 386) echo 386;; amd64) echo amd64;; arm) echo armv6l;; arm64) echo arm64;; *) echo GOARCH $$(GOARCH_NATIVE) 1>&2 ; exit 1 ;; esac
GOARCH_DOWNLOAD ?= $(shell $(GOARCH_DOWNLOAD_SHELL))
export GOARCH ?= $(GOARCH_DOWNLOAD)
ifneq ($(.SHELLSTATUS),0)
  $(error GOARCH_DOWNLOAD failed! output was $(GOARCH_DOWNLOAD))
endif

.PHONY: GOARCH
GOARCH:
	@echo $(GOARCH)

GOROOT_PREFIX := $(RESONANCE_CACHE)/GOROOT
GOROOT := $(GOROOT_PREFIX)/$(GOVERSION).$(GOOS)-$(GOARCH_DOWNLOAD)
GO := $(GOROOT)/bin/go
.PHONY: GOROOT
GOROOT:
	@echo $(GOROOT)
PATH := $(GOROOT)/bin:$(PATH)

export GOCACHE := $(RESONANCE_CACHE)/GOCACHE
.PHONY: GOCACHE
GOCACHE:
	@echo $(GOCACHE)
export GOMODCACHE := $(RESONANCE_CACHE)/GOMODCACHE

.PHONY: GOMODCACHE
GOMODCACHE:
	@echo $(GOMODCACHE)

# osusergo have Lookup and LookupGroup to use pure Go implementation to enable
# management of local users
GO_BUILD_FLAGS := -tags osusergo

GOARCHS_AGENT := 386 amd64 arm arm64

export GO_MODULE := $(shell cat go.mod | awk '/^module /{print $$2}')

GOIMPORTS := $(GO) run golang.org/x/tools/cmd/goimports
GOIMPORTS_LOCAL := $(GO_MODULE)

STATICCHECK := $(GO) run honnef.co/go/tools/cmd/staticcheck

GOCYCLO := $(GO) run github.com/fzipp/gocyclo/cmd/gocyclo
GOCYCLO_OVER := 15

GO_TEST := $(GO) run github.com/rakyll/gotest ./...
GO_TEST_FLAGS := -coverprofile cover.txt -coverpkg ./... -count=1 -failfast
ifeq ($(V),1)
GO_TEST_FLAGS := -v $(GO_TEST_FLAGS)
endif
# https://go.dev/doc/articles/race_detector#Requirements
ifeq ($(GOOS)/$(GOARCH),linux/amd64)
GO_TEST_FLAGS := -race $(GO_TEST_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),linux/ppc64le)
GO_TEST_FLAGS := -race $(GO_TEST_FLAGS)
endif
# https://github.com/golang/go/issues/29948
# ifeq ($(GOOS)/$(GOARCH),linux/arm64)
# GO_TEST_FLAGS := -race $(GO_TEST_FLAGS)
# endif
ifeq ($(GOOS)/$(GOARCH),freebsd/amd64)
GO_TEST_FLAGS := -race $(GO_TEST_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),netbsd/amd64)
GO_TEST_FLAGS := -race $(GO_TEST_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),darwin/amd64)
GO_TEST_FLAGS := -race $(GO_TEST_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),darwin/arm64)
GO_TEST_FLAGS := -race $(GO_TEST_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),windows/amd64)
GO_TEST_FLAGS := -race $(GO_TEST_FLAGS)
endif

GCOV2LCOV := $(GO) run github.com/jandelgado/gcov2lcov

RRB := $(GO) run github.com/fornellas/rrb
RRB_DEBOUNCE ?= 500ms
RRB_LOG_LEVEL ?= info
RRB_IGNORE_PATTERN ?= '.cache/**/*,host/agent_*_*_gz.go'
RRB_PATTERN ?= '**/*.{go},Makefile'
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
## Go
##

.PHONY: go
go:
	set -e
	if [ -d $(GOROOT) ] ; then exit ; fi
	rm -rf $(GOROOT_PREFIX)/go
	mkdir -p $(GOROOT_PREFIX)
	curl -sSfL  https://go.dev/dl/$(GOVERSION).$(GOOS)-$(GOARCH_DOWNLOAD).tar.gz | \
		tar -zx -C $(GOROOT_PREFIX) && \
		touch $(GOROOT_PREFIX)/go &&
		mv $(GOROOT_PREFIX)/go $(GOROOT)

.PHONY: clean-go
clean-go:
	rm -rf $(GOROOT_PREFIX)
	rm -rf $(GOCACHE)
	find $(GOMODCACHE) -print0 | xargs -0 chmod u+w
	rm -rf $(GOMODCACHE)
clean: clean-go

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
go-generate: go
	$(GO) generate ./...

# go mod tidy

.PHONY: go-mod-tidy
go-mod-tidy: go go-generate
	$(GO) mod tidy
lint: go-mod-tidy

# goimports

.PHONY: goimports
goimports: go go-mod-tidy
	$(GOIMPORTS) -w -local $(GOIMPORTS_LOCAL) $$(find . -name \*.go ! -path './.cache/*')
lint: goimports

# staticcheck

.PHONY: staticcheck
staticcheck: go go-mod-tidy go-generate goimports
	$(STATICCHECK) $(GO_MODULE)/...
lint: staticcheck

.PHONY: clean-staticcheck
clean-staticcheck:
	rm -rf $(HOME)/.cache/staticcheck/
clean: clean-staticcheck

# misspell

.PHONY: misspell
misspell: go go-mod-tidy go-generate
	$(GO) run github.com/client9/misspell/cmd/misspell -error .
lint: misspell

.PHONY: clean-misspell
clean-misspell:
	rm -rf $(HOME)/.cache/misspell/
clean: clean-misspell

# gocyclo

.PHONY: gocyclo
gocyclo: go go-generate go-mod-tidy
	$(GOCYCLO) -over $(GOCYCLO_OVER) -avg .
lint: gocyclo

# go vet

.PHONY: go-vet
go-vet: go go-mod-tidy go-generate
	$(GO) vet ./...
lint: go-vet

# go get -u

.PHONY: go-get-u
go-get-u: go go-mod-tidy
	$(GO) get -u ./...

##
## Test
##

# test

.PHONY: test-help
test-help:
	@echo 'test: runs all tests; use V=1 for verbose'
help: test-help

.PHONY: test
test: build-agent-$(GOARCH_NATIVE)

# gotest

.PHONY: gotest
gotest: go go-generate
	$(GO_TEST) $(GO_TEST_FLAGS) $(GO_BUILD_FLAGS)
test: gotest

.PHONY: clean-gotest
clean-gotest:
	$(GO) env &>/dev/null && $(GO) clean -r -testcache
	rm -f cover.txt cover.html
clean: clean-gotest

# cover.html

.PHONY: cover.html
cover.html: go gotest
	$(GO) tool cover -html cover.txt -o cover.html
test: cover.html

.PHONY: clean-cover.html
clean-cover.html:
	rm -f cover.html
clean: clean-cover.html

# cover.lcov

.PHONY: cover.lcov
cover.lcov: go gotest
	$(GCOV2LCOV) -infile cover.txt -outfile cover.lcov
test: cover.lcov

.PHONY: clean-cover.lcov
clean-cover.lcov:
	rm -f cover.lcov
clean: clean-cover.lcov

# cover-func

.PHONY: cover-func
cover-func: go cover.html
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

.PHONY: build-agent-%
build-agent-%: go-generate
	GOARCH=$* GOOS=linux $(GO) build -o host/agent/agent_linux_$* $(GO_BUILD_FLAGS) ./host/agent/
	gzip < host/agent/agent_linux_$* > host/agent/agent_linux_$*.gz
	cat << EOF > host/agent_linux_$*_gz.go
	package host
	import _ "embed"
	//go:embed agent/agent_linux_$*.gz
	var agent_linux_$* []byte
	func init() {
		AgentBinGz["linux.$*"] = agent_linux_$*
	}
	EOF
build-agent: $(foreach GOARCH,$(GOARCHS_AGENT),build-agent-$(GOARCH))

.PHONY: clean-agent-%
clean-agent-%:
	rm -f host/agent/agent_linux_$*
	rm -f host/agent/agent_linux_$*.gz
	rm -rf host/agent_linux_$*_gz.go
clean-agent: $(foreach GOARCH,$(GOARCHS_AGENT),clean-agent-$(GOARCH))
clean: clean-agent
build: clean-agent
go-generate: clean-agent
goimports: clean-agent
go-mod-tidy: clean-agent
go-get-u: clean-agent
staticcheck: clean-agent
misspell: clean-agent
gocyclo: clean-agent
go-vet: clean-agent

.PHONY: build
build: go go-generate build-agent
	$(GO) build -o resonance.$(GOOS).$(GOARCH) $(GO_BUILD_FLAGS) .

.PHONY: clean-build
clean-build:
	$(GO) env &>/dev/null && $(GO) clean -r -cache -modcache
	rm -f version/.version
	rm -f resonance.*.*
clean: clean-build

##
## ci
##

.PHONY: ci-help
ci-help:
	@echo 'ci: runs the whole build'
help: ci-help

.PHONY: ci
ci: lint test build

##
## rrb
##

ifeq ($(GOOS),linux)

.PHONY: rrb-help
rrb-help:
	@echo 'rrb: rerun build automatically on file changes then runs RRB_EXTRA_CMD'
help: rrb-help

.PHONY: rrb
rrb: go
	$(RRB) \
		--debounce $(RRB_DEBOUNCE) \
		--ignore-pattern $(RRB_IGNORE_PATTERN) \
		--log-level $(RRB_LOG_LEVEL) \
		--pattern $(RRB_PATTERN) \
		-- \
		sh -c "$(MAKE) $(MFLAGS) ci && $(RRB_EXTRA_CMD)"

##
## shell
##

.PHONY: shell-help
shell-help:
	@echo 'shell: starts a development shell'
help: shell-help

.PHONY: shell
shell:
	@echo Make targets:
	@$(MAKE) help
	@PATH=$(GOROOT)/bin:$(PATH) \
		GOOS=$(GOOS) \
		GOARCH=$(GOARCH) \
		GOROOT=$(GOROOT) \
		GOCACHE=$(GOCACHE) \
		GOMODCACHE=$(GOMODCACHE) \
		bash --rcfile .bashrc

endif