[![Latest Release](https://img.shields.io/github/v/release/fornellas/resonance)](https://github.com/fornellas/resonance/releases) [![Push](https://github.com/fornellas/resonance/actions/workflows/push.yaml/badge.svg)](https://github.com/fornellas/resonance/actions/workflows/push.yaml) [![Update Deps](https://github.com/fornellas/resonance/actions/workflows/update_deps.yaml/badge.svg?branch=master)](https://github.com/fornellas/resonance/actions/workflows/update_deps.yaml) [![Go Report Card](https://goreportcard.com/badge/github.com/fornellas/resonance)](https://goreportcard.com/report/github.com/fornellas/resonance) [![Coverage Status](https://coveralls.io/repos/github/fornellas/resonance/badge.svg?branch=master)](https://coveralls.io/github/fornellas/resonance?branch=master) [![Go Reference](https://pkg.go.dev/badge/github.com/fornellas/resonance.svg)](https://pkg.go.dev/github.com/fornellas/resonance) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0) [![Buy me a beer: donate](https://img.shields.io/badge/Donate-Buy%20me%20a%20beer-yellow)](https://www.paypal.com/donate?hosted_button_id=AX26JVRT2GS2Q)

**Status**: under active development, targetting a [v1 milestone](https://github.com/fornellas/resonance/milestone/1).

**🙏Help wanted🙏**: There are [plenty of issues to work on](https://github.com/fornellas/resonance/issues), of all sizes and difficulty. Getting a dev environment setup [takes less than 5 minutes](#development), so you can get your first PR easily!

# resonance

A configuration management tool with novel features to Ansible, Chef or Puppet:

- **Stateful**: Persistent host state enables deletion of old resources, rollback to previous state (on failures) and detection of external changes.
- **Transactional changes**: things such as APT packages are done "all together or nothing" (single `apt` call), eliminating isses with conflicting packages.
- **Painless refresh**: no need to manually tell "please restart the service after changes" as these are implicitly declared so things "just work".
- **Painless dependencies**: declared order is always respected.
- **Speed**: read-only checks and possible changes happen concurrently; a lightweight agent is used so things fly even via SSH.

## Install

Pick the [latest release](https://github.com/fornellas/resonance/releases) with:

```bash
GOARCH=$(case $(uname -m) in i[23456]86) echo 386;; x86_64) echo amd64;; armv6l|armv7l) echo arm;; aarch64) echo arm64;; *) echo Unknown machine $(uname -m) 1>&2 ; exit 1 ;; esac) && wget -O- https://github.com/fornellas/resonance/releases/latest/download/resonance.$(uname -s | tr A-Z a-z).$GOARCH.gz | gunzip > resonance && chmod 755 resonance
./resonance --help
```

## Development

Getting started is _super easy_: just run these commands under Linux or MacOS:

```bash
git clone git@github.com:fornellas/resonance.git
cd resonance/
./make.sh ci # runs linters, tests and build
```

That's it! The local build happens inside a [Docker](https://www.docker.com/), so it is easily reproducible on any machine.

If you're running an old Arm Mac (eg: M1), Docker is _very_ slow, you should consider a no container build (see below).

### Shell

You can start a development shell, which gives you access to the whole build environemnt, including all build tools with the correct versions. Eg:

```shell
./make.sh shell
make ci
go get -u
```

### No container

You may also run the _unofficial_[^1] build, without a container. This requires Installing [GNU Make](https://www.gnu.org/software/make/):

- Ubuntu: `apt install make`.
- MacOS: `brew install make`[^2]

Then:

```shell
make ci # on MacOS, gmake ci
```

Or start a bash[^3] development shell:

```shell
make shell
```

[^1]: unnofficial in the sense that it is _not_ the same as it happens on CI, as unforeseen environment differences may impact the signal. Additionally, a lot of code / tests is Linux only (eg: `*_linux.go`), so none of this signal will be available on MacOS.

[^2]: note that brew install the command as `gmake`. Apple's ancient `make` shipped with MacOS will NOT work.

[^3]: bash must be installed separately, eg, under MacOS, `brew install bash`.

### Using gopls, the Go Language Server

Your editor may already support using [gopls](https://github.com/golang/tools/tree/master/gopls), and you should follow its documentation on how to set it up. This may require having the correct go (and gopls) versions installed and available for your editor. This can be annoying and error prone.

In this scenario, you should leverage the "no container" option:

```shell
make shell
make ci # installs all development tools
```

And then start you code editor from the development shell (eg: for Sublime, do `subl .`). This enables the code editor to have access to all the _exact_ versions of tools required.

### Faster builds

The default build with `ci` reproduces the official build, but this may be too slow during development. You can use one of the `*-dev` targets to do a "dev build": bulid is a lot faster, at the expense of minimal signal loss:

```shell
./make.sh ci-dev # or "make ci-dev"
```

### Enable debugging symbols

By default, both the embedded agents and final binaries are built without debugging symbols by use of extra `go build` flags.

During development, it can be useful to have debugging symbols enabled. This can be accomplished by:

```shell
make ci GO_BUILD_FLAGS='' GO_BUILD_MAX_AGENT_SIZE=10000000
```

PS: tests _do_ run with debugging symbols.

### Automatic builds on Linux

The build system is integrated with [rrb](https://github.com/fornellas/rrb), which enables the build to run automatically as you edit the files.

First, start rrb:

```shell
./make.sh rrb # or "make rrb"
```

then just edit the files with your preferred editor. As soon as you save any file, the build will be automatically run, interrupting any ongoing build, so you always get a fresh signal.

There's also a `rrb-dev` target, which yields faster builds during development.
