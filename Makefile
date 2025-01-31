help:

##
## Make
##

SHELL := /bin/bash
.ONESHELL:

MAKE_MAJOR_VERSION := $(word 1, $(subst ., , $(MAKE_VERSION)))
MAKE_REQUIRED_MAJOR_VERSION := 4
MAKE_BAD_VERSION := $(shell [ $(MAKE_MAJOR_VERSION) -lt $(MAKE_REQUIRED_MAJOR_VERSION) ] && echo true)
ifeq ($(MAKE_BAD_VERSION),true)
  $(error Make version is below $(MAKE_REQUIRED_MAJOR_VERSION), please update it.)
endif

##
## uname
##

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

##
## Cache
##

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

##
## Go
##

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

export GOPATH := $(CACHE_PATH)/GOPATH
PATH := $(GOPATH)/bin:$(PATH)

export GOCACHE := $(CACHE_PATH)/GOCACHE

export GOMODCACHE := $(CACHE_PATH)/GOMODCACHE

##
## Go source
##

SHELL_GO_MODULE := cat go.mod | awk '/^module /{print $$2}'
export GO_MODULE := $(shell $(SHELL_GO_MODULE))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GO_MODULE): $(GO_MODULE))
endif

GO_SOURCE_FILES := $$(find $$PWD -name \*.go ! -path '$(CACHE_PATH)/*')

##
## goimports
##

GOIMPORTS := $(GO) run golang.org/x/tools/cmd/goimports
GOIMPORTS_LOCAL := $(GO_MODULE)

##
## govulncheck
##

GOVULNCHECK := $(GO) run golang.org/x/vuln/cmd/govulncheck
LINT_GOVULNCHECK_DISABLE :=

##
## staticcheck
##

STATICCHECK := $(GO) run honnef.co/go/tools/cmd/staticcheck
export STATICCHECK_CACHE := $(CACHE_PATH)/staticcheck

##
## misspell
##

MISSPELL := $(GO) run github.com/client9/misspell/cmd/misspell

##
## gocyclo
##

GOCYCLO_IGNORE_REGEX := '.*\.pb\.go'
GOCYCLO := $(GO) run github.com/fzipp/gocyclo/cmd/gocyclo
GOCYCLO_OVER := 15

##
## ineffassign
##

INEFFASSIGN := $(GO) run github.com/gordonklaus/ineffassign

##
## go test
##

GO_TEST := $(GO) run github.com/rakyll/gotest

GO_TEST_FLAGS :=

define go_test_build_flags
$(value GO_TEST_BUILD_FLAGS_$(1))
endef

GO_TEST_BUILD_FLAGS :=
# https://go.dev/doc/articles/race_detector#Requirements
ifneq ($(GO_TEST_BUILD_FLAGS_NO_RACE),1)
ifeq ($(GOOS)/$(GOARCH),linux/amd64)
GO_TEST_BUILD_FLAGS_linux_amd64 := -race $(GO_TEST_BUILD_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),linux/ppc64le)
GO_TEST_BUILD_FLAGS_linux_ppc64le := -race $(GO_TEST_BUILD_FLAGS)
endif
# https://github.com/golang/go/issues/29948
# ifeq ($(GOOS)/$(GOARCH),linux/arm64)
#_LINUX_ARM64 GO_TEST_BUILD_FLAGS := -race $(GO_TEST_BUILD_FLAGS)
# endif
ifeq ($(GOOS)/$(GOARCH),freebsd/amd64)
GO_TEST_BUILD_FLAGS_freebsd_amd64 := -race $(GO_TEST_BUILD_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),netbsd/amd64)
GO_TEST_BUILD_FLAGS_netbsd_amd64 := -race $(GO_TEST_BUILD_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),darwin/amd64)
GO_TEST_BUILD_FLAGS_darwin_amd64 := -race $(GO_TEST_BUILD_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),darwin/arm64)
GO_TEST_BUILD_FLAGS_darwin_arm64 := -race $(GO_TEST_BUILD_FLAGS)
endif
ifeq ($(GOOS)/$(GOARCH),windows/amd64)
GO_TEST_BUILD_FLAGS_windows_amd64 := -race $(GO_TEST_BUILD_FLAGS)
endif
endif

GO_TEST_PACKAGES_DEFAULT := $(GO_MODULE)/...
GO_TEST_PACKAGES := $(GO_TEST_PACKAGES_DEFAULT)

GO_TEST_BINARY_FLAGS :=
ifneq ($(GO_TEST_NO_COVER),1)
GO_TEST_BINARY_FLAGS := -coverprofile cover.txt -coverpkg $(GO_TEST_PACKAGES) $(GO_TEST_BINARY_FLAGS)
endif
GO_TEST_BINARY_FLAGS := -count=1 $(GO_TEST_BINARY_FLAGS)
GO_TEST_BINARY_FLAGS := -failfast $(GO_TEST_BINARY_FLAGS)

GO_TEST_BINARY_FLAGS_EXTRA :=

GCOV2LCOV := $(GO) run github.com/jandelgado/gcov2lcov

GO_TEST_MIN_COVERAGE := 50

##
## protobuf
##

PROTOC_VERSION := 29.2

ifeq ($(UNAME_S),Linux)
PROTOC_OS := linux
else
ifeq ($(UNAME_S),Darwin)
PROTOC_OS := osx
else
$(error Unsupported system: $(UNAME_S))
endif
endif

SHELL_PROTOC_ARCH := case $(UNAME_M) in i[23456]86) echo x86_32;; x86_64) echo x86_64;; aarch64|arm64) echo aarch_64;; *) echo Unknown machine $(UNAME_M) 1>&2 ; exit 1 ;; esac
PROTOC_ARCH ?= $(shell $(SHELL_PROTOC_ARCH))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_PROTOC_ARCH): $(PROTOC_ARCH))
endif

PROTOC_BIN_PATH := $(CACHE_PATH)/protoc/$(PROTOC_VERSION)/$(PROTOC_OS)-$(PROTOC_ARCH)
PATH := $(PROTOC_BIN_PATH):$(PATH)

PROTOC := $(PROTOC_BIN_PATH)/protoc
PROTOC_PROTO_PATH := ./internal/host/agent_server_grpc/proto

PROTOLINT := $(GO) run github.com/yoheimuta/protolint/cmd/protolint
PROTOLINT_ARGS :=

##
## go build
##

GO_BUILD_FLAGS :=

# osusergo have Lookup and LookupGroup to use pure Go implementation to enable
# management of local users
GO_BUILD_FLAGS_COMMON := -tags osusergo

ifneq ($(GO_BUILD_AGENT_NATIVE_ONLY),1)
GO_BUILD_AGENT_GOARCHS := 386 amd64 arm arm64
else
GO_BUILD_AGENT_GOARCHS := $(GOARCH)
endif

GO_BUILD_MAX_AGENT_SIZE := 8000000

##
## rrb
##

RRB := $(GO) run github.com/fornellas/rrb
RRB_DEBOUNCE ?= 500ms
RRB_IGNORE_PATTERN ?= 'internal/host/agent_server_http_linux_*_gz.go,internal/host/agent_server_grpc_linux_*_gz.go,internal/host/agent_server_grpc/proto/*.pb.go'
RRB_LOG_LEVEL ?= info
RRB_PATTERN ?= '**/*.go,**/*.proto,Makefile'
RRB_MAKE_TARGET ?= ci
RRB_EXTRA_CMD ?= true

##
## Help
##

.PHONY: help
help:

##
## Clean
##

.PHONY: help-clean
help-clean:
	@echo 'clean: clean all files'
help: help-clean

.PHONY: clean
clean:

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
## Protobuf
##

.PHONY: install-protoc
install-protoc:
	set -e
	if [ -x $(PROTOC_BIN_PATH)/protoc ] ; then exit ; fi
	mkdir -p $(PROTOC_BIN_PATH)
	curl -sSfL https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-$(PROTOC_OS)-$(PROTOC_ARCH).zip > $(CACHE_PATH)/protoc.zip
	unzip -p $(CACHE_PATH)/protoc.zip bin/protoc > $(PROTOC_BIN_PATH)/protoc.tmp
	chmod +x $(PROTOC_BIN_PATH)/protoc.tmp
	mv $(PROTOC_BIN_PATH)/protoc.tmp $(PROTOC_BIN_PATH)/protoc

.PHONY: clean-install-protoc
clean-install-protoc:
	rm -f $(PROTOC_BIN_PATH)/protoc.tmp
	rm -f $(PROTOC_BIN_PATH)/protoc
clean: clean-install-protoc

.PHONY: install-protoc-gen-go-grpc
install-protoc-gen-go-grpc: go
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc

.PHONY: install-protoc-gen-go
install-protoc-gen-go: go
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go

.PHONY: gen-protofiles
gen-protofiles: install-protoc install-protoc-gen-go install-protoc-gen-go-grpc protolint
	$(PROTOC) \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		$(PROTOC_PROTO_PATH)/*.proto

.PHONY: clean-gen-protofiles
clean-gen-protofiles:
	rm -f $(PROTOC_PROTO_PATH)/*.pb.go
clean: clean-gen-protofiles

##
## Lint
##

# lint

.PHONY: help-lint
help-lint:
	@echo 'lint: runs all linters'
	@echo '  use LINT_GOVULNCHECK_DISABLE=1 to disable govulncheck (faster)'
	@echo '  use PROTOLINT_ARGS to set `protolint lint` arguments (eg: -fix)'
help: help-lint

.PHONY: lint
lint:

# protolint

.PHONY: protolint
protolint: go
	$(PROTOLINT) lint $(PROTOLINT_ARGS) .
lint: protolint

# Generate

.PHONY: go-generate
go-generate: go
	$(GO) generate ./...

# go mod tidy

.PHONY: go-mod-tidy
go-mod-tidy: go go-generate gen-protofiles
	$(GO) mod tidy
lint: go-mod-tidy

# goimports

.PHONY: goimports
goimports: go go-mod-tidy
	$(GOIMPORTS) -w -local $(GOIMPORTS_LOCAL) $(GO_SOURCE_FILES)
lint: goimports

# govulncheck

ifneq ($(LINT_GOVULNCHECK_DISABLE),1)
.PHONY: govulncheck
govulncheck: go-generate go go-mod-tidy
	$(GOVULNCHECK) $(GO_MODULE)/...
lint: govulncheck
endif

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
	$(MISSPELL) -error $(GO_SOURCE_FILES)
lint: misspell

# gocyclo

.PHONY: gocyclo
gocyclo: go go-generate go-mod-tidy
	$(GOCYCLO) -over $(GOCYCLO_OVER) -avg -ignore $(GOCYCLO_IGNORE_REGEX) .

lint: gocyclo

# ineffassign

.PHONY: ineffassign
ineffassign: go go-generate go-mod-tidy
	$(INEFFASSIGN) ./...

lint: ineffassign

# go vet

.PHONY: go-vet
go-vet: go go-mod-tidy go-generate
	$(GO) vet ./...
lint: go-vet

# go-update
.PHONY: go-update
go-update: go
	set -e
	set -o pipefail
	$(GO) mod edit -go $$(curl -s https://go.dev/VERSION?m=text | head -n 1 | cut -c 3-)
update-deps: go-update

# go get -u

.PHONY: go-get-u-t
go-get-u-t: go go-mod-tidy
	$(GO) get -u ./...
update-deps: go-get-u-t

##
## Test
##

# test

.PHONY: help-test
help-test:
	@echo 'test: runs all tests:'
	@echo '  use GO_TEST_BUILD_FLAGS to set test build flags (see `go test build`)'
	@echo '  use GO_TEST_FLAGS to set test flags (see `go help test`)'
	@echo '  use GO_TEST_PACKAGES to set packages to test (default: $(GO_TEST_PACKAGES_DEFAULT))'
	@echo '  use GO_TEST_BINARY_FLAGS_EXTRA to pass extra flags to the test binary (see `go help testflag`)'
	@echo '  use GO_TEST_NO_COVER=1 to disable code coverage (faster)'
	@echo '  use GO_TEST_BUILD_FLAGS_NO_RACE=1 to disable -race build flag (faster)'
help: help-test

.PHONY: test

# gotest

.PHONY: gotest
gotest: go go-generate gen-protofiles
	$(GO_TEST) \
		$(GO_BUILD_FLAGS_COMMON) \
		$(call go_test_build_flags,$(GOOS)_$(GOARCH_NATIVE)) \
		$(GO_TEST_FLAGS) \
		$(GO_TEST_PACKAGES) \
		$(GO_TEST_BINARY_FLAGS) \
		$(GO_TEST_BINARY_FLAGS_EXTRA)
gotest: build-agent-native
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

# help

.PHONY: help-build
help-build:
	@echo 'build: build everything'
	@echo '  use GO_BUILD_FLAGS to add extra build flags (see `go help build`)'
	@echo '  use GO_BUILD_AGENT_NATIVE_ONLY=1 to only build agent to native arch (faster)'
help: help-build

# agent http

.PHONY: build-agent-http-%
build-agent-http-%: go-generate
	set -e
	GOARCH=$* GOOS=linux $(GO) \
		build \
		-o internal/host/agent_server_http/agent_server_http_linux_$* \
		$(GO_BUILD_FLAGS_COMMON) \
		$(GO_BUILD_FLAGS) \
		./internal/host/agent_server_http/
	gzip < internal/host/agent_server_http/agent_server_http_linux_$* > internal/host/agent_server_http/agent_server_http_linux_$*.gz
	if ! size=$$(stat -f %z internal/host/agent_server_http/agent_server_http_linux_$*.gz  2>/dev/null) ; then size=$$(stat --printf=%s internal/host/agent_server_http/agent_server_http_linux_$*.gz) ; fi
	[ "$$size" -gt $(GO_BUILD_MAX_AGENT_SIZE) ] && { echo "Compressed agent size exceeds $(GO_BUILD_MAX_AGENT_SIZE) bytes" ; exit 1 ; }
	cat << EOF > internal/host/agent_server_http_linux_$*_gz.go
	package host
	import _ "embed"
	//go:embed agent_server_http/agent_server_http_linux_$*.gz
	var agent_server_http_linux_$* []byte
	func init() {
		AgentHttpBinGz["linux.$*"] = agent_server_http_linux_$*
	}
	EOF
build-agent: $(foreach GOARCH,$(GO_BUILD_AGENT_GOARCHS),build-agent-http-$(GOARCH))
build-agent-native: build-agent-http-$(GOARCH_NATIVE)

.PHONY: clean-build-agent-http-%
clean-build-agent-http-%:
	rm -f internal/host/agent_server_http/agent_server_http_linux_$*
	rm -f internal/host/agent_server_http/agent_server_http_linux_$*.gz
	rm -f internal/host/agent_server_http_linux_$*_gz.go
clean-agent: $(foreach GOARCH,$(GO_BUILD_AGENT_GOARCHS),clean-build-agent-http-$(GOARCH))

# agent grpc

.PHONY: build-agent-grpc-%
build-agent-grpc-%: go-generate gen-protofiles
	set -e
	GOARCH=$* GOOS=linux $(GO) \
		build \
		-o internal/host/agent_server_grpc/agent_server_grpc_linux_$* \
		$(GO_BUILD_FLAGS_COMMON) \
		$(GO_BUILD_FLAGS) \
		./internal/host/agent_server_grpc/
	gzip < internal/host/agent_server_grpc/agent_server_grpc_linux_$* > internal/host/agent_server_grpc/agent_server_grpc_linux_$*.gz
	if ! size=$$(stat -f %z internal/host/agent_server_grpc/agent_server_grpc_linux_$*.gz  2>/dev/null) ; then size=$$(stat --printf=%s internal/host/agent_server_grpc/agent_server_grpc_linux_$*.gz) ; fi
	[ "$$size" -gt $(GO_BUILD_MAX_AGENT_SIZE) ] && { echo "Compressed agent size exceeds $(GO_BUILD_MAX_AGENT_SIZE) bytes" ; exit 1 ; }
	cat << EOF > internal/host/agent_server_grpc_linux_$*_gz.go
	package host
	import _ "embed"
	//go:embed agent_server_grpc/agent_server_grpc_linux_$*.gz
	var agent_server_grpc_linux_$* []byte
	func init() {
		AgentGrpcBinGz["linux.$*"] = agent_server_grpc_linux_$*
	}
	EOF
build-agent: $(foreach GOARCH,$(GO_BUILD_AGENT_GOARCHS),build-agent-grpc-$(GOARCH))
build-agent-native: build-agent-grpc-$(GOARCH_NATIVE)

.PHONY: clean-build-agent-grpc-%
clean-build-agent-grpc-%:
	rm -f internal/host/agent_server_grpc/agent_server_grpc_linux_$*
	rm -f internal/host/agent_server_grpc/agent_server_grpc_linux_$*.gz
	rm -f internal/host/agent_server_grpc_linux_$*_gz.go
clean-agent: $(foreach GOARCH,$(GO_BUILD_AGENT_GOARCHS),clean-build-agent-grpc-$(GOARCH))

# clean agent

clean: clean-agent
build: clean-agent
go-generate: clean-agent
goimports: clean-agent
go-mod-tidy: clean-agent
go-get-u-t: clean-agent
staticcheck: clean-agent
misspell: clean-agent
gocyclo: clean-agent
go-vet: clean-agent

# build

.PHONY: build
build: go go-generate build-agent gen-protofiles
	$(GO) \
		build \
		-o resonance.$(GOOS).$(GOARCH) \
		$(GO_BUILD_FLAGS_COMMON) \
		$(GO_BUILD_FLAGS) \
		./cmd/

.PHONY: clean-build
clean-build:
	$(GO) env &>/dev/null && $(GO) clean -r -cache -modcache
	rm -f internal/.version
	rm -f internal/.git-toplevel
	rm -f resonance.*.*
clean: clean-build

##
## ci
##

.PHONY: help-ci
help-ci:
	@echo 'ci: runs the whole build'
	@echo 'ci-dev: similar to ci, but uses options that speed up the build, at the expense of minimal signal loss;'
help: help-ci

.PHONY: ci
ci: lint test build

.PHONY: ci-dev
ci-dev:
	$(MAKE) $(MFLAGS) MAKELEVEL= ci \
		LINT_GOVULNCHECK_DISABLE=1 \
		GO_TEST_NO_COVER=1 \
		GO_TEST_BUILD_FLAGS_NO_RACE=1 \
		GO_BUILD_AGENT_NATIVE_ONLY=1

##
## update
##

.PHONY: help-update-deps
help-update-deps:
	@echo 'update-deps: Update all dependencies'
help: help-update-deps

.PHONY: update-deps
update-deps:

##
## rrb
##

ifeq ($(GOOS),linux)

.PHONY: help-rrb
help-rrb:
	@echo 'rrb: rerun build automatically on file changes'
	@echo ' use RRB_DEBOUNCE to set debounce (default: $(RRB_DEBOUNCE))'
	@echo ' use RRB_IGNORE_PATTERN to set ignore pattern (default: $(RRB_IGNORE_PATTERN))'
	@echo ' use RRB_LOG_LEVEL to set log level (default: $(RRB_LOG_LEVEL))'
	@echo ' use RRB_PATTERN to set the pattern (default: $(RRB_PATTERN))'
	@echo ' use RRB_MAKE_TARGET to set the make target (default: $(RRB_MAKE_TARGET))'
	@echo ' use RRB_EXTRA_CMD to set a command to run after the build is successful (default: $(RRB_EXTRA_CMD))'
	@echo 'rrb-dev: similar to rrb, but with RRB_MAKE_TARGET=ci-dev'
help: help-rrb

.PHONY: rrb
rrb: go
	$(RRB) \
		--debounce $(RRB_DEBOUNCE) \
		--ignore-pattern $(RRB_IGNORE_PATTERN) \
		--log-level $(RRB_LOG_LEVEL) \
		--pattern $(RRB_PATTERN) \
		-- \
		sh -c "$(MAKE) $(MFLAGS) $(RRB_MAKE_TARGET) && $(RRB_EXTRA_CMD)"

.PHONY: rrb-dev
rrb-dev:
	$(MAKE) $(MFLAGS) MAKELEVEL= \
		rrb \
			RRB_MAKE_TARGET=ci-dev

endif

##
## shell
##

.PHONY: help-shell
help-shell:
	@echo 'shell: starts a development shell'
help: help-shell

.PHONY: shell
shell:
	@echo Make targets:
	@$(MAKE) help MAKELEVEL=
	@PATH=$(GOROOT)/bin:$(PATH) \
		GOOS=$(GOOS) \
		GOARCH=$(GOARCH) \
		GOROOT=$(GOROOT) \
		GOCACHE=$(GOCACHE) \
		GOMODCACHE=$(GOMODCACHE) \
		STATICCHECK_CACHE=$(STATICCHECK_CACHE) \
		bash --rcfile .bashrc
