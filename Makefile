help:

SHELL := /bin/bash
.ONESHELL:

MAKE_MAJOR_VERSION := $(word 1, $(subst ., , $(MAKE_VERSION)))
MAKE_REQUIRED_MAJOR_VERSION := 4
MAKE_BAD_VERSION := $(shell [ $(MAKE_MAJOR_VERSION) -lt $(MAKE_REQUIRED_MAJOR_VERSION) ] && echo true)
ifeq ($(MAKE_BAD_VERSION),true)
  $(error Make version is below $(MAKE_REQUIRED_MAJOR_VERSION), please update it.)
endif

SHELL_UNAME_S := uname -s
UNAME_S := $(shell $(SHELL_UNAME_S))
ifneq ($(.SHELLSTATUS),0)
$(error $(SHELL_UNAME_S): $(UNAME_S))
endif

SHELL_UNAME_M := uname -m
UNAME_M := $(shell $(SHELL_UNAME_M))
ifneq ($(.SHELLSTATUS),0)
$(error $(SHELL_UNAME_M): $(UNAME_M))
endif

ifeq ($(UNAME_S),Linux)
XDG_CACHE_HOME ?= $(HOME)/.cache
else
ifeq ($(UNAME_S),Darwin)
XDG_CACHE_HOME ?= $(HOME)/Library/Caches
else
$(error Unsupported system: $(UNAME_S))
endif
endif

CACHE_PATH ?= $(XDG_CACHE_HOME)/resonance

SHELL_GO_VERSION := cat go.mod | awk '/^go /{print $$2}'
export GOVERSION := go$(shell $(SHELL_GO_VERSION))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GO_VERSION): $(GOVERSION))
endif

SHELL_GOOS := case $(UNAME_S) in Linux) echo linux;; Darwin) echo darwin;; *) echo Unknown system $(UNAME_S) 1>&2 ; exit 1 ;; esac
export GOOS ?= $(shell $(SHELL_GOOS))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GOOS): $(GOOS))
endif

SHELL_GOARCH_NATIVE := case $(UNAME_M) in i[23456]86) echo 386;; x86_64) echo amd64;; armv6l|armv7l) echo arm;; aarch64|arm64) echo arm64;; *) echo Unknown machine $(UNAME_M) 1>&2 ; exit 1 ;; esac
GOARCH_NATIVE := $(shell $(SHELL_GOARCH_NATIVE))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GOARCH_NATIVE): $(GOARCH_NATIVE))
endif

export GOARCH ?= $(GOARCH_NATIVE)

SHELL_GOARCH_DOWNLOAD := case $(GOARCH_NATIVE) in 386) echo 386;; amd64) echo amd64;; arm) echo armv6l;; arm64) echo arm64;; *) echo GOARCH $(GOARCH_NATIVE) 1>&2 ; exit 1 ;; esac
GOARCH_DOWNLOAD ?= $(shell $(SHELL_GOARCH_DOWNLOAD))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GOARCH_DOWNLOAD): $(GOARCH_DOWNLOAD))
endif


GOROOT_PREFIX := $(CACHE_PATH)/GOROOT
GOROOT := $(GOROOT_PREFIX)/$(GOVERSION).$(GOOS)-$(GOARCH_DOWNLOAD)
GO := $(GOROOT)/bin/go
PATH := $(GOROOT)/bin:$(PATH)

export GOCACHE := $(CACHE_PATH)/GOCACHE

export GOMODCACHE := $(CACHE_PATH)/GOMODCACHE

GO_BUILD_FLAGS_COMMON :=
# osusergo have Lookup and LookupGroup to use pure Go implementation to enable
# management of local users
GO_BUILD_FLAGS_COMMON := -tags osusergo

define get_go_build_flags
$(value GO_BUILD_FLAGS_$(1))
endef

# https://go.dev/doc/articles/race_detector#Requirements
ifneq ($(GO_BUILD_FLAGS_NO_RACE),1)
ifeq ($(GOOS)/$(GOARCH),linux/amd64)
GO_BUILD_FLAGS_linux_amd64 := -race $(GO_BUILD_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),linux/ppc64le)
GO_BUILD_FLAGS_linux_ppc64le := -race $(GO_BUILD_FLAGS)
endif
# https://github.com/golang/go/issues/29948
# ifeq ($(GOOS)/$(GOARCH),linux/arm64)
#_LINUX_ARM64 GO_BUILD_FLAGS := -race $(GO_BUILD_FLAGS)
# endif
ifeq ($(GOOS)/$(GOARCH),freebsd/amd64)
GO_BUILD_FLAGS_freebsd_amd64 := -race $(GO_BUILD_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),netbsd/amd64)
GO_BUILD_FLAGS_netbsd_amd64 := -race $(GO_BUILD_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),darwin/amd64)
GO_BUILD_FLAGS_darwin_amd64 := -race $(GO_BUILD_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),darwin/arm64)
GO_BUILD_FLAGS_darwin_arm64 := -race $(GO_BUILD_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),windows/amd64)
GO_BUILD_FLAGS_windows_amd64 := -race $(GO_BUILD_FLAGS)
endif
endif

GOARCHS_AGENT := 386 amd64 arm arm64

SHELL_GO_MODULE := cat go.mod | awk '/^module /{print $$2}'
export GO_MODULE := $(shell $(SHELL_GO_MODULE))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GO_MODULE): $(GO_MODULE))
endif

GO_SOURCE_FILES := $$(find $$PWD -name \*.go ! -path '$(CACHE_PATH)/*')

GOIMPORTS := $(GO) run golang.org/x/tools/cmd/goimports
GOIMPORTS_LOCAL := $(GO_MODULE)

STATICCHECK := $(GO) run honnef.co/go/tools/cmd/staticcheck
export STATICCHECK_CACHE := $(CACHE_PATH)/staticcheck

GOCYCLO := $(GO) run github.com/fzipp/gocyclo/cmd/gocyclo
GOCYCLO_OVER := 15

GO_TEST := $(GO) run github.com/rakyll/gotest
GO_TEST_FLAGS :=
GO_TEST_PACKAGES := ./...
GO_TEST_BINARY_FLAGS :=
ifneq ($(GO_TEST_NO_COVER),1)
GO_TEST_BINARY_FLAGS := -coverprofile cover.txt -coverpkg $(GO_TEST_PACKAGES) $(GO_TEST_BINARY_FLAGS)
endif
GO_TEST_BINARY_FLAGS := -count=1 $(GO_TEST_BINARY_FLAGS)
GO_TEST_BINARY_FLAGS := -failfast $(GO_TEST_BINARY_FLAGS)
GO_TEST_MIN_COVERAGE := 67

GCOV2LCOV := $(GO) run github.com/jandelgado/gcov2lcov

RRB := $(GO) run github.com/fornellas/rrb
RRB_DEBOUNCE ?= 500ms
RRB_LOG_LEVEL ?= info
RRB_IGNORE_PATTERN ?= '$(CACHE_PATH)/**/*,host/agent_*_*_gz.go'
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
	$(GOIMPORTS) -w -local $(GOIMPORTS_LOCAL) $(GO_SOURCE_FILES)
lint: goimports

# staticcheck

.PHONY: staticcheck
staticcheck: go go-mod-tidy go-generate goimports
	$(STATICCHECK) $(GO_MODULE)/...
lint: staticcheck

.PHONY: clean-staticcheck
clean-staticcheck:
	rm -rf $(STATICCHECK_CACHE)
clean: clean-staticcheck

# misspell

.PHONY: misspell
misspell: go go-mod-tidy go-generate
	$(GO) run github.com/client9/misspell/cmd/misspell -error $(GO_SOURCE_FILES)
lint: misspell

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
	@echo 'test: runs all tests:'
	@echo '  use GO_TEST_NO_COVER=1 to disable code coverage'
	@echo '  use GO_TEST_PACKAGES to set packages to test (default: $(GO_TEST_PACKAGES))'
	@echo '  use GO_TEST_BINARY_FLAGS_EXTRA to pass extra flags to the test binary (eg: -v)'
help: test-help

.PHONY: test

# gotest

.PHONY: gotest
gotest: go go-generate
	$(GO_TEST) \
		$(GO_BUILD_FLAGS_COMMON) \
		$(call get_go_build_flags,$(GOOS)_$(GOARCH_NATIVE)) \
		$(GO_TEST_FLAGS) \
		$(GO_TEST_PACKAGES) \
		$(GO_TEST_BINARY_FLAGS) \
		$(GO_TEST_BINARY_FLAGS_EXTRA)
gotest: build-agent-$(GOARCH_NATIVE)
test: gotest

.PHONY: clean-gotest
clean-gotest:
	$(GO) env &>/dev/null && $(GO) clean -r -testcache
	rm -f cover.txt cover.html
clean: clean-gotest

# cover.html

ifneq ($(GO_TEST_NO_COVER),1)
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

# test-coverage

ifeq ($(GOOS),linux)
.PHONY: test-coverage
test-coverage: go cover.txt
	PERCENT=$$($(GO) tool cover -func cover.txt | awk '/^total:/{print $$NF}' | tr -d % | cut -d. -f1) && \
		echo "Coverage: $$PERCENT%" && \
		if [ $$PERCENT -lt $(GO_TEST_MIN_COVERAGE) ] ; then \
			echo "Minimum coverage required: $(GO_TEST_MIN_COVERAGE)%" ; \
			exit 1 ; \
		fi
test: test-coverage
endif

endif

##
## Build
##

.PHONY: build-help
build-help:
	@echo 'build: build everything'
help: build-help

.PHONY: build-agent-%
build-agent-%: go-generate
	GOARCH=$* GOOS=linux $(GO) \
		build \
		-o host/agent/agent_linux_$* \
		$(GO_BUILD_FLAGS_COMMON) \
		$(call get_go_build_flags,linux_$*) \
		./host/agent/
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
	$(GO) \
		build \
		-o resonance.$(GOOS).$(GOARCH) \
		$(GO_BUILD_FLAGS_COMMON) \
		$(call get_go_build_flags,$(GOOS)_$(GOARCH_NATIVE)) \
		.

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
		STATICCHECK_CACHE=$(STATICCHECK_CACHE) \
		bash --rcfile .bashrc

endif