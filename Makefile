help:

##
## Variables
##

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

# system

SHELL_UNAME_S := uname -s
UNAME_S := $(shell $(SHELL_UNAME_S))
ifneq ($(.SHELLSTATUS),0)
$(error $(SHELL_UNAME_S): $(UNAME_S))
endif

# machine

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

# go

GO := go

SHELL_GOPATH := go env GOPATH
GOPATH := $(shell $(SHELL_GOPATH))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GOPATH): $(GOPATH))
endif

GOBIN := $(GOPATH)/bin

SHELL_GOOS := go env GOOS
GOOS := $(shell $(SHELL_GOOS))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GOOS): $(GOOS))
endif

GOOS_BUILD ?= $(GOOS)

SHELL_GOARCH := go env GOARCH
GOARCH := $(shell $(SHELL_GOARCH))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GOARCH): $(GOARCH))
endif

GOARCH_BUILD ?= $(GOARCH)

SHELL_GOARCH_NATIVE := go env GOARCH
GOARCH_NATIVE := $(shell $(SHELL_GOARCH_NATIVE))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GOARCH_NATIVE): $(GOARCH_NATIVE))
endif

# sources

SHELL_GO_MODULE := cat go.mod | awk '/^module /{print $$2}'
export GO_MODULE := $(shell $(SHELL_GO_MODULE))
ifneq ($(.SHELLSTATUS),0)
  $(error $(SHELL_GO_MODULE): $(GO_MODULE))
endif

GO_SOURCE_FILES := $$(find $$PWD -name \*.go ! -path $$PWD'/.home/*')

# goimports

GOIMPORTS := $(GO) tool goimports
GOIMPORTS_LOCAL := $(GO_MODULE)

# govulncheck

GOVULNCHECK := $(GO) tool govulncheck
LINT_GOVULNCHECK_DISABLE :=

# staticcheck

STATICCHECK := $(GO) tool staticcheck
export STATICCHECK_CACHE := $(CACHE_PATH)/staticcheck

# misspell

MISSPELL := $(GO) tool misspell

# gocyclo

GOCYCLO_IGNORE_REGEX := '.*\.pb\.go'
GOCYCLO := $(GO) tool gocyclo
GOCYCLO_OVER := 15

# ineffassign

INEFFASSIGN := $(GO) tool ineffassign

# go test

GO_TEST := $(GO) tool gotest

GO_TEST_FLAGS :=

GO_TEST_BUILD_FLAGS :=
ifneq ($(GO_TEST_BUILD_FLAGS_NO_RACE),1)
GO_TEST_BUILD_FLAGS := -race $(GO_TEST_BUILD_FLAGS)
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

GCOV2LCOV := $(GO) tool gcov2lcov

GO_TEST_MIN_COVERAGE := 50

# protoc

SHELL_PROTOC_VERSION := cat .protoc_version
PROTOC_VERSION := $(shell $(SHELL_PROTOC_VERSION))
ifneq ($(.SHELLSTATUS),0)
$(error $(SHELL_PROTOC_VERSION): $(PROTOC_VERSION))
endif

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

PROTOC_PREFIX := $(CACHE_PATH)/protoc
PROTOC_BIN_PATH := ${PROTOC_PREFIX}/$(PROTOC_VERSION)/$(PROTOC_OS)-$(PROTOC_ARCH)
PATH := $(PROTOC_BIN_PATH):$(PATH)

PROTOC := $(PROTOC_BIN_PATH)/protoc
PROTOC_PROTO_PATH := ./host/agent_server/proto

# protolint

PROTOLINT := $(GO) tool protolint
PROTOLINT_ARGS :=

# go build

GO_BUILD_FLAGS := -trimpath -ldflags "-s -w"

# osusergo have Lookup and LookupGroup to use pure Go implementation to enable
# management of local users
GO_BUILD_FLAGS_COMMON := -tags osusergo

GO_BUILD_AGENT_GOARCHS_ALL := 386 amd64 arm arm64

ifneq ($(GO_BUILD_AGENT_NATIVE_ONLY),1)
GO_BUILD_AGENT_GOARCHS := $(GO_BUILD_AGENT_GOARCHS_ALL)
else
GO_BUILD_AGENT_GOARCHS := $(GOARCH_NATIVE)
endif

GO_BUILD_MAX_AGENT_SIZE := 4300000

# rrb

RRB := $(GO) tool rrb
RRB_DEBOUNCE ?= 500ms
RRB_IGNORE_PATTERN ?= 'host/agent_server_linux_*_gz.go' 'host/agent_server/proto/*.pb.go'
RRB_LOG_LEVEL ?= info
RRB_PATTERN ?= '**/*.go' '**/*.proto' Makefile
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
## Tools
##

# go

.PHONY: go-update
go-update:
	set -e
	set -o pipefail
	$(GO) mod edit -go $$(curl -s https://go.dev/VERSION?m=text | head -n 1 | cut -c 3-)
update-deps: go-update

# FIXME go-clean

# protoc

.PHONY: update-protoc
update-protoc:
	set -e
	V="$$(curl -fv https://github.com/protocolbuffers/protobuf/releases/latest/ 2>&1 | \
    	grep -Ei '^< location: ' | \
    	tr / \\n | \
    	tail -n 1 | \
    	cut -c 2- | \
    	tr -d '\r')"
	echo "$$V" > .protoc_version
update-deps: update-protoc

.PHONY: clean-install-protoc
clean-install-protoc:
	PROTOC_PREFIX
	rm -rf $(PROTOC_PREFIX)
	rm -f $(CACHE_PATH)/protoc.zip
clean: clean-install-protoc

# protoc-gen-go-grpc

.PHONY: install-protoc-gen-go-grpc
install-protoc-gen-go-grpc: install-go
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc
install-tools: install-protoc-gen-go-grpc

.PHONY: clean-install-protoc-gen-go-grpc
clean-install-protoc-gen-go-grpc:
	rm -f $(GOTOOLDIR)/protoc-gen-go-grpc
clean: clean-install-protoc-gen-go-grpc

# protoc-gen-go

.PHONY: install-protoc-gen-go
install-protoc-gen-go:
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go
install-tools: install-protoc-gen-go

.PHONY: clean-install-protoc-gen-go
clean-install-protoc-gen-go:
	rm -f $(GOBIN)/protoc-gen-go
clean: clean-install-protoc-gen-go

# protoc-gen-go-grpc

.PHONY: install-protoc-gen-go-grpc
install-protoc-gen-go-grpc:
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc
install-tools: install-protoc-gen-go-grpc

.PHONY: clean-install-protoc-gen-go-grpc
clean-install-protoc-gen-go-grpc:
	rm -f $(GOBIN)/protoc-gen-go-grpc
clean: clean-install-protoc-gen-go-grpc

##
## Generate
##

# protoc

.PHONY: gen-protofiles
gen-protofiles: protolint install-protoc-gen-go install-protoc-gen-go-grpc
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

# go generate

.PHONY: go-generate
go-generate:
	$(GO) generate ./...

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
protolint:
	$(PROTOLINT) lint $(PROTOLINT_ARGS) .
lint: protolint

# go mod tidy

.PHONY: go-mod-tidy
go-mod-tidy: go-generate gen-protofiles
	$(GO) mod tidy
lint: go-mod-tidy

# goimports

.PHONY: goimports
goimports: go-mod-tidy
	$(GOIMPORTS) -w -local $(GOIMPORTS_LOCAL) $(GO_SOURCE_FILES)
lint: goimports

# govulncheck

ifneq ($(LINT_GOVULNCHECK_DISABLE),1)
.PHONY: govulncheck
govulncheck: go-generate go-mod-tidy
	$(GOVULNCHECK) $(GO_MODULE)/...
lint: govulncheck
endif

# staticcheck

.PHONY: staticcheck
staticcheck: go-mod-tidy go-generate goimports
	$(STATICCHECK) $(GO_MODULE)/...
lint: staticcheck

.PHONY: clean-staticcheck
clean-staticcheck:
	rm -rf $(STATICCHECK_CACHE)
clean: clean-staticcheck

# misspell

.PHONY: misspell
misspell: go-mod-tidy go-generate
	$(MISSPELL) -error $(GO_SOURCE_FILES)
lint: misspell

# gocyclo

.PHONY: gocyclo
gocyclo: go-generate go-mod-tidy
	$(GOCYCLO) -over $(GOCYCLO_OVER) -avg -ignore $(GOCYCLO_IGNORE_REGEX) .

lint: gocyclo

# ineffassign

.PHONY: ineffassign
ineffassign: go-generate go-mod-tidy
	$(INEFFASSIGN) ./...

lint: ineffassign

# go vet

.PHONY: go-vet
go-vet: go-mod-tidy go-generate
	$(GO) vet ./...
lint: go-vet

# go get -u

.PHONY: go-get-u-t
go-get-u-t: go-mod-tidy
	$(GO) get -u ./...
update-deps: go-get-u-t

# shellcheck

.PHONY: shellcheck
shellcheck:
	set -e
	shellcheck dev.sh
	shellcheck \
        .bashrc \
        .bashrc.vars \
        .profile
lint: shellcheck

# shfmt
.PHONY: shfmt
shfmt:
	shfmt --write --simplify --language-dialect bash --indent 4 \
		.bashrc \
		.bashrc.vars \
		.profile \
		dev.sh
lint: shfmt

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
gotest: go-generate gen-protofiles
	$(GO_TEST) \
		$(GO_BUILD_FLAGS_COMMON) \
		$(GO_TEST_BUILD_FLAGS) \
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
cover.html: gotest
	$(GO) tool cover -html cover.txt -o cover.html
test: cover.html

.PHONY: clean-cover.html
clean-cover.html:
	rm -f cover.html
clean: clean-cover.html

# cover.lcov

.PHONY: cover.lcov
cover.lcov: gotest
	$(GCOV2LCOV) -infile cover.txt -outfile cover.lcov
test: cover.lcov

.PHONY: clean-cover.lcov
clean-cover.lcov:
	rm -f cover.lcov
clean: clean-cover.lcov

# test-coverage

.PHONY: test-coverage
test-coverage: cover.txt
	PERCENT=$$($(GO) tool cover -func cover.txt | awk '/^total:/{print $$NF}' | tr -d % | cut -d. -f1) && \
		echo "Coverage: $$PERCENT%" && \
		if [ $$PERCENT -lt $(GO_TEST_MIN_COVERAGE) ] ; then \
			echo "Minimum coverage required: $(GO_TEST_MIN_COVERAGE)%" ; \
			exit 1 ; \
		fi
test: test-coverage

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
	@echo '  set GOOS_BUILD to cross compile'
	@echo '  set GOARCH_BUILD to cross compile'
help: help-build

# agent

.PHONY: build-agent-%
build-agent-%: go-generate gen-protofiles
	set -e
	GOARCH=$* GOOS=linux $(GO) \
		build \
		-o host/agent_server/agent_server_linux_$* \
		$(GO_BUILD_FLAGS_COMMON) \
		$(GO_BUILD_FLAGS) \
		./host/agent_server/
	gzip < host/agent_server/agent_server_linux_$* > host/agent_server/agent_server_linux_$*.gz
	if ! size=$$(stat -f %z host/agent_server/agent_server_linux_$*.gz  2>/dev/null) ; then size=$$(stat --printf=%s host/agent_server/agent_server_linux_$*.gz) ; fi
	[ "$$size" -gt $(GO_BUILD_MAX_AGENT_SIZE) ] && { echo "Compressed agent size exceeds $(GO_BUILD_MAX_AGENT_SIZE) bytes" ; exit 1 ; }
	cat << EOF > host/agent_server_linux_$*_gz.go
	package host
	import _ "embed"
	//go:embed agent_server/agent_server_linux_$*.gz
	var agent_server_linux_$* []byte
	func init() {
		AgentBinGz["linux.$*"] = agent_server_linux_$*
	}
	EOF
build-agent: $(foreach GOARCH,$(GO_BUILD_AGENT_GOARCHS),build-agent-$(GOARCH))

.PHONY: clean-build-agent-%
clean-build-agent-%:
	rm -f host/agent_server/agent_server_linux_$*
	rm -f host/agent_server/agent_server_linux_$*.gz
	rm -f host/agent_server_linux_$*_gz.go
clean-agent: $(foreach GOARCH,$(GO_BUILD_AGENT_GOARCHS_ALL),clean-build-agent-$(GOARCH))

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
build: go-generate build-agent gen-protofiles
	GOOS=$(GOOS_BUILD) \
	GOARCH=$(GOARCH_BUILD) \
        $(GO) \
            build \
            -o resonance.$(GOOS_BUILD).$(GOARCH_BUILD) \
            $(GO_BUILD_FLAGS_COMMON) \
            $(GO_BUILD_FLAGS) \
            ./cmd/

.PHONY: clean-build
clean-build:
	$(GO) env &>/dev/null && $(GO) clean -r -cache -modcache
	rm -f .version
	rm -f .git-toplevel
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

.PHONY: help-rrb
help-rrb:
	@echo 'rrb: rerun build automatically on file changes (Linux hosts only)'
	@echo ' use RRB_DEBOUNCE to set debounce (default: $(RRB_DEBOUNCE))'
	@echo ' use RRB_IGNORE_PATTERN to set ignore pattern (default: $(RRB_IGNORE_PATTERN))'
	@echo ' use RRB_LOG_LEVEL to set log level (default: $(RRB_LOG_LEVEL))'
	@echo ' use RRB_PATTERN to set the pattern (default: $(RRB_PATTERN))'
	@echo ' use RRB_MAKE_TARGET to set the make target (default: $(RRB_MAKE_TARGET))'
	@echo ' use RRB_EXTRA_CMD to set a command to run after the build is successful (default: $(RRB_EXTRA_CMD))'
	@echo 'rrb-dev: similar to rrb, but with RRB_MAKE_TARGET=ci-dev'
help: help-rrb

.PHONY: rrb
rrb:
	$(RRB) \
		--debounce $(RRB_DEBOUNCE) \
		$(foreach pattern,$(RRB_IGNORE_PATTERN),--ignore-pattern $(pattern)) \
		--log-level $(RRB_LOG_LEVEL) \
		$(foreach pattern,$(RRB_PATTERN),--pattern $(pattern)) \
		-- \
		sh -c "$(MAKE) $(MFLAGS) $(RRB_MAKE_TARGET) && $(RRB_EXTRA_CMD)"

.PHONY: rrb-dev
rrb-dev:
	$(MAKE) $(MFLAGS) MAKELEVEL= \
		rrb \
			RRB_MAKE_TARGET=ci-dev
